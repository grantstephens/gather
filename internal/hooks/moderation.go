package hooks

import (
	"errors"
	"fmt"
	"hash/fnv"

	"github.com/pocketbase/pocketbase/core"
)

// hashColor returns a deterministic HSL color string derived from the tag name.
func hashColor(name string) string {
	h := fnv.New32a()
	h.Write([]byte(name))
	hue := h.Sum32() % 360
	return fmt.Sprintf("hsl(%d, 65%%, 45%%)", hue)
}

func RegisterModerationHooks(app core.App) {
	// Backfill colors for any existing tags that have none (non-blocking)
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		go func() {
			tags, err := e.App.FindAllRecords("tags")
			if err == nil {
				for _, tag := range tags {
					if tag.GetString("color") == "" {
						tag.Set("color", hashColor(tag.GetString("name")))
						_ = e.App.Save(tag)
					}
				}
			}
		}()
		return e.Next()
	})

	// Auto-assign color to tags on create if not provided
	app.OnRecordCreateRequest("tags").BindFunc(func(e *core.RecordRequestEvent) error {
		if e.Record.GetString("color") == "" {
			e.Record.Set("color", hashColor(e.Record.GetString("name")))
		}
		return e.Next()
	})

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

	// Auto-approve place and tags when an admin/editor creates or updates an event
	approveEventDeps := func(e *core.RecordRequestEvent) {
		isPrivileged := e.HasSuperuserAuth() || (e.Auth != nil && e.Auth.GetString("role") == "admin")
		if !isPrivileged {
			return
		}
		if placeId := e.Record.GetString("place"); placeId != "" {
			if place, err := app.FindRecordById("places", placeId); err == nil && place.GetString("status") == "pending" {
				place.Set("status", "approved")
				_ = app.Save(place)
			}
		}
		for _, tagId := range e.Record.GetStringSlice("tags") {
			if tag, err := app.FindRecordById("tags", tagId); err == nil && tag.GetString("status") == "pending" {
				tag.Set("status", "approved")
				_ = app.Save(tag)
			}
		}
	}

	app.OnRecordCreateRequest("events").BindFunc(func(e *core.RecordRequestEvent) error {
		approveEventDeps(e)
		return e.Next()
	})

	app.OnRecordUpdateRequest("events").BindFunc(func(e *core.RecordRequestEvent) error {
		approveEventDeps(e)
		return e.Next()
	})

	// Cascade blocking: prevent event publish if place or tags are pending
	app.OnRecordUpdateRequest("events").BindFunc(func(e *core.RecordRequestEvent) error {
		newStatus := e.Record.GetString("status")

		// Only check when publishing
		if newStatus != "published" {
			return e.Next()
		}

		// Superusers and admin/editor role users can bypass cascade checks
		if e.HasSuperuserAuth() {
			return e.Next()
		}
		if e.Auth != nil {
			role := e.Auth.GetString("role")
			if role == "admin" || role == "editor" {
				return e.Next()
			}
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
