package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

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

		field.MaxSize = 20 * 1024 * 1024

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

		field.MaxSize = 5 * 1024 * 1024

		return app.Save(events)
	})
}
