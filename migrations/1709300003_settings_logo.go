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

		// Add logo field
		logoField := &core.FileField{
			Name:      "logo",
			MaxSelect: 1,
			MaxSize:   2 * 1024 * 1024, // 2MB
			MimeTypes: []string{"image/png", "image/jpeg", "image/svg+xml", "image/webp"},
			Thumbs:    []string{"100x100", "200x200"},
		}

		collection.Fields.Add(logoField)

		// Update rules to allow admin access
		adminRule := "@request.auth.role = 'admin'"
		collection.ListRule = &adminRule
		collection.ViewRule = &adminRule
		collection.CreateRule = &adminRule
		collection.UpdateRule = &adminRule

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

		// Remove logo field
		collection.Fields.RemoveByName("logo")

		// Revert rules to nil (superuser only)
		collection.ListRule = nil
		collection.ViewRule = nil
		collection.CreateRule = nil
		collection.UpdateRule = nil

		// Save collection
		if err := app.Save(collection); err != nil {
			return err
		}

		return nil
	})
}
