package hooks

import (
	"log"

	"gather/internal/activitypub"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterEventHooks(app core.App, baseURL string) {
	app.OnRecordAfterCreateSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		// Send moderator alert for pending events
		sendModeratorAlert(app, *e.Record, baseURL)
		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		oldStatus := e.Record.Original().GetString("status")
		newStatus := e.Record.GetString("status")

		// Send approval notification
		if oldStatus == "pending" && newStatus == "published" {
			sendApprovalNotification(app, *e.Record, baseURL)
		}

		// Send rejection notification
		if oldStatus == "pending" && newStatus == "cancelled" {
			sendRejectionNotification(app, *e.Record, baseURL)
		}

		if oldStatus != "published" && newStatus == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Create")
			go func() {
				if err := activitypub.QueueDeliveryToFollowers(app, activity); err != nil {
					log.Println("ap: failed to queue Create activity for event", e.Record.Id, ":", err)
				}
			}()
		} else if oldStatus == "published" && newStatus == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Update")
			go func() {
				if err := activitypub.QueueDeliveryToFollowers(app, activity); err != nil {
					log.Println("ap: failed to queue Update activity for event", e.Record.Id, ":", err)
				}
			}()
		}

		return e.Next()
	})

	app.OnRecordAfterDeleteSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("status") == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Delete")
			go func() {
				if err := activitypub.QueueDeliveryToFollowers(app, activity); err != nil {
					log.Println("ap: failed to queue Delete activity for event", e.Record.Id, ":", err)
				}
			}()
		}
		return e.Next()
	})
}
