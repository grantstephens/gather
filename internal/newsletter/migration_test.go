package newsletter

import (
	"strings"
	"testing"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func TestNewsletterSchema(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()

	subs, err := app.FindCollectionByNameOrId("newsletter_subscribers")
	if err != nil {
		t.Fatalf("newsletter_subscribers collection missing: %v", err)
	}
	for _, name := range []string{"email", "status", "confirm_token", "unsubscribe_token", "confirm_sent_at", "confirmed_at", "created"} {
		if subs.Fields.GetByName(name) == nil {
			t.Errorf("newsletter_subscribers.%s missing", name)
		}
	}
	if subs.ListRule != nil || subs.ViewRule != nil || subs.CreateRule != nil || subs.UpdateRule != nil || subs.DeleteRule != nil {
		t.Error("newsletter_subscribers must have nil (superuser-only) rules")
	}
	for _, field := range []string{"email", "confirm_token", "unsubscribe_token"} {
		found := false
		for _, idx := range subs.Indexes {
			if strings.Contains(strings.ToUpper(idx), "UNIQUE") && strings.Contains(idx, field) {
				found = true
			}
		}
		if !found {
			t.Errorf("no unique index on %s (indexes: %v)", field, subs.Indexes)
		}
	}

	picks, _ := app.FindCollectionByNameOrId("picks")
	for _, name := range []string{"newsletter_sent_at", "newsletter_sent_count"} {
		f := picks.Fields.GetByName(name)
		if f == nil {
			t.Errorf("picks.%s missing", name)
			continue
		}
		if !f.GetHidden() {
			t.Errorf("picks.%s must be hidden from the public API", name)
		}
	}

	settings, _ := app.FindCollectionByNameOrId("settings")
	for _, name := range []string{"newsletter_enabled", "newsletter_send_day", "newsletter_send_time", "newsletter_footer"} {
		f := settings.Fields.GetByName(name)
		if f == nil {
			t.Errorf("settings.%s missing", name)
			continue
		}
		if f.GetHidden() {
			t.Errorf("settings.%s must not be hidden (admin form runs as role=admin)", name)
		}
	}
}

func TestSendTimePattern(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()

	settings, _ := app.FindCollectionByNameOrId("settings")
	save := func(sendTime string, day int) error {
		rec := core.NewRecord(settings)
		rec.Set("instance_name", "t")
		rec.Set("newsletter_send_time", sendTime)
		rec.Set("newsletter_send_day", day)
		return app.Save(rec)
	}
	if err := save("9:00", 4); err == nil {
		t.Error(`"9:00" should fail the HH:MM pattern`)
	}
	if err := save("25:00", 4); err == nil {
		t.Error(`"25:00" should fail the HH:MM pattern`)
	}
	if err := save("09:00", 7); err == nil {
		t.Error("send day 7 should be rejected (0-6)")
	}
	if err := save("09:00", 4); err != nil {
		t.Errorf(`"09:00" day 4 should be valid: %v`, err)
	}
}
