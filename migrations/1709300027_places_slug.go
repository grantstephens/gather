package migrations

import (
	"gather/internal/slug"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Places were addressed by record id (/place/z7k4c8qg71zto5f). Give them
// readable slugs (/place/pitlochry-festival-theatre); old id URLs keep working
// via a 301 in main.go.
func init() {
	m.Register(func(app core.App) error {
		places, err := app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}
		if places.Fields.GetByName("slug") == nil {
			places.Fields.Add(&core.TextField{Name: "slug"})
		}
		places.AddIndex("idx_places_slug", true, "slug", "slug != ''")
		if err := app.Save(places); err != nil {
			return err
		}
		return slug.BackfillPlaces(app)
	}, func(app core.App) error {
		places, err := app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}
		places.RemoveIndex("idx_places_slug")
		places.Fields.RemoveByName("slug")
		return app.Save(places)
	})
}
