package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		settings, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		settings.Fields.Add(&core.TextField{
			Name: "custom_head",
		})
		return app.Save(settings)
	}, func(app core.App) error {
		settings, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		settings.Fields.RemoveByName("custom_head")
		return app.Save(settings)
	})
}
