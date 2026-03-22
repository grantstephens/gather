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

		// Step 1: Add favicon field (keep logo for now so both columns exist)
		faviconField := &core.FileField{
			Name:      "favicon",
			MaxSelect: 1,
			MaxSize:   2 * 1024 * 1024, // 2MB
			MimeTypes: []string{"image/png", "image/jpeg", "image/svg+xml", "image/webp"},
			Thumbs:    []string{"100x100", "200x200"},
		}
		collection.Fields.Add(faviconField)

		if err := app.Save(collection); err != nil {
			return err
		}

		// Step 2: Copy data while both columns exist
		_, err = app.DB().NewQuery(
			"UPDATE settings SET favicon = logo WHERE logo IS NOT NULL AND logo != ''",
		).Execute()
		if err != nil {
			return err
		}

		// Step 3: Remove logo field
		collection, err = app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("logo")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		// Step 1: Add logo field back
		logoField := &core.FileField{
			Name:      "logo",
			MaxSelect: 1,
			MaxSize:   2 * 1024 * 1024,
			MimeTypes: []string{"image/png", "image/jpeg", "image/svg+xml", "image/webp"},
			Thumbs:    []string{"100x100", "200x200"},
		}
		collection.Fields.Add(logoField)

		if err := app.Save(collection); err != nil {
			return err
		}

		// Step 2: Copy data back
		_, err = app.DB().NewQuery(
			"UPDATE settings SET logo = favicon WHERE favicon IS NOT NULL AND favicon != ''",
		).Execute()
		if err != nil {
			return err
		}

		// Step 3: Remove favicon field
		collection, err = app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("favicon")

		return app.Save(collection)
	})
}
