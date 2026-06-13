package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		picks, err := app.FindCollectionByNameOrId("picks")
		if err != nil {
			// Collection doesn't exist yet — handled by 1709300015_picks
			return nil
		}

		if picks.Fields.GetByName("start_date") != nil {
			return nil // Already has the field
		}

		picks.Fields.Add(&core.DateField{
			Name: "start_date",
		})

		return app.Save(picks)
	}, func(app core.App) error {
		picks, err := app.FindCollectionByNameOrId("picks")
		if err != nil {
			return nil
		}
		picks.Fields.RemoveByName("start_date")
		return app.Save(picks)
	})
}
