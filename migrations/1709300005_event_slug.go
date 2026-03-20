package migrations

import (
	"gather/internal/slug"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("events")
		if err != nil {
			return err
		}

		// Add slug field
		slugField := &core.TextField{
			Name:     "slug",
			Required: false,
		}
		events.Fields.Add(slugField)
		if err := app.Save(events); err != nil {
			return err
		}

		// Backfill slugs for existing records
		records, err := app.FindRecordsByFilter("events", "", "", 0, 0)
		if err != nil {
			return err
		}
		for _, record := range records {
			if record.GetString("slug") != "" {
				continue
			}
			record.Set("slug", slug.Generate(record.GetString("title"), record.Id))
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("events")
		if err != nil {
			return err
		}
		events.Fields.RemoveByName("slug")
		return app.Save(events)
	})
}
