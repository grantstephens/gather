package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Add created/updated autodate fields to events, places, and tags.
// These weren't added originally because PocketBase v0.36 no longer adds them automatically.
func init() {
	collections := []string{"events", "places", "tags"}

	m.Register(func(app core.App) error {
		for _, name := range collections {
			col, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}

			if col.Fields.GetByName("created") == nil {
				col.Fields.Add(&core.AutodateField{
					Name:     "created",
					OnCreate: true,
				})
			}
			if col.Fields.GetByName("updated") == nil {
				col.Fields.Add(&core.AutodateField{
					Name:     "updated",
					OnCreate: true,
					OnUpdate: true,
				})
			}

			if err := app.Save(col); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, name := range collections {
			col, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			col.Fields.RemoveByName("created")
			col.Fields.RemoveByName("updated")
			if err := app.Save(col); err != nil {
				return err
			}
		}
		return nil
	})
}
