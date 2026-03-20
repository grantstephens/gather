package hooks

import (
	"gather/internal/slug"

	"github.com/pocketbase/pocketbase/core"
)

// RegisterSlugHooks generates slugs for new events on create.
// Slugs are never changed after creation to preserve bookmarked URLs.
func RegisterSlugHooks(app core.App) {
	app.OnRecordCreate("events").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("slug") == "" {
			e.Record.Set("slug", slug.Generate(e.Record.GetString("title"), e.Record.Id))
		}
		return e.Next()
	})
}
