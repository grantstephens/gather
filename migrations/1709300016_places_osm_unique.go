package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		places, err := app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}

		places.Indexes = append(places.Indexes,
			"CREATE UNIQUE INDEX idx_places_osm_unique ON places (osm_id, osm_type) WHERE osm_id IS NOT NULL AND osm_type IS NOT NULL",
		)

		return app.Save(places)
	}, func(app core.App) error {
		places, err := app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}

		filtered := places.Indexes[:0]
		for _, idx := range places.Indexes {
			if idx != "CREATE UNIQUE INDEX idx_places_osm_unique ON places (osm_id, osm_type) WHERE osm_id IS NOT NULL AND osm_type IS NOT NULL" {
				filtered = append(filtered, idx)
			}
		}
		places.Indexes = filtered

		return app.Save(places)
	})
}
