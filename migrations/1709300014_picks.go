package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("events")
		if err != nil {
			return err
		}

		picks := core.NewBaseCollection("picks")

		picks.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
		})
		picks.Fields.Add(&core.TextField{
			Name:     "slug",
			Required: true,
		})
		picks.Fields.Add(&core.EditorField{
			Name: "blurb",
		})
		picks.Fields.Add(&core.RelationField{
			Name:         "events",
			CollectionId: events.Id,
			MaxSelect:    999,
		})
		picks.Fields.Add(&core.BoolField{
			Name: "hidden",
		})
		picks.Fields.Add(&core.DateField{
			Name: "start_date",
		})
		picks.Fields.Add(&core.AutodateField{
			Name:     "created",
			OnCreate: true,
		})
		picks.Fields.Add(&core.AutodateField{
			Name:     "updated",
			OnCreate: true,
			OnUpdate: true,
		})

		picks.Indexes = []string{
			"CREATE UNIQUE INDEX idx_picks_slug ON picks (slug)",
		}

		publicRule := ""
		editorRule := `@request.auth.role = 'admin' || @request.auth.role = 'editor'`

		picks.ListRule = &publicRule
		picks.ViewRule = &publicRule
		picks.CreateRule = &editorRule
		picks.UpdateRule = &editorRule
		picks.DeleteRule = &editorRule

		return app.Save(picks)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("picks")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
