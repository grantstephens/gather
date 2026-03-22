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

		collection.Fields.Add(&core.TextField{
			Name:    "subtitle",
			Max:     60,
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("subtitle")

		return app.Save(collection)
	})
}
