package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The frontend requests a "200x150" thumb for event card srcsets
// (see frontend/src/components/EventCard.tsx), but that size was never
// registered on the events.image field. PocketBase only generates a thumb
// for sizes present in FileField.Thumbs — any other size falls back to
// serving the original, unprocessed file. For large source images this
// meant a 71x112px card image could ship an 800KB+ original instead of a
// small cropped thumb. Registering "200x150" here lets PocketBase generate
// and cache the actual thumbnail.
func init() {
	m.Register(func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("events")
		if err != nil {
			return err
		}

		field, ok := events.Fields.GetByName("image").(*core.FileField)
		if !ok {
			return nil // field doesn't exist, nothing to do
		}

		field.Thumbs = []string{"100x100", "200x150", "400x300", "800x600"}

		return app.Save(events)
	}, func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("events")
		if err != nil {
			return err
		}

		field, ok := events.Fields.GetByName("image").(*core.FileField)
		if !ok {
			return nil
		}

		field.Thumbs = []string{"100x100", "400x300", "800x600"}

		return app.Save(events)
	})
}
