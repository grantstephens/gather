package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Weekly picks newsletter: subscriber list, per-post sent markers and the
// admin-configurable schedule. newsletter_subscribers has nil rules so only
// superusers (and our custom endpoints) can touch it.
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("newsletter_subscribers"); err != nil {
			subs := core.NewBaseCollection("newsletter_subscribers")
			subs.Fields.Add(&core.EmailField{Name: "email", Required: true})
			subs.Fields.Add(&core.SelectField{
				Name:      "status",
				Required:  true,
				MaxSelect: 1,
				Values:    []string{"pending", "confirmed"},
			})
			subs.Fields.Add(&core.TextField{Name: "confirm_token"})
			subs.Fields.Add(&core.TextField{Name: "unsubscribe_token"})
			subs.Fields.Add(&core.DateField{Name: "confirm_sent_at"})
			subs.Fields.Add(&core.DateField{Name: "confirmed_at"})
			subs.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
			subs.AddIndex("idx_newsletter_subscribers_email", true, "email", "")
			subs.AddIndex("idx_newsletter_subscribers_confirm_token", true, "confirm_token", "confirm_token != ''")
			subs.AddIndex("idx_newsletter_subscribers_unsubscribe_token", true, "unsubscribe_token", "unsubscribe_token != ''")
			if err := app.Save(subs); err != nil {
				return err
			}
		}

		// picks is publicly listable: keep the sent markers out of the public API.
		picks, err := app.FindCollectionByNameOrId("picks")
		if err != nil {
			return err
		}
		if picks.Fields.GetByName("newsletter_sent_at") == nil {
			picks.Fields.Add(&core.DateField{Name: "newsletter_sent_at", Hidden: true})
		}
		if picks.Fields.GetByName("newsletter_sent_count") == nil {
			picks.Fields.Add(&core.NumberField{Name: "newsletter_sent_count", Hidden: true, OnlyInt: true})
		}
		if err := app.Save(picks); err != nil {
			return err
		}

		// settings is publicly readable and the admin form runs as role=admin
		// (not superuser), so these must NOT be hidden. None of them is sensitive.
		settings, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		minDay, maxDay := 0.0, 6.0
		if settings.Fields.GetByName("newsletter_enabled") == nil {
			settings.Fields.Add(&core.BoolField{Name: "newsletter_enabled"})
		}
		if settings.Fields.GetByName("newsletter_send_day") == nil {
			settings.Fields.Add(&core.NumberField{Name: "newsletter_send_day", Min: &minDay, Max: &maxDay, OnlyInt: true})
		}
		if settings.Fields.GetByName("newsletter_send_time") == nil {
			settings.Fields.Add(&core.TextField{Name: "newsletter_send_time", Pattern: `^([01]\d|2[0-3]):[0-5]\d$`})
		}
		if settings.Fields.GetByName("newsletter_footer") == nil {
			settings.Fields.Add(&core.TextField{Name: "newsletter_footer"})
		}
		return app.Save(settings)
	}, func(app core.App) error {
		if settings, err := app.FindCollectionByNameOrId("settings"); err == nil {
			for _, f := range []string{"newsletter_enabled", "newsletter_send_day", "newsletter_send_time", "newsletter_footer"} {
				settings.Fields.RemoveByName(f)
			}
			if err := app.Save(settings); err != nil {
				return err
			}
		}
		if picks, err := app.FindCollectionByNameOrId("picks"); err == nil {
			picks.Fields.RemoveByName("newsletter_sent_at")
			picks.Fields.RemoveByName("newsletter_sent_count")
			if err := app.Save(picks); err != nil {
				return err
			}
		}
		if subs, err := app.FindCollectionByNameOrId("newsletter_subscribers"); err == nil {
			return app.Delete(subs)
		}
		return nil
	})
}
