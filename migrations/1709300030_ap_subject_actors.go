package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Per-place and per-town ActivityPub actors, created lazily on first request.
// ap_actors holds each actor's keypair; ap_subject_followers holds who follows which.
// `subject` is "place:{recordId}" or "town:{citySlug}".
func init() {
	m.Register(func(app core.App) error {
		actors := core.NewBaseCollection("ap_actors")
		actors.Fields.Add(&core.TextField{Name: "subject", Required: true})
		actors.Fields.Add(&core.TextField{Name: "public_key", Required: true})
		actors.Fields.Add(&core.TextField{Name: "private_key", Required: true, Hidden: true})
		actors.AddIndex("idx_ap_actors_subject", true, "subject", "")
		if err := app.Save(actors); err != nil {
			return err
		}

		followers := core.NewBaseCollection("ap_subject_followers")
		followers.Fields.Add(&core.TextField{Name: "subject", Required: true})
		followers.Fields.Add(&core.TextField{Name: "actor_url", Required: true})
		followers.Fields.Add(&core.TextField{Name: "inbox_url", Required: true})
		followers.Fields.Add(&core.TextField{Name: "shared_inbox_url"})
		followers.AddIndex("idx_ap_subject_followers_pair", true, "subject, actor_url", "")
		return app.Save(followers)
	}, func(app core.App) error {
		for _, name := range []string{"ap_subject_followers", "ap_actors"} {
			if col, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(col); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
