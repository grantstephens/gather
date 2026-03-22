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

		// Rename the logo field to favicon
		logoField := collection.Fields.GetByName("logo")
		if logoField != nil {
			// Remove old field and add new one with same config
			collection.Fields.RemoveByName("logo")

			faviconField := &core.FileField{
				Name:      "favicon",
				MaxSelect: 1,
				MaxSize:   2 * 1024 * 1024, // 2MB
				MimeTypes: []string{"image/png", "image/jpeg", "image/svg+xml", "image/webp"},
				Thumbs:    []string{"100x100", "200x200"},
			}
			collection.Fields.Add(faviconField)
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		// Migrate data from logo column to favicon column
		_, err = app.DB().NewQuery(
			"UPDATE settings SET favicon = logo WHERE logo IS NOT NULL AND logo != ''",
		).Execute()

		return err
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		// Reverse: rename favicon back to logo
		faviconField := collection.Fields.GetByName("favicon")
		if faviconField != nil {
			collection.Fields.RemoveByName("favicon")

			logoField := &core.FileField{
				Name:      "logo",
				MaxSelect: 1,
				MaxSize:   2 * 1024 * 1024,
				MimeTypes: []string{"image/png", "image/jpeg", "image/svg+xml", "image/webp"},
				Thumbs:    []string{"100x100", "200x200"},
			}
			collection.Fields.Add(logoField)
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		_, err = app.DB().NewQuery(
			"UPDATE settings SET logo = favicon WHERE favicon IS NOT NULL AND favicon != ''",
		).Execute()

		return err
	})
}
