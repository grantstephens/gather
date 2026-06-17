package main

import (
	"html"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	dbx "github.com/pocketbase/dbx"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"gather/internal/activitypub"
	"gather/internal/cron"
	"gather/internal/hooks"
	"gather/internal/ical"
	"gather/internal/middleware"
	"gather/internal/recurrence"
	"gather/internal/rss"
	"gather/internal/seo"
	_ "gather/migrations"
)

const defaultFavicon = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" rx="6" fill="#6366f1"/><rect x="6" y="4" width="20" height="24" rx="3" fill="#fff"/><rect x="10" y="3" width="2" height="4" rx="1" fill="#6366f1"/><rect x="20" y="3" width="2" height="4" rx="1" fill="#6366f1"/><line x1="6" y1="12" x2="26" y2="12" stroke="#e0e0e0" stroke-width="1"/><circle cx="12" cy="17" r="1.5" fill="#6366f1"/><circle cx="16" cy="17" r="1.5" fill="#6366f1"/><circle cx="20" cy="17" r="1.5" fill="#6366f1"/><circle cx="12" cy="22" r="1.5" fill="#6366f1"/><circle cx="16" cy="22" r="1.5" fill="#6366f1"/></svg>`

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Enable daily backups by default if not already configured
		if se.App.Settings().Backups.Cron == "" {
			se.App.Settings().Backups.Cron = "0 3 * * *" // 3 AM daily
			se.App.Settings().Backups.CronMaxKeep = 7     // keep 7 backups
			if err := se.App.Save(se.App.Settings()); err != nil {
				log.Println("Warning: failed to configure default backup schedule:", err)
			}
		}

		// Configure rate limits (skip for superusers — PocketBase handles that automatically)
		appSettings := se.App.Settings()
		appSettings.RateLimits.Enabled = true
		appSettings.RateLimits.Rules = []core.RateLimitRule{
			// Guests: tight limits to prevent brute force / spam
			{Label: "*:auth", MaxRequests: 10, Duration: 60, Audience: core.RateLimitRuleAudienceGuest},
			{Label: "*:create", MaxRequests: 20, Duration: 60, Audience: core.RateLimitRuleAudienceGuest},
			{Label: "/api/search", MaxRequests: 60, Duration: 60, Audience: core.RateLimitRuleAudienceGuest},
			// trailing-slash prefix matches /feed/tag/<name> and /ics/tag/<name>
			{Label: "/feed/tag/", MaxRequests: 30, Duration: 60, Audience: core.RateLimitRuleAudienceGuest},
			{Label: "/ics/tag/", MaxRequests: 30, Duration: 60, Audience: core.RateLimitRuleAudienceGuest},
			// Authenticated users: very high limits — effectively no restriction
			{Label: "*:auth", MaxRequests: 1000, Duration: 60, Audience: core.RateLimitRuleAudienceAuth},
			{Label: "*:create", MaxRequests: 1000, Duration: 60, Audience: core.RateLimitRuleAudienceAuth},
			{Label: "/api/search", MaxRequests: 1000, Duration: 60, Audience: core.RateLimitRuleAudienceAuth},
			{Label: "/feed/tag/", MaxRequests: 1000, Duration: 60, Audience: core.RateLimitRuleAudienceAuth},
			{Label: "/ics/tag/", MaxRequests: 1000, Duration: 60, Audience: core.RateLimitRuleAudienceAuth},
		}
		// Note: these rules are written to the DB on every startup, so any changes
		// made via the admin dashboard will be reverted on restart.
		if err := se.App.Save(appSettings); err != nil {
			log.Println("Warning: failed to configure rate limits:", err)
		}

		// Build initial CSP from custom_head setting
		var cachedCSP atomic.Value
		if s, err := se.App.FindFirstRecordByFilter("settings", ""); err == nil {
			origins := seo.ExtractExternalOrigins(s.GetString("custom_head"))
			if o := seo.OriginFromURL(s.GetString("umami_src")); o != "" {
				origins = append(origins, o)
			}
			cachedCSP.Store(seo.BuildCSP(origins))
		} else {
			cachedCSP.Store(seo.BuildCSP(nil))
		}

		// Refresh CSP whenever settings are updated
		se.App.OnRecordAfterUpdateSuccess("settings").BindFunc(func(e *core.RecordEvent) error {
			origins := seo.ExtractExternalOrigins(e.Record.GetString("custom_head"))
			if o := seo.OriginFromURL(e.Record.GetString("umami_src")); o != "" {
				origins = append(origins, o)
			}
			cachedCSP.Store(seo.BuildCSP(origins))
			return e.Next()
		})

		// Security headers middleware
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			path := e.Request.URL.Path
			h := e.Response.Header()

			// Apply to all routes
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Permitted-Cross-Domain-Policies", "none")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Cross-Origin-Resource-Policy", "same-origin")
			h.Set("Permissions-Policy", "accelerometer=(), autoplay=(), camera=(), display-capture=(), encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), screen-wake-lock=(), usb=(), web-share=()")

			// CSP only for frontend routes — skip admin UI and API
			if !strings.HasPrefix(path, "/_/") && !strings.HasPrefix(path, "/api/") {
				h.Set("Content-Security-Policy", cachedCSP.Load().(string))
			}

			// Short cache for public read-only API list/view endpoints (GET only, no auth)
			if strings.HasPrefix(path, "/api/collections/") && e.Request.Method == "GET" &&
				e.Request.Header.Get("Authorization") == "" {
				h.Set("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
			}

			// Cache uploaded file downloads (images etc.) — content-addressed by filename
			if strings.HasPrefix(path, "/api/files/") && e.Request.Method == "GET" {
				h.Set("Cache-Control", "public, max-age=604800, stale-while-revalidate=86400")
			}

			return e.Next()
		})

		// Compression middleware (zstd, gzip, deflate via klauspost/compress)
		// Skip in dev: compressWriter doesn't implement http.Hijacker, breaking Vite HMR WebSocket upgrades
		if os.Getenv("DEV") == "" {
			se.Router.BindFunc(middleware.Compress)
		}

		// Initialize AP keypair on first run
		if err := activitypub.EnsureKeypair(se.App); err != nil {
			log.Println("Warning: failed to ensure AP keypair:", err)
		}

		// Serve embedded frontend
		frontend, err := frontendFS()
		if err != nil {
			return err
		}

		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = se.App.Settings().Meta.AppURL
		}
		if baseURL == "" {
			baseURL = "http://127.0.0.1:8090"
		}

		// Build SPA index.html with injected instance metadata
		var spaHTML []byte
		if rawHTML, err := fs.ReadFile(frontend, "index.html"); err == nil {
			spaHTML = rawHTML
			if s, err := se.App.FindFirstRecordByFilter("settings", ""); err == nil {
				page := string(rawHTML)
				if name := s.GetString("instance_name"); name != "" {
					page = strings.Replace(page, "<title>Gather</title>", "<title>"+html.EscapeString(name)+"</title>", 1)
				}
				if desc := s.GetString("instance_description"); desc != "" {
					page = strings.Replace(page,
						`content="Community events calendar — discover upcoming events near you."`,
						`content="`+html.EscapeString(desc)+`"`, 1)
				}
				spaHTML = []byte(page)
			}
		}
		serveSPA := func(re *core.RequestEvent) error {
			re.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
			re.Response.Header().Set("Cache-Control", "no-cache")
			_, err := re.Response.Write(spaHTML)
			return err
		}

		// RSS feeds
		se.Router.GET("/feed.rss", func(re *core.RequestEvent) error {
			data, err := rss.GenerateFeed(se.App, baseURL, "")
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "application/rss+xml")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "application/rss+xml", data)
		})

		se.Router.GET("/feed/tag/{tagname}", func(re *core.RequestEvent) error {
			tagname := re.Request.PathValue("tagname")
			// Strip .rss extension if present
			tagname = strings.TrimSuffix(tagname, ".rss")
			tag, err := se.App.FindFirstRecordByFilter("tags", "name = {:name}", map[string]any{"name": tagname})
			if err != nil {
				return re.NotFoundError("Tag not found", err)
			}
			filter := "tags.id ?= '" + tag.Id + "'"
			data, err := rss.GenerateFeed(se.App, baseURL, filter)
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "application/rss+xml")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "application/rss+xml", data)
		})

		// ICS feeds
		se.Router.GET("/feed.ics", func(re *core.RequestEvent) error {
			data, err := ical.GenerateFeed(se.App, baseURL, "")
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "text/calendar")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "text/calendar", data)
		})

		se.Router.GET("/ics/tag/{tagname}", func(re *core.RequestEvent) error {
			tagname := re.Request.PathValue("tagname")
			tag, err := se.App.FindFirstRecordByFilter("tags", "name = {:name}", map[string]any{"name": tagname})
			if err != nil {
				return re.NotFoundError("Tag not found", err)
			}
			filter := "tags.id ?= '" + tag.Id + "'"
			data, err := ical.GenerateFeed(se.App, baseURL, filter)
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "text/calendar")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "text/calendar", data)
		})

		se.Router.GET("/ics/event/{id}", func(re *core.RequestEvent) error {
			id := re.Request.PathValue("id")
			data, err := ical.GenerateSingleEvent(se.App, baseURL, id)
			if err != nil {
				return re.NotFoundError("Event not found", err)
			}
			re.Response.Header().Set("Content-Type", "text/calendar")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "text/calendar", data)
		})

		// Sitemap
		se.Router.GET("/sitemap.xml", func(re *core.RequestEvent) error {
			data, err := seo.GenerateSitemap(se.App, baseURL)
			if err != nil {
				return re.InternalServerError("Failed to generate sitemap", err)
			}
			re.Response.Header().Set("Cache-Control", "public, max-age=3600")
			return re.Blob(200, "application/xml", data)
		})

		// Robots.txt
		se.Router.GET("/robots.txt", func(re *core.RequestEvent) error {
			content := seo.BuildRobotsTxt(baseURL)
			re.Response.Header().Set("Cache-Control", "public, max-age=86400")
			return re.String(200, content)
		})

		// ActivityPub actor
		se.Router.GET("/ap/actor", func(re *core.RequestEvent) error {
			actor, err := activitypub.GetActor(se.App, baseURL)
			if err != nil {
				return re.InternalServerError("Failed to get actor", err)
			}
			data, err := actor.ToJSON()
			if err != nil {
				return re.InternalServerError("Failed to serialize actor", err)
			}
			re.Response.Header().Set("Content-Type", "application/activity+json")
			re.Response.Header().Set("Cache-Control", "public, max-age=3600")
			return re.Blob(200, "application/activity+json", data)
		})

		// Webfinger
		se.Router.GET("/.well-known/webfinger", func(re *core.RequestEvent) error {
			resource := re.Request.URL.Query().Get("resource")
			if resource == "" {
				return re.BadRequestError("Missing resource parameter", nil)
			}
			data, err := activitypub.HandleWebfinger(resource, baseURL)
			if err != nil {
				return re.NotFoundError("Resource not found", err)
			}
			re.Response.Header().Set("Content-Type", "application/jrd+json")
			re.Response.Header().Set("Cache-Control", "public, max-age=3600")
			return re.Blob(200, "application/jrd+json", data)
		})

		// ActivityPub outbox
		se.Router.GET("/ap/outbox", func(re *core.RequestEvent) error {
			data, err := activitypub.GetOutbox(se.App, baseURL)
			if err != nil {
				return re.InternalServerError("Failed to get outbox", err)
			}
			re.Response.Header().Set("Content-Type", "application/activity+json")
			re.Response.Header().Set("Cache-Control", "public, max-age=300")
			return re.Blob(200, "application/activity+json", data)
		})

		// ActivityPub inbox
		se.Router.POST("/ap/inbox", func(re *core.RequestEvent) error {
			if err := activitypub.HandleInbox(se.App, baseURL, re.Request.Body); err != nil {
				return re.InternalServerError("Failed to process activity", err)
			}
			return re.NoContent(202)
		})

		// Favicon route - serves favicon from settings, with default SVG fallback
		se.Router.GET("/favicon.ico", func(re *core.RequestEvent) error {
			settings, err := se.App.FindFirstRecordByFilter("settings", "")
			if err == nil {
				favicon := settings.GetString("favicon")
				if favicon != "" {
					fs, err := se.App.NewFilesystem()
					if err == nil {
						defer fs.Close()

						filePath := settings.BaseFilesPath() + "/" + favicon
						reader, err := fs.GetReader(filePath)
						if err == nil {
							defer reader.Close()

							contentType := "image/svg+xml"
							if strings.HasSuffix(favicon, ".png") {
								contentType = "image/png"
							} else if strings.HasSuffix(favicon, ".webp") {
								contentType = "image/webp"
							} else if strings.HasSuffix(favicon, ".jpg") || strings.HasSuffix(favicon, ".jpeg") {
								contentType = "image/jpeg"
							} else if strings.HasSuffix(favicon, ".ico") {
								contentType = "image/x-icon"
							}

							re.Response.Header().Set("Content-Type", contentType)
							re.Response.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=300")
							re.Response.WriteHeader(http.StatusOK)
							_, err = io.Copy(re.Response, reader)
							return err
						}
					}
				}
			}

			// Default SVG favicon
			re.Response.Header().Set("Content-Type", "image/svg+xml")
			re.Response.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=300")
			re.Response.WriteHeader(http.StatusOK)
			_, err = re.Response.Write([]byte(defaultFavicon))
			return err
		})

		// Nearby places — GET /api/places/nearby?lat=X&lon=Y&radius=km
		se.Router.GET("/api/places/nearby", func(re *core.RequestEvent) error {
			latStr := re.Request.URL.Query().Get("lat")
			lonStr := re.Request.URL.Query().Get("lon")
			radiusStr := re.Request.URL.Query().Get("radius")

			if latStr == "" || lonStr == "" {
				return re.BadRequestError("lat and lon are required", nil)
			}

			lat, err := strconv.ParseFloat(latStr, 64)
			if err != nil || lat < -90 || lat > 90 {
				return re.BadRequestError("invalid lat", nil)
			}
			lon, err := strconv.ParseFloat(lonStr, 64)
			if err != nil || lon < -180 || lon > 180 {
				return re.BadRequestError("invalid lon", nil)
			}

			radius := 10.0 // default 10 km
			if radiusStr != "" {
				if r, err := strconv.ParseFloat(radiusStr, 64); err == nil && r > 0 && r <= 500 {
					radius = r
				}
			}

			// Use PocketBase filter with geoDistance function
			filter := "status = 'approved' && geoDistance(location.lon, location.lat, {:lon}, {:lat}) <= {:radius}"
			nearby, err := se.App.FindRecordsByFilter(
				"places",
				filter,
				"",
				50,
				0,
				dbx.Params{"lat": lat, "lon": lon, "radius": radius},
			)
			if err != nil {
				return re.InternalServerError("Failed to query nearby places", err)
			}

			type placeResult struct {
				ID          string  `json:"id"`
				Name        string  `json:"name"`
				Address     string  `json:"address,omitempty"`
				City        string  `json:"city,omitempty"`
				CountryCode string  `json:"country_code,omitempty"`
				Lat         float64 `json:"lat"`
				Lon         float64 `json:"lon"`
				Status      string  `json:"status"`
			}
			results := make([]placeResult, 0, len(nearby))
			for _, p := range nearby {
				loc := p.GetGeoPoint("location")
				results = append(results, placeResult{
					ID:          p.Id,
					Name:        p.GetString("name"),
					Address:     p.GetString("address"),
					City:        p.GetString("city"),
					CountryCode: p.GetString("country_code"),
					Lat:         loc.Lat,
					Lon:         loc.Lon,
					Status:      p.GetString("status"),
				})
			}

			re.Response.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
			return re.JSON(200, results)
		})

		// Unified search endpoint
		se.Router.GET("/api/search", func(re *core.RequestEvent) error {
			q := strings.TrimSpace(re.Request.URL.Query().Get("q"))
			if len(q) < 2 {
				return re.JSON(200, map[string]any{
					"events": []any{},
					"places": []any{},
					"tags":   []any{},
				})
			}

			limit := 20
			if l, err := strconv.Atoi(re.Request.URL.Query().Get("limit")); err == nil && l > 0 && l <= 50 {
				limit = l
			}

			// Search events
			events, _ := se.App.FindRecordsByFilter(
				"events",
				"status = 'published' && (title ~ {:q} || description ~ {:q})",
				"-start_datetime",
				limit,
				0,
				dbx.Params{"q": q},
			)

			// Expand place and tags for events
			eventResults := make([]map[string]any, 0, len(events))
			for _, e := range events {
				ev := map[string]any{
					"id":             e.Id,
					"slug":           e.GetString("slug"),
					"title":          e.GetString("title"),
					"description":    e.GetString("description"),
					"start_datetime": e.GetString("start_datetime"),
					"end_datetime":   e.GetString("end_datetime"),
					"status":         e.GetString("status"),
					"image":          e.GetString("image"),
				}
				expand := map[string]any{}
				if placeID := e.GetString("place"); placeID != "" {
					if place, err := se.App.FindRecordById("places", placeID); err == nil {
						expand["place"] = map[string]any{
							"id":      place.Id,
							"name":    place.GetString("name"),
							"address": place.GetString("address"),
							"city":    place.GetString("city"),
						}
					}
				}
				tagIDs := e.GetStringSlice("tags")
				if len(tagIDs) > 0 {
					tagList := make([]map[string]any, 0, len(tagIDs))
					for _, tid := range tagIDs {
						if tag, err := se.App.FindRecordById("tags", tid); err == nil {
							tagList = append(tagList, map[string]any{
								"id":    tag.Id,
								"name":  tag.GetString("name"),
								"color": tag.GetString("color"),
							})
						}
					}
					expand["tags"] = tagList
				}
				if len(expand) > 0 {
					ev["expand"] = expand
				}
				eventResults = append(eventResults, ev)
			}

			// Search places
			places, _ := se.App.FindRecordsByFilter(
				"places",
				"status = 'approved' && (name ~ {:q} || address ~ {:q} || city ~ {:q})",
				"name",
				limit,
				0,
				dbx.Params{"q": q},
			)
			placeResults := make([]map[string]any, 0, len(places))
			for _, p := range places {
				placeResults = append(placeResults, map[string]any{
					"id":      p.Id,
					"name":    p.GetString("name"),
					"address": p.GetString("address"),
					"city":    p.GetString("city"),
				})
			}

			// Search tags
			tags, _ := se.App.FindRecordsByFilter(
				"tags",
				"status = 'approved' && name ~ {:q}",
				"name",
				limit,
				0,
				dbx.Params{"q": q},
			)
			tagResults := make([]map[string]any, 0, len(tags))
			for _, t := range tags {
				tagResults = append(tagResults, map[string]any{
					"id":    t.Id,
					"name":  t.GetString("name"),
					"color": t.GetString("color"),
				})
			}

			return re.JSON(200, map[string]any{
				"events": eventResults,
				"places": placeResults,
				"tags":   tagResults,
			})
		})

		// Expanded events listing with recurrence support
		se.Router.GET("/api/events/expanded", func(re *core.RequestEvent) error {
			rangeStartStr := re.Request.URL.Query().Get("start")
			rangeEndStr := re.Request.URL.Query().Get("end")
			town := re.Request.URL.Query().Get("town")
			tagIDsStr := re.Request.URL.Query().Get("tags")
			var tagIDs []string
			if tagIDsStr != "" {
				tagIDs = strings.Split(tagIDsStr, ",")
			}
			pageStr := re.Request.URL.Query().Get("page")
			pageSizeStr := re.Request.URL.Query().Get("pageSize")

			page := 1
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
			pageSize := 20
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
				pageSize = ps
			}

			now := time.Now().UTC()
			rangeStart := now
			if rangeStartStr != "" {
				if t, err := time.Parse("2006-01-02", rangeStartStr); err == nil {
					rangeStart = t
				}
			}
			rangeEnd := now.AddDate(0, 3, 0)
			if rangeEndStr != "" {
				if t, err := time.Parse("2006-01-02", rangeEndStr); err == nil {
					rangeEnd = t.Add(24*time.Hour - time.Second)
				}
			}

			events, total, err := recurrence.ListExpandedEvents(se.App, rangeStart, rangeEnd, town, tagIDs, page, pageSize)
			if err != nil {
				return re.InternalServerError("Failed to list events", err)
			}

			re.Response.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
			return re.JSON(200, map[string]any{
				"items":      events,
				"totalItems": total,
				"page":       page,
				"perPage":    pageSize,
			})
		})

		// Single event by virtual or real ID
		se.Router.GET("/api/events/expanded/{id}", func(re *core.RequestEvent) error {
			id := re.Request.PathValue("id")

			baseID, dateStr, isVirtual := recurrence.ParseVirtualID(id)
			if !isVirtual {
				record, err := se.App.FindRecordById("events", id)
				if err != nil {
					return re.NotFoundError("Event not found", err)
				}
				return re.JSON(200, recurrence.RecordToExpandedPublic(se.App, record, "", nil))
			}

			record, err := se.App.FindRecordById("events", baseID)
			if err != nil {
				return re.NotFoundError("Event not found", err)
			}

			occDate, err := time.Parse("20060102", dateStr)
			if err != nil {
				return re.BadRequestError("Invalid date in event ID", err)
			}

			originalStart := record.GetDateTime("start_datetime").Time()
			occStart := time.Date(occDate.Year(), occDate.Month(), occDate.Day(),
				originalStart.Hour(), originalStart.Minute(), originalStart.Second(), 0, time.UTC)

			rule := record.GetString("recurrence_rule")
			exceptions := record.GetString("recurrence_exceptions")

			occurrences, err := recurrence.Expand(rule, originalStart, occStart, occStart.Add(time.Second))
			if err != nil || len(occurrences) == 0 {
				return re.NotFoundError("Occurrence not found", nil)
			}

			if len(recurrence.FilterExceptions(occurrences, exceptions)) == 0 {
				return re.NotFoundError("This occurrence has been cancelled", nil)
			}

			originalEnd := record.GetDateTime("end_datetime").Time()
			ve := recurrence.BuildVirtualEvent(baseID, originalStart, originalEnd, occStart)
			return re.JSON(200, recurrence.RecordToExpandedPublic(se.App, record, ve.ID, &ve))
		})

		// Cancel a single occurrence
		se.Router.POST("/api/events/{id}/cancel-occurrence", func(re *core.RequestEvent) error {
			info, _ := re.RequestInfo()
			if info == nil || info.Auth == nil {
				return re.UnauthorizedError("Authentication required", nil)
			}
			role := info.Auth.GetString("role")
			if role != "admin" && role != "editor" {
				return re.ForbiddenError("Insufficient permissions", nil)
			}

			id := re.Request.PathValue("id")
			var body struct {
				Date string `json:"date"`
			}
			if err := re.BindBody(&body); err != nil || body.Date == "" {
				return re.BadRequestError("Missing 'date' field", nil)
			}

			record, err := se.App.FindRecordById("events", id)
			if err != nil {
				return re.NotFoundError("Event not found", err)
			}

			if record.GetString("recurrence_rule") == "" {
				return re.BadRequestError("Event is not recurring", nil)
			}

			exceptions := record.GetString("recurrence_exceptions")
			if exceptions == "" {
				exceptions = body.Date
			} else {
				exceptions = exceptions + "," + body.Date
			}
			record.Set("recurrence_exceptions", exceptions)

			if err := se.App.Save(record); err != nil {
				return re.InternalServerError("Failed to save", err)
			}

			return re.JSON(200, map[string]string{"status": "ok"})
		})

		// Register cron jobs
		cron.Register(se.App, baseURL)

		// ActivityPub delivery worker — every 5 minutes
		se.App.Cron().Add("ap-delivery", "*/5 * * * *", func() {
			activitypub.ProcessDeliveryQueue(se.App)
		})

		// POST /api/ap/test-delivery — send a test Note to a given inbox (superuser/admin only)
		se.Router.POST("/api/ap/test-delivery", func(re *core.RequestEvent) error {
			info, _ := re.RequestInfo()
			if info == nil || info.Auth == nil {
				return re.UnauthorizedError("Authentication required", nil)
			}
			if info.Auth.Collection().Name != "_superusers" {
				role := info.Auth.GetString("role")
				if role != "admin" {
					return re.ForbiddenError("Superuser or admin required", nil)
				}
			}

			var body struct {
				TargetInbox string `json:"target_inbox"`
			}
			if err := re.BindBody(&body); err != nil || body.TargetInbox == "" {
				return re.BadRequestError("Missing target_inbox", nil)
			}

			// Validate URL to prevent SSRF — must be http or https
			parsed, parseErr := url.Parse(body.TargetInbox)
			if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return re.BadRequestError("target_inbox must be a valid http or https URL", nil)
			}

			actor, err := activitypub.GetActor(se.App, baseURL)
			if err != nil {
				return re.InternalServerError("Failed to get actor", err)
			}

			testActivity := activitypub.Activity{
				Context: "https://www.w3.org/ns/activitystreams",
				Type:    "Create",
				ID:      baseURL + "/ap/test/" + time.Now().Format("20060102150405"),
				Actor:   actor.ID,
				To:      []string{"https://www.w3.org/ns/activitystreams#Public"},
				Object: map[string]any{
					"type":    "Note",
					"content": "Test delivery from Gather instance",
				},
			}

			err = activitypub.DeliverActivity(se.App, testActivity, body.TargetInbox)
			if err != nil {
				return re.InternalServerError("Delivery failed: "+err.Error(), err)
			}
			return re.JSON(200, map[string]any{"success": true})
		})

		// POST /api/ap/retry-queue — immediately process all pending queue records (superuser/admin only)
		se.Router.POST("/api/ap/retry-queue", func(re *core.RequestEvent) error {
			info, _ := re.RequestInfo()
			if info == nil || info.Auth == nil {
				return re.UnauthorizedError("Authentication required", nil)
			}
			if info.Auth.Collection().Name != "_superusers" {
				role := info.Auth.GetString("role")
				if role != "admin" {
					return re.ForbiddenError("Superuser or admin required", nil)
				}
			}
			go activitypub.ProcessDeliveryQueue(se.App)
			return re.JSON(200, map[string]any{"status": "processing"})
		})

		// Register hooks
		hooks.RegisterUserHooks(se.App)
		hooks.RegisterEventHooks(se.App, baseURL)
		hooks.RegisterModerationHooks(se.App)
		hooks.RegisterImageConversionHooks(se.App)
		hooks.RegisterSlugHooks(se.App)

		// Dev mode: proxy to Vite dev server
		devMode := os.Getenv("DEV") != ""
		var viteProxy *httputil.ReverseProxy
		if devMode {
			viteURL, _ := url.Parse("http://localhost:5173")
			viteProxy = httputil.NewSingleHostReverseProxy(viteURL)
			log.Println("Dev mode: proxying frontend to http://localhost:5173")
		}

		// Schema.org Event metadata for bots
		se.Router.GET("/event/{id}", func(re *core.RequestEvent) error {
			userAgent := re.Request.Header.Get("User-Agent")

			// Non-bots get the SPA
			if !seo.IsBot(userAgent) {
				return serveSPA(re)
			}

			id := re.Request.PathValue("id")

			// Try slug lookup first, then fall back to ID (backward compat)
			event, err := se.App.FindFirstRecordByFilter("events", "slug = {:slug}", map[string]any{"slug": id})
			if err != nil {
				event, err = se.App.FindRecordById("events", id)
			}
			if err != nil {
				return serveSPA(re) // Let SPA handle 404
			}

			// Only serve metadata for published events
			if event.GetString("status") != "published" {
				return serveSPA(re)
			}

			// Generate HTML with metadata
			html, err := seo.GenerateEventHTML(se.App, event, baseURL)
			if err != nil {
				log.Println("Failed to generate SEO HTML:", err)
				return serveSPA(re) // Fallback to SPA
			}

			re.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
			re.Response.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=300")
			return re.Blob(200, "text/html", html)
		})

		// Home page: SSR for bots, SPA/Vite for humans
		// Use /{$} to match only exact root — GET / conflicts with GET /{path...} in Go 1.22+
		se.Router.GET("/{$}", func(re *core.RequestEvent) error {
			userAgent := re.Request.Header.Get("User-Agent")
			if !seo.IsBot(userAgent) {
				if devMode && viteProxy != nil {
					viteProxy.ServeHTTP(re.Response, re.Request)
					return nil
				}
				return serveSPA(re)
			}

			html, err := seo.GenerateHomeHTML(se.App, baseURL)
			if err != nil {
				log.Println("Failed to generate home SEO HTML:", err)
				return serveSPA(re)
			}

			re.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
			// no-store: prevent CDN from caching bot-specific HTML and serving it to humans
			re.Response.Header().Set("Cache-Control", "no-store")
			return re.Blob(200, "text/html", html)
		})

		// Serve static files, fallback to index.html for SPA routing
		se.Router.GET("/{path...}", func(re *core.RequestEvent) error {
			path := re.Request.PathValue("path")

			// Skip API and admin routes
			if strings.HasPrefix(path, "api/") || strings.HasPrefix(path, "_/") {
				return re.Next()
			}

			// Dev mode: proxy to Vite
			if devMode && viteProxy != nil {
				viteProxy.ServeHTTP(re.Response, re.Request)
				return nil
			}

			// Try to serve the file
			f, err := frontend.Open(path)
			if err == nil {
				f.Close()
				// Vite assets have content hashes — safe to cache for 1 year
				if strings.HasPrefix(path, "assets/") {
					re.Response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				return re.FileFS(frontend, path)
			}

			// Fallback to index.html for SPA
			return serveSPA(re)
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
