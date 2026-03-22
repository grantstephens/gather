package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pages")
		if err != nil {
			return err
		}

		collection.Fields.Add(&core.NumberField{
			Name: "sort_order",
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pages")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("sort_order")
		return app.Save(collection)
	})
}
