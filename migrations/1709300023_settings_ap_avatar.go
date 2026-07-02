package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		col.Fields.Add(&core.FileField{
			Name:      "ap_avatar",
			MaxSelect: 1,
			MaxSize:   2 * 1024 * 1024,
			MimeTypes: []string{"image/png", "image/jpeg", "image/webp"},
			Thumbs:    []string{"200x200"},
		})
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		col.Fields.RemoveByName("ap_avatar")
		return app.Save(col)
	})
}
