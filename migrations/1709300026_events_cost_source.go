package migrations

import (
	"gather/internal/textutil"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Scrapers append "Cost: …" and "Source: <url>" lines to the description text.
// Promote them to real fields so structured data (schema.org offers) can use
// them, and backfill existing events from the boilerplate.
func init() {
	m.Register(func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("events")
		if err != nil {
			return err
		}
		if events.Fields.GetByName("cost") == nil {
			events.Fields.Add(&core.TextField{Name: "cost", Max: 200})
		}
		if events.Fields.GetByName("source_url") == nil {
			events.Fields.Add(&core.URLField{Name: "source_url"})
		}
		if err := app.Save(events); err != nil {
			return err
		}
		return textutil.BackfillCostSource(app)
	}, func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("events")
		if err != nil {
			return err
		}
		events.Fields.RemoveByName("cost")
		events.Fields.RemoveByName("source_url")
		return app.Save(events)
	})
}
