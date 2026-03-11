package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add status field to places collection
		places, err := app.FindCollectionByNameOrId("places")
		if err != nil {
			return err
		}

		places.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"pending", "approved"},
		})

		// Update rules: approved visible to all, admins/editors can see all
		listRule := "status = 'approved' || @request.auth.role = 'admin' || @request.auth.role = 'editor'"
		places.ListRule = &listRule
		places.ViewRule = &listRule

		// Allow public creation (anonymous users can create places)
		publicCreate := ""
		places.CreateRule = &publicCreate

		if err := app.Save(places); err != nil {
			return err
		}

		// Set existing places to approved
		existingPlaces, err := app.FindAllRecords("places")
		if err == nil {
			for _, place := range existingPlaces {
				place.Set("status", "approved")
				if err := app.Save(place); err != nil {
					return err
				}
			}
		}

		// Add status field to tags collection
		tags, err := app.FindCollectionByNameOrId("tags")
		if err != nil {
			return err
		}

		tags.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"pending", "approved"},
		})

		// Update rules: approved visible to all, admins/editors can see all
		tags.ListRule = &listRule
		tags.ViewRule = &listRule

		// Allow public creation (anonymous users can create tags)
		tags.CreateRule = &publicCreate

		if err := app.Save(tags); err != nil {
			return err
		}

		// Set existing tags to approved
		existingTags, err := app.FindAllRecords("tags")
		if err == nil {
			for _, tag := range existingTags {
				tag.Set("status", "approved")
				if err := app.Save(tag); err != nil {
					return err
				}
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove status fields and restore original rules
		places, err := app.FindCollectionByNameOrId("places")
		if err == nil {
			places.Fields.RemoveByName("status")
			publicRule := ""
			places.ListRule = &publicRule
			places.ViewRule = &publicRule
			places.CreateRule = nil // admin only
			app.Save(places)
		}

		tags, err := app.FindCollectionByNameOrId("tags")
		if err == nil {
			tags.Fields.RemoveByName("status")
			publicRule := ""
			tags.ListRule = &publicRule
			tags.ViewRule = &publicRule
			tags.CreateRule = nil // admin only
			app.Save(tags)
		}

		return nil
	})
}
