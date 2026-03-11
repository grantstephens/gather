package hooks

import (
	"errors"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterModerationHooks(app core.App) {
	// Auto-set status on places create
	app.OnRecordCreateRequest("places").BindFunc(func(e *core.RecordRequestEvent) error {
		if e.HasSuperuserAuth() {
			// Superusers can set any status, default to approved
			if e.Record.GetString("status") == "" {
				e.Record.Set("status", "approved")
			}
			return e.Next()
		}
		// Non-superusers always get pending status
		e.Record.Set("status", "pending")
		return e.Next()
	})

	// Auto-set status on tags create
	app.OnRecordCreateRequest("tags").BindFunc(func(e *core.RecordRequestEvent) error {
		if e.HasSuperuserAuth() {
			// Superusers can set any status, default to approved
			if e.Record.GetString("status") == "" {
				e.Record.Set("status", "approved")
			}
			return e.Next()
		}
		// Non-superusers always get pending status
		e.Record.Set("status", "pending")
		return e.Next()
	})

	// Cascade blocking: prevent event publish if place or tags are pending
	app.OnRecordUpdateRequest("events").BindFunc(func(e *core.RecordRequestEvent) error {
		newStatus := e.Record.GetString("status")

		// Only check when publishing
		if newStatus != "published" {
			return e.Next()
		}

		// Check place status
		placeId := e.Record.GetString("place")
		if placeId != "" {
			place, err := app.FindRecordById("places", placeId)
			if err != nil {
				return errors.New("failed to verify place status")
			}
			if place.GetString("status") != "approved" {
				return errors.New("cannot publish event: the associated place is still pending approval")
			}
		}

		// Check tags status
		tagIds := e.Record.GetStringSlice("tags")
		for _, tagId := range tagIds {
			tag, err := app.FindRecordById("tags", tagId)
			if err != nil {
				return errors.New("failed to verify tag status")
			}
			if tag.GetString("status") != "approved" {
				return errors.New("cannot publish event: one or more tags are still pending approval")
			}
		}

		return e.Next()
	})
}
