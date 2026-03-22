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

		// Allow anyone to read settings (instance name, description, etc.)
		// Keep create/update restricted to admins
		publicRule := ""
		collection.ListRule = &publicRule
		collection.ViewRule = &publicRule

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		// Revert to admin-only read access
		adminRule := "@request.auth.role = 'admin'"
		collection.ListRule = &adminRule
		collection.ViewRule = &adminRule

		return app.Save(collection)
	})
}
