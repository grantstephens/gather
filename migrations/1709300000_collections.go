package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Tags collection
		tags := core.NewBaseCollection("tags")
		tags.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})
		tags.Fields.Add(&core.TextField{
			Name: "color",
		})
		tags.Indexes = []string{
			"CREATE UNIQUE INDEX idx_tags_name ON tags (name)",
		}
		// Empty string = public access, nil = admin only
		publicRule := ""
		tags.ListRule = &publicRule
		tags.ViewRule = &publicRule
		if err := app.Save(tags); err != nil {
			return err
		}

		// Places collection
		places := core.NewBaseCollection("places")
		places.Fields.Add(&core.NumberField{
			Name: "osm_id",
		})
		places.Fields.Add(&core.SelectField{
			Name:   "osm_type",
			Values: []string{"node", "way", "relation"},
		})
		places.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})
		places.Fields.Add(&core.TextField{
			Name: "address",
		})
		places.Fields.Add(&core.NumberField{
			Name:     "latitude",
			Required: true,
		})
		places.Fields.Add(&core.NumberField{
			Name:     "longitude",
			Required: true,
		})
		places.Fields.Add(&core.TextField{
			Name: "city",
		})
		places.Fields.Add(&core.TextField{
			Name: "country_code",
			Max:  2,
		})
		places.Fields.Add(&core.JSONField{
			Name: "osm_data",
		})
		places.ListRule = &publicRule
		places.ViewRule = &publicRule
		if err := app.Save(places); err != nil {
			return err
		}

		// Events collection
		events := core.NewBaseCollection("events")
		events.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
			Max:      200,
		})
		events.Fields.Add(&core.EditorField{
			Name: "description",
		})
		events.Fields.Add(&core.DateField{
			Name:     "start_datetime",
			Required: true,
		})
		events.Fields.Add(&core.DateField{
			Name: "end_datetime",
		})
		events.Fields.Add(&core.RelationField{
			Name:         "place",
			CollectionId: places.Id,
			MaxSelect:    1,
		})
		events.Fields.Add(&core.JSONField{
			Name: "online_locations",
		})
		events.Fields.Add(&core.RelationField{
			Name:         "tags",
			CollectionId: tags.Id,
			MaxSelect:    99,
		})
		events.Fields.Add(&core.FileField{
			Name:      "image",
			MaxSelect: 1,
			MaxSize:   20 * 1024 * 1024,
			MimeTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif"},
			Thumbs:    []string{"100x100", "400x300", "800x600"},
		})
		events.Fields.Add(&core.TextField{
			Name: "author_email",
		})
		events.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"draft", "pending", "published", "cancelled"},
		})
		events.Fields.Add(&core.TextField{
			Name: "recurrence_rule",
		})
		events.Fields.Add(&core.TextField{
			Name: "ap_id",
		})
		events.Fields.Add(&core.TextField{
			Name: "edit_token",
		})
		// API rules (basic rules first, update/delete rules added after author field)
		// Published events visible to all, admins/editors can see all events
		listRule := "status = 'published' || @request.auth.role = 'admin' || @request.auth.role = 'editor'"
		events.ListRule = &listRule
		viewRule := "status = 'published' || @request.auth.role = 'admin' || @request.auth.role = 'editor'"
		events.ViewRule = &viewRule
		createRule := ""
		events.CreateRule = &createRule
		if err := app.Save(events); err != nil {
			return err
		}

		// Add author relation (needs users collection ID) and parent_event self-reference
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err == nil {
			events.Fields.Add(&core.RelationField{
				Name:         "author",
				CollectionId: usersCollection.Id,
				MaxSelect:    1,
			})
		}
		// Add parent_event self-reference now that events has an ID
		events.Fields.Add(&core.RelationField{
			Name:         "parent_event",
			CollectionId: events.Id,
			MaxSelect:    1,
		})
		// Now add update/delete rules (after author field exists)
		updateRule := "@request.auth.role = 'admin' || @request.auth.role = 'editor' || (@request.auth.id = author && status != 'published')"
		events.UpdateRule = &updateRule
		deleteRule := "@request.auth.role = 'admin' || @request.auth.role = 'editor'"
		events.DeleteRule = &deleteRule
		if err := app.Save(events); err != nil {
			return err
		}

		// Settings collection (singleton)
		settings := core.NewBaseCollection("settings")
		settings.Fields.Add(&core.TextField{
			Name: "instance_name",
		})
		settings.Fields.Add(&core.TextField{
			Name: "instance_description",
		})
		settings.Fields.Add(&core.BoolField{
			Name: "allow_anonymous",
		})
		settings.Fields.Add(&core.BoolField{
			Name: "require_moderation",
		})
		settings.Fields.Add(&core.TextField{
			Name: "custom_css",
		})
		settings.Fields.Add(&core.BoolField{
			Name: "ap_enabled",
		})
		settings.Fields.Add(&core.TextField{
			Name: "ap_private_key",
		})
		settings.Fields.Add(&core.TextField{
			Name: "ap_public_key",
		})
		// Only admins can access settings
		settings.ListRule = nil
		settings.ViewRule = nil
		if err := app.Save(settings); err != nil {
			return err
		}

		// AP Followers collection
		apFollowers := core.NewBaseCollection("ap_followers")
		apFollowers.Fields.Add(&core.TextField{
			Name:     "actor_url",
			Required: true,
		})
		apFollowers.Fields.Add(&core.TextField{
			Name:     "inbox_url",
			Required: true,
		})
		apFollowers.Fields.Add(&core.TextField{
			Name: "shared_inbox_url",
		})
		if err := app.Save(apFollowers); err != nil {
			return err
		}

		// AP Delivery Queue collection
		apQueue := core.NewBaseCollection("ap_delivery_queue")
		apQueue.Fields.Add(&core.JSONField{
			Name:     "activity",
			Required: true,
		})
		apQueue.Fields.Add(&core.TextField{
			Name:     "inbox_url",
			Required: true,
		})
		apQueue.Fields.Add(&core.NumberField{
			Name: "attempts",
		})
		apQueue.Fields.Add(&core.TextField{
			Name: "last_error",
		})
		apQueue.Fields.Add(&core.DateField{
			Name: "next_retry",
		})
		if err := app.Save(apQueue); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback
		collections := []string{"ap_delivery_queue", "ap_followers", "settings", "events", "places", "tags"}
		for _, name := range collections {
			col, err := app.FindCollectionByNameOrId(name)
			if err == nil {
				if err := app.Delete(col); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
