package middleware

import (
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// RateLimitWithRoleExemption replaces PocketBase's default rate limit middleware
// (same ID, same priority) and adds exemptions for app admins and editors on top
// of the existing superuser exemption.
//
// Use in OnServe after unbinding apis.DefaultRateLimitMiddlewareId:
//
//	se.Router.Unbind(apis.DefaultRateLimitMiddlewareId)
//	se.Router.Bind(middleware.RateLimitWithRoleExemption())
func RateLimitWithRoleExemption() *hook.Handler[*core.RequestEvent] {
	store := &rateLimiterStore{}

	return &hook.Handler[*core.RequestEvent]{
		Id:       apis.DefaultRateLimitMiddlewareId,
		Priority: apis.DefaultRateLimitMiddlewarePriority,
		Func: func(e *core.RequestEvent) error {
			settings := e.App.Settings()
			if !settings.RateLimits.Enabled {
				return e.Next()
			}
			if e.HasSuperuserAuth() {
				return e.Next()
			}
			if e.Auth != nil {
				role := e.Auth.GetString("role")
				if role == "admin" || role == "editor" {
					return e.Next()
				}
			}

			// Determine audience for rule lookup
			var audience []string
			if e.Auth != nil {
				audience = []string{core.RateLimitRuleAudienceAll, core.RateLimitRuleAudienceAuth}
			} else {
				audience = []string{core.RateLimitRuleAudienceAll, core.RateLimitRuleAudienceGuest}
			}

			labels := []string{
				e.Request.Method + " " + e.Request.URL.Path,
				e.Request.URL.Path,
			}

			rule, ok := settings.RateLimits.FindRateLimitRule(labels, audience...)
			if !ok {
				return e.Next()
			}

			key := e.RealIP() + "|" + rule.Label + rule.Audience
			if !store.isAllowed(key, rule.MaxRequests, int64(rule.Duration)) {
				return e.TooManyRequestsError("", nil)
			}

			return e.Next()
		},
	}
}

// rateLimiterStore is a simple fixed-window rate limiter backed by sync.Map.
type rateLimiterStore struct {
	m sync.Map
}

type rateLimitEntry struct {
	mu          sync.Mutex
	maxAllowed  int
	available   int
	interval    int64
	lastConsume int64
}

func (s *rateLimiterStore) isAllowed(key string, max int, interval int64) bool {
	v, _ := s.m.LoadOrStore(key, &rateLimitEntry{
		maxAllowed: max,
		interval:   interval,
	})
	entry := v.(*rateLimitEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now().Unix()
	if now-entry.lastConsume >= entry.interval {
		entry.available = entry.maxAllowed
	}
	if entry.available > 0 {
		entry.available--
		entry.lastConsume = now
		return true
	}
	return false
}
