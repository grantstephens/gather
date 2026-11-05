package newsletter

import (
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/routine"
)

// requireAdmin mirrors the check on /api/ap/test-delivery: a superuser or a
// user with role "admin". It returns the caller's email.
func requireAdmin(re *core.RequestEvent) (string, error) {
	info, _ := re.RequestInfo()
	if info == nil || info.Auth == nil {
		return "", re.UnauthorizedError("Authentication required", nil)
	}
	if info.Auth.Collection().Name != "_superusers" && info.Auth.GetString("role") != "admin" {
		return "", re.ForbiddenError("Superuser or admin required", nil)
	}
	return info.Auth.GetString("email"), nil
}

// Register adds the newsletter HTTP routes.
func Register(se *core.ServeEvent, baseURL string) {
	app := se.App

	// Public: does this instance offer the newsletter, and is there a privacy page?
	se.Router.GET("/api/newsletter/config", func(re *core.RequestEvent) error {
		var privacy any
		if p := PrivacyURL(app); p != "" {
			privacy = p
		}
		return re.JSON(http.StatusOK, map[string]any{
			"enabled":     Available(app, baseURL),
			"privacy_url": privacy,
		})
	})

	se.Router.POST("/api/newsletter/subscribe", func(re *core.RequestEvent) error {
		var body struct {
			Email   string `json:"email"`
			Website string `json:"website"` // honeypot
		}
		if err := re.BindBody(&body); err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"message": "Please enter a valid email address."})
		}
		pc, err := Subscribe(app, body.Email, body.Website, baseURL, time.Now())
		switch err {
		case nil:
		case ErrInvalidEmail:
			return re.JSON(http.StatusBadRequest, map[string]string{"message": "Please enter a valid email address."})
		case ErrUnavailable:
			return re.JSON(http.StatusServiceUnavailable, map[string]string{"message": "The newsletter isn't available right now."})
		default:
			app.Logger().Error("newsletter: subscribe failed", "error", err)
			return re.JSON(http.StatusInternalServerError, map[string]string{"message": "Something went wrong. Please try again."})
		}
		if pc != nil {
			// Sent after the response so timing doesn't reveal whether an address
			// was already known.
			routine.FireAndForget(func() {
				if err := SendConfirmation(app, pc); err != nil {
					app.Logger().Error("newsletter: confirmation email failed", "error", err)
				}
			})
		}
		return re.JSON(http.StatusOK, map[string]string{"message": "Check your inbox to confirm."})
	})

	// State changes are POST-only behind an SPA button so link scanners that
	// GET every URL in an email can't confirm or unsubscribe anyone.
	se.Router.POST("/api/newsletter/confirm", func(re *core.RequestEvent) error {
		if Confirm(app, re.Request.URL.Query().Get("token"), time.Now()) {
			return re.JSON(http.StatusOK, map[string]string{"status": "confirmed"})
		}
		return re.JSON(http.StatusBadRequest, map[string]string{"status": "invalid"})
	})

	// Also the RFC 8058 one-click target: the body is form-encoded and ignored.
	se.Router.POST("/api/newsletter/unsubscribe", func(re *core.RequestEvent) error {
		Unsubscribe(app, re.Request.URL.Query().Get("token"))
		return re.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	se.Router.GET("/newsletter/open/{id}", PixelHandler(app, baseURL))

	se.Router.GET("/api/newsletter/admin", func(re *core.RequestEvent) error {
		if _, err := requireAdmin(re); err != nil {
			return err
		}
		cfg := LoadConfig(app)
		confirmed, _ := app.CountRecords(subscribersCol, dbx.HashExp{"status": "confirmed"})
		pending, _ := app.CountRecords(subscribersCol, dbx.HashExp{"status": "pending"})
		return re.JSON(http.StatusOK, map[string]any{
			"enabled":         cfg.Enabled() && !localOnly(baseURL),
			"base_url_local":  localOnly(baseURL),
			"toggle_on":       cfg.ToggleOn,
			"mail_configured": cfg.MailConfigured,
			"confirmed":       confirmed,
			"pending":         pending,
			"timezone":        time.Local.String(),
			"send_day":        cfg.SendDay,
			"send_time":       cfg.SendTime,
		})
	})

	se.Router.POST("/api/newsletter/test", func(re *core.RequestEvent) error {
		email, err := requireAdmin(re)
		if err != nil {
			return err
		}
		if !LoadConfig(app).MailConfigured {
			return re.BadRequestError("Mail isn't configured in PocketBase.", nil)
		}
		if email == "" {
			return re.BadRequestError("Your account has no email address.", nil)
		}
		if err := SendTest(app, email, baseURL); err != nil {
			return re.BadRequestError("Couldn't send the test email: "+err.Error(), nil)
		}
		return re.JSON(http.StatusOK, map[string]string{"status": "sent"})
	})
}

// RegisterCron adds the every-minute send check and the daily purge of stale
// unconfirmed signups.
func RegisterCron(app core.App, baseURL string) {
	if err := app.Cron().Add("newsletter-tick", "* * * * *", func() {
		// +30s then truncate: a tick landing at 08:59:59.9 still counts as 09:00.
		at := time.Now().Add(30 * time.Second).Truncate(time.Minute)
		if _, err := Tick(app, at, time.Local, baseURL); err != nil {
			app.Logger().Error("newsletter-tick failed", "error", err)
		}
	}); err != nil {
		app.Logger().Error("newsletter: couldn't register tick job", "error", err)
	}
	if err := app.Cron().Add("newsletter-purge", "0 4 * * *", func() {
		if n, err := PurgePending(app, time.Now()); err != nil {
			app.Logger().Error("newsletter-purge failed", "error", err)
		} else if n > 0 {
			app.Logger().Info("newsletter-purge: removed stale unconfirmed signups", "count", n)
		}
	}); err != nil {
		app.Logger().Error("newsletter: couldn't register purge job", "error", err)
	}
}
