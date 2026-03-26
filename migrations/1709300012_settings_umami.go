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
			Name: "umami_src",
		})
		collection.Fields.Add(&core.TextField{
			Name: "umami_website_id",
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("umami_src")
		collection.Fields.RemoveByName("umami_website_id")

		return app.Save(collection)
	})
}
