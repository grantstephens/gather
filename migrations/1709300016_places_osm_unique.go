package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Reassign events from duplicate places to the keeper (MIN id per group)
		_, err := app.DB().NewQuery(`
			UPDATE events SET place = (
				SELECT MIN(p.id) FROM places p
				WHERE p.osm_id = (SELECT osm_id FROM places WHERE id = events.place)
				  AND p.osm_type = (SELECT osm_type FROM places WHERE id = events.place)
			)
			WHERE place IN (
				SELECT id FROM places
				WHERE osm_id IS NOT NULL AND osm_type IS NOT NULL
				  AND id NOT IN (
					SELECT MIN(id) FROM places
					WHERE osm_id IS NOT NULL AND osm_type IS NOT NULL
					GROUP BY osm_id, osm_type
				  )
			)
		`).Execute()
		if err != nil {
			return fmt.Errorf("reassigning events from duplicate places: %w", err)
		}

		// Delete duplicate places, keeping one per (osm_id, osm_type)
		_, err = app.DB().NewQuery(`
			DELETE FROM places
			WHERE osm_id IS NOT NULL AND osm_type IS NOT NULL
			  AND id NOT IN (
				SELECT MIN(id) FROM places
				WHERE osm_id IS NOT NULL AND osm_type IS NOT NULL
				GROUP BY osm_id, osm_type
			  )
		`).Execute()
		if err != nil {
			return fmt.Errorf("deleting duplicate places: %w", err)
		}

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
