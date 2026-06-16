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
		// Only the private key must be hidden — it must never be exposed via API.
		// The public key is intentionally public (served at /ap/actor) so we leave it visible.
		if f, ok := col.Fields.GetByName("ap_private_key").(*core.TextField); ok {
			f.Hidden = true
		}
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		if f, ok := col.Fields.GetByName("ap_private_key").(*core.TextField); ok {
			f.Hidden = false
		}
		return app.Save(col)
	})
}
