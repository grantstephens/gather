package newsletter

import (
	"testing"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// setNewsletterSettings upserts the settings singleton with the given toggle.
func setNewsletterSettings(t testing.TB, app *tests.TestApp, toggle bool, fields map[string]any) {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := app.FindFirstRecordByFilter("settings", "")
	if err != nil {
		rec = core.NewRecord(col)
		rec.Set("instance_name", "Test Gather")
	}
	rec.Set("newsletter_enabled", toggle)
	for k, v := range fields {
		rec.Set(k, v)
	}
	if err := app.Save(rec); err != nil {
		t.Fatal(err)
	}
}

func setMail(app *tests.TestApp, smtp bool, sender string) {
	app.Settings().SMTP.Enabled = smtp
	app.Settings().Meta.SenderAddress = sender
}

func TestEnabledRequiresMail(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()

	cases := []struct {
		name   string
		toggle bool
		smtp   bool
		sender string
		want   bool
	}{
		{"toggle on, smtp off", true, false, "a@x.test", false},
		{"toggle on, smtp on, no sender", true, true, "", false},
		{"toggle on, smtp on, sender", true, true, "a@x.test", true},
		{"toggle off, smtp on, sender", false, true, "a@x.test", false},
	}
	for _, c := range cases {
		setNewsletterSettings(t, app, c.toggle, nil)
		setMail(app, c.smtp, c.sender)
		if got := LoadConfig(app).Enabled(); got != c.want {
			t.Errorf("%s: Enabled() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestLoadConfigReadsSchedule(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	setNewsletterSettings(t, app, true, map[string]any{
		"newsletter_send_day":  4,
		"newsletter_send_time": "08:30",
		"newsletter_footer":    "Gather, 1 High St",
	})
	cfg := LoadConfig(app)
	if cfg.SendDay != 4 || cfg.SendTime != "08:30" || cfg.Footer != "Gather, 1 High St" || !cfg.ToggleOn {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigNoSettingsIsZero(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	if cfg := LoadConfig(app); cfg.Enabled() || cfg.ToggleOn {
		t.Errorf("expected zero config, got %+v", cfg)
	}
}

func TestPrivacyURL(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()

	if got := PrivacyURL(app); got != "" {
		t.Fatalf("expected empty without a privacy page, got %q", got)
	}
	pages, err := app.FindCollectionByNameOrId("pages")
	if err != nil {
		t.Fatal(err)
	}
	rec := core.NewRecord(pages)
	rec.Set("title", "Privacy")
	rec.Set("slug", "privacy")
	if err := app.Save(rec); err != nil {
		t.Fatal(err)
	}
	if got := PrivacyURL(app); got != "/privacy" {
		t.Errorf("PrivacyURL = %q, want /privacy", got)
	}
}
