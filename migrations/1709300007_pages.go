package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		pages := core.NewBaseCollection("pages")

		pages.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
		})
		pages.Fields.Add(&core.TextField{
			Name:     "slug",
			Required: true,
		})
		pages.Fields.Add(&core.EditorField{
			Name: "content",
		})
		pages.Fields.Add(&core.BoolField{
			Name: "show_in_nav",
		})
		pages.Fields.Add(&core.BoolField{
			Name: "show_in_footer",
		})

		pages.Indexes = []string{
			"CREATE UNIQUE INDEX idx_pages_slug ON pages (slug)",
		}

		publicRule := ""
		pages.ListRule = &publicRule
		pages.ViewRule = &publicRule

		adminRule := "@request.auth.role = 'admin'"
		pages.CreateRule = &adminRule
		pages.UpdateRule = &adminRule
		pages.DeleteRule = &adminRule

		return app.Save(pages)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("pages")
		if err != nil {
			return nil // already gone
		}
		return app.Delete(col)
	})
}
