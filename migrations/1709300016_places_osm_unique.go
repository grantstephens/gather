package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Find duplicate (osm_id, osm_type) pairs
		type dupRow struct {
			OsmID   string
			OsmType string
		}
		var dups []dupRow
		err := app.DB().NewQuery(`
			SELECT osm_id, osm_type
			FROM places
			WHERE osm_id IS NOT NULL AND osm_type IS NOT NULL
			GROUP BY osm_id, osm_type
			HAVING COUNT(*) > 1
		`).All(&dups)
		if err != nil {
			return fmt.Errorf("querying duplicate places: %w", err)
		}

		for _, dup := range dups {
			// Find all place records with this osm_id+osm_type, ordered so the
			// one with the most events comes first (ties broken by oldest record).
			type placeRow struct {
				ID         string `db:"id"`
				EventCount int    `db:"event_count"`
			}
			var rows []placeRow
			err := app.DB().NewQuery(`
				SELECT p.id, COUNT(e.id) AS event_count
				FROM places p
				LEFT JOIN events e ON e.place = p.id
				WHERE p.osm_id = {:osmId} AND p.osm_type = {:osmType}
				GROUP BY p.id
				ORDER BY event_count DESC, p.created ASC
			`).Bind(map[string]any{
				"osmId":   dup.OsmID,
				"osmType": dup.OsmType,
			}).All(&rows)
			if err != nil {
				return fmt.Errorf("querying places for osm_id=%s osm_type=%s: %w", dup.OsmID, dup.OsmType, err)
			}
			if len(rows) < 2 {
				continue
			}

			keepID := rows[0].ID
			for _, row := range rows[1:] {
				// Reassign events pointing at the duplicate to the keeper
				_, err := app.DB().NewQuery(`
					UPDATE events SET place = {:keepId} WHERE place = {:dropId}
				`).Bind(map[string]any{
					"keepId": keepID,
					"dropId": row.ID,
				}).Execute()
				if err != nil {
					return fmt.Errorf("reassigning events from place %s to %s: %w", row.ID, keepID, err)
				}

				// Delete the duplicate place record
				dupe, err := app.FindRecordById("places", row.ID)
				if err != nil {
					return fmt.Errorf("finding duplicate place %s: %w", row.ID, err)
				}
				if err := app.Delete(dupe); err != nil {
					return fmt.Errorf("deleting duplicate place %s: %w", row.ID, err)
				}
			}
		}

		// Now safe to add the unique index
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
