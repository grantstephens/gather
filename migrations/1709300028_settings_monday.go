package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Get settings collection
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		
		collection.Fields.Add(&core.BoolField{
			Name: "monday_as_start_of_week",
		})

		// Save collection
		if err := app.Save(collection); err != nil {
			return err
		}

		return nil
				
	}, func(app core.App) error {
		// Rollback: remove logo field and revert rules to nil (superuser only)
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		
		collection.Fields.RemoveByName("monday_as_start_of_week")
		
		// Save collection
		if err := app.Save(collection); err != nil {
			return err
		}

		return nil
	})
}
