// Package newsletter implements the weekly picks email: subscribers, schedule,
// rendering, sending and open tracking.
package newsletter

import (
	"github.com/pocketbase/pocketbase/core"
)

// Config is the newsletter's view of the settings singleton plus PocketBase's
// mail configuration.
type Config struct {
	ToggleOn       bool
	MailConfigured bool
	SendDay        int // 0 = Sunday
	SendTime       string
	Footer         string
}

// Enabled is the master switch: the admin toggle AND working mail config.
func (c Config) Enabled() bool { return c.ToggleOn && c.MailConfigured }

// LoadConfig reads the current config. It is never cached: the toggle and
// PocketBase's SMTP settings can change at any time.
func LoadConfig(app core.App) Config {
	cfg := Config{
		MailConfigured: app.Settings().SMTP.Enabled && app.Settings().Meta.SenderAddress != "",
	}
	rec, err := app.FindFirstRecordByFilter("settings", "")
	if err != nil {
		return cfg
	}
	cfg.ToggleOn = rec.GetBool("newsletter_enabled")
	cfg.SendDay = rec.GetInt("newsletter_send_day")
	cfg.SendTime = rec.GetString("newsletter_send_time")
	cfg.Footer = rec.GetString("newsletter_footer")
	return cfg
}

// PrivacyURL returns "/privacy" when the operator has published a page with
// slug "privacy", otherwise "" (so links are never dead).
func PrivacyURL(app core.App) string {
	if _, err := app.FindFirstRecordByFilter("pages", "slug = 'privacy'"); err != nil {
		return ""
	}
	return "/privacy"
}
