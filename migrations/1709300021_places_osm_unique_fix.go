package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Fix the osm_unique index to exclude zero osm_id and empty osm_type.
// In SQLite, 0 IS NOT NULL and '' IS NOT NULL, so the previous index
// treated default zero values as unique constraints, blocking creation of
// multiple places that lack real OSM data.
func init() {
	oldIndex := "CREATE UNIQUE INDEX idx_places_osm_unique ON places (osm_id, osm_type) WHERE osm_id IS NOT NULL AND osm_type IS NOT NULL"
	newIndex := "CREATE UNIQUE INDEX idx_places_osm_unique ON places (osm_id, osm_type) WHERE osm_id IS NOT NULL AND osm_type IS NOT NULL AND osm_id != 0 AND osm_type != ''"

	m.Register(func(app core.App) error {
		places, err := app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}
		for i, idx := range places.Indexes {
			if idx == oldIndex {
				places.Indexes[i] = newIndex
				return app.Save(places)
			}
		}
		// Index not found with old definition — check if new one already exists
		for _, idx := range places.Indexes {
			if idx == newIndex {
				return nil // already correct
			}
		}
		// Neither found — add new one
		places.Indexes = append(places.Indexes, newIndex)
		return app.Save(places)
	}, func(app core.App) error {
		places, err := app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}
		for i, idx := range places.Indexes {
			if idx == newIndex {
				places.Indexes[i] = oldIndex
				return app.Save(places)
			}
		}
		return nil
	})
}
