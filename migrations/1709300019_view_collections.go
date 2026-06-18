package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// tag_counts view: count published upcoming events per tag using json_each unnesting
		tagCounts := core.NewViewCollection("tag_counts")
		tagCounts.ViewQuery = `
			SELECT je.value AS id, je.value AS tag_id, COUNT(*) AS event_count
			FROM events e, json_each(e.tags) je
			WHERE e.status = 'published'
			  AND e.start_datetime >= date('now')
			  AND json_valid(e.tags)
			GROUP BY je.value`
		emptyStr := ""
		tagCounts.ListRule = &emptyStr
		tagCounts.ViewRule = &emptyStr
		if err := app.Save(tagCounts); err != nil {
			return err
		}

		// town_counts view: count distinct upcoming published events per city
		townCounts := core.NewViewCollection("town_counts")
		townCounts.ViewQuery = `
			SELECT p.city AS id, p.city, COUNT(DISTINCT e.id) AS event_count
			FROM events e JOIN places p ON p.id = e.place
			WHERE e.status = 'published'
			  AND e.start_datetime >= date('now')
			  AND p.status = 'approved'
			  AND p.city != ''
			GROUP BY p.city
			ORDER BY event_count DESC`
		townCounts.ListRule = &emptyStr
		townCounts.ViewRule = &emptyStr
		if err := app.Save(townCounts); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Down: remove both view collections
		for _, name := range []string{"tag_counts", "town_counts"} {
			col, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			if err := app.Delete(col); err != nil {
				return err
			}
		}
		return nil
	})
}
