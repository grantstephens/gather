package hooks

import (
	"gather/internal/activitypub"

	"github.com/pocketbase/pocketbase/core"
)

// queueRelay sends the event to relay followers (Mobilizon) in the Event shape.
func queueRelay(app core.App, baseURL string, event *core.Record, activityType string) {
	go func() {
		if err := activitypub.QueueRelayEvent(app, baseURL, event, activityType); err != nil {
			app.Logger().Error("ap-relay: failed to queue "+activityType, "event_id", event.Id, "error", err)
		}
	}()
}

func RegisterEventHooks(app core.App, baseURL string) {
	app.OnRecordAfterCreateSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		sendModeratorAlert(app, *e.Record, baseURL)
		if e.Record.GetString("status") == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Create")
			go func() {
				if err := activitypub.QueueDeliveryToFollowers(app, activity); err != nil {
					app.Logger().Error("ap: failed to queue Create activity", "event_id", e.Record.Id, "error", err)
				}
			}()
			queueRelay(app, baseURL, e.Record, "Create")
		}
		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		oldStatus := e.Record.Original().GetString("status")
		newStatus := e.Record.GetString("status")

		if oldStatus == "pending" && newStatus == "published" {
			sendApprovalNotification(app, *e.Record, baseURL)
		}

		if oldStatus == "pending" && newStatus == "cancelled" {
			sendRejectionNotification(app, *e.Record, baseURL)
		}

		if oldStatus != "published" && newStatus == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Create")
			go func() {
				if err := activitypub.QueueDeliveryToFollowers(app, activity); err != nil {
					app.Logger().Error("ap: failed to queue Create activity", "event_id", e.Record.Id, "error", err)
				}
			}()
			queueRelay(app, baseURL, e.Record, "Create")
		} else if oldStatus == "published" && newStatus == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Update")
			go func() {
				if err := activitypub.QueueDeliveryToFollowers(app, activity); err != nil {
					app.Logger().Error("ap: failed to queue Update activity", "event_id", e.Record.Id, "error", err)
				}
			}()
			queueRelay(app, baseURL, e.Record, "Update")
		} else if oldStatus == "published" && newStatus != "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Delete")
			go func() {
				if err := activitypub.QueueDeliveryToFollowers(app, activity); err != nil {
					app.Logger().Error("ap: failed to queue Delete activity", "event_id", e.Record.Id, "error", err)
				}
			}()
			if newStatus == "cancelled" {
				queueRelay(app, baseURL, e.Record, "Update")
			} else {
				queueRelay(app, baseURL, e.Record, "Delete")
			}
		}

		return e.Next()
	})

	app.OnRecordAfterDeleteSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("status") == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Delete")
			go func() {
				if err := activitypub.QueueDeliveryToFollowers(app, activity); err != nil {
					app.Logger().Error("ap: failed to queue Delete activity", "event_id", e.Record.Id, "error", err)
				}
			}()
			queueRelay(app, baseURL, e.Record, "Delete")
		}
		return e.Next()
	})
}
