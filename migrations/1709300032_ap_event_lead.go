package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Fediverse event posts go out N days before the event instead of the moment
// it is published. events.ap_posted_at records that followers have the post
// (and is cleared again when a Delete is sent), so edits to events that were
// never posted stay silent and the scheduler never posts twice. Events that
// were already published before this migration were posted then, so they are
// backfilled as posted. Idempotent: safe to re-run.
func init() {
	m.Register(func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("events")
		if err != nil {
			return err
		}
		if events.Fields.GetByName("ap_posted_at") == nil {
			events.Fields.Add(&core.DateField{Name: "ap_posted_at", Hidden: true})
			if err := app.Save(events); err != nil {
				return err
			}
			if _, err := app.DB().NewQuery(
				"UPDATE events SET ap_posted_at = created WHERE status = 'published' AND (ap_posted_at IS NULL OR ap_posted_at = '')",
			).Execute(); err != nil {
				return err
			}
		}

		settings, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		if settings.Fields.GetByName("ap_event_lead_days") == nil {
			min, max := 0.0, 60.0
			settings.Fields.Add(&core.NumberField{Name: "ap_event_lead_days", Min: &min, Max: &max, OnlyInt: true})
			if err := app.Save(settings); err != nil {
				return err
			}
			// existing instances get the new default; 0 would keep the old "post immediately"
			if _, err := app.DB().Update("settings", dbx.Params{"ap_event_lead_days": 3}, nil).Execute(); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		if settings, err := app.FindCollectionByNameOrId("settings"); err == nil {
			settings.Fields.RemoveByName("ap_event_lead_days")
			if err := app.Save(settings); err != nil {
				return err
			}
		}
		if events, err := app.FindCollectionByNameOrId("events"); err == nil {
			events.Fields.RemoveByName("ap_posted_at")
			return app.Save(events)
		}
		return nil
	})
}
