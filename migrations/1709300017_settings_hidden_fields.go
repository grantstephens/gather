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
		if f := col.Fields.GetByName("ap_private_key"); f != nil {
			f.(*core.TextField).Hidden = true
		}
		if f := col.Fields.GetByName("ap_public_key"); f != nil {
			f.(*core.TextField).Hidden = true
		}
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		if f := col.Fields.GetByName("ap_private_key"); f != nil {
			f.(*core.TextField).Hidden = false
		}
		if f := col.Fields.GetByName("ap_public_key"); f != nil {
			f.(*core.TextField).Hidden = false
		}
		return app.Save(col)
	})
}
