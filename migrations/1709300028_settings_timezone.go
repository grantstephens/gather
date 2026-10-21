package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// IANA timezone events are shown in. Federated Event objects need an explicit
// zone so remote calendars render the right local time.
func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		if col.Fields.GetByName("timezone") == nil {
			col.Fields.Add(&core.TextField{Name: "timezone", Max: 64})
		}
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		col.Fields.RemoveByName("timezone")
		return app.Save(col)
	})
}
