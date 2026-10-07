package migrations

import (
	"gather/internal/textutil"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Scrapers stored HTML-escaped text ("Potter&#039;s"). New saves are decoded by
// hooks.RegisterTextHooks; this fixes the rows already in the database.
// Slugs are left alone so bookmarked URLs keep working.
func init() {
	m.Register(func(app core.App) error {
		return textutil.DecodeAllExisting(app)
	}, func(app core.App) error {
		return nil // decoding is not reversible, and not worth reversing
	})
}
