package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Followers of the relay actor (Mobilizon instances following relay@host).
// Kept apart from ap_followers: these get Event objects and Announces, not Notes.
func init() {
	m.Register(func(app core.App) error {
		col := core.NewBaseCollection("ap_relay_followers")
		col.Fields.Add(&core.TextField{Name: "actor_url", Required: true})
		col.Fields.Add(&core.TextField{Name: "inbox_url", Required: true})
		col.Fields.Add(&core.TextField{Name: "shared_inbox_url"})
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("ap_relay_followers")
		if err != nil {
			return nil
		}
		return app.Delete(col)
	})
}
