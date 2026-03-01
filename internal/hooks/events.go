package hooks

import (
	"gather/internal/activitypub"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterEventHooks(app core.App, baseURL string) {
	app.OnRecordAfterUpdateSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		oldStatus := e.Record.Original().GetString("status")
		newStatus := e.Record.GetString("status")

		if oldStatus != "published" && newStatus == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Create")
			go activitypub.QueueDeliveryToFollowers(app, activity)
		} else if oldStatus == "published" && newStatus == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Update")
			go activitypub.QueueDeliveryToFollowers(app, activity)
		}

		return e.Next()
	})

	app.OnRecordAfterDeleteSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("status") == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Delete")
			go activitypub.QueueDeliveryToFollowers(app, activity)
		}
		return e.Next()
	})
}
