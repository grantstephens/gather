package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("picks")
		if err != nil {
			return err
		}
		col.Fields.Add(&core.BoolField{
			Name: "ap_announced",
		})
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("picks")
		if err != nil {
			return err
		}
		if f := col.Fields.GetByName("ap_announced"); f != nil {
			col.Fields.RemoveById(f.GetId())
		}
		return app.Save(col)
	})
}
