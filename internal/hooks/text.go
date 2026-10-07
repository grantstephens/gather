package hooks

import (
	"gather/internal/textutil"

	"github.com/pocketbase/pocketbase/core"
)

// RegisterTextHooks stores text decoded: scrapers HTML-escape titles
// ("Potter&#039;s"), which then got double-escaped on render and mangled slugs.
func RegisterTextHooks(app core.App) {
	handler := func(e *core.RecordEvent) error {
		textutil.DecodeRecordText(e.Record)
		return e.Next()
	}
	app.OnRecordCreate("events", "places", "tags").BindFunc(handler)
	app.OnRecordUpdate("events", "places", "tags").BindFunc(handler)
}
