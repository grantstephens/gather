package hooks

import (
	"gather/internal/slug"

	"github.com/pocketbase/pocketbase/core"
)

// RegisterPlaceSlugHooks gives every new place a readable, unique slug. Like
// event slugs they are set once on creation and never change, so URLs stay stable.
func RegisterPlaceSlugHooks(app core.App) {
	app.OnRecordCreate("places").BindFunc(func(e *core.RecordEvent) error {
		// Always generated here: places can be created anonymously, so a client
		// must not be able to choose (squat) a slug. Admins can still edit it later.
		e.Record.Set("slug", slug.ForNewPlace(e.App, e.Record.GetString("name")))
		return e.Next()
	})
}
