package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("monday_as_start_of_week") != nil {
			return nil
		}
		collection.Fields.Add(&core.BoolField{Name: "monday_as_start_of_week"})
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("monday_as_start_of_week") == nil {
			return nil
		}
		collection.Fields.RemoveByName("monday_as_start_of_week")
		return app.Save(collection)
	})
}
