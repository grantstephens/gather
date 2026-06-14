package activitypub

import (
	"encoding/json"
	"log"
	"math"
	"time"

	dbx "github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const maxAttempts = 5

// ProcessDeliveryQueue picks up pending ap_delivery_queue records and delivers them.
func ProcessDeliveryQueue(app core.App) {
	nowStr := time.Now().UTC().Format("2006-01-02 15:04:05")
	records, err := app.FindRecordsByFilter(
		"ap_delivery_queue",
		"attempts < {:max} && next_retry <= {:now}",
		"next_retry", 50, 0,
		dbx.Params{"max": maxAttempts, "now": nowStr},
	)
	if err != nil {
		log.Println("ap-delivery: queue query error:", err)
		return
	}

	for _, r := range records {
		inboxURL := r.GetString("inbox_url")
		var activity Activity
		if err := json.Unmarshal([]byte(r.GetString("activity")), &activity); err != nil {
			log.Println("ap-delivery: unmarshal error for", inboxURL, ":", err)
			continue
		}

		err := DeliverActivity(app, activity, inboxURL)
		if err == nil {
			app.Logger().Info("ap-delivery: delivered", "inbox", inboxURL)
			if delErr := app.Delete(r); delErr != nil {
				log.Println("ap-delivery: delete error:", delErr)
			}
		} else {
			attempts := r.GetInt("attempts") + 1
			backoff := time.Duration(5*math.Pow(2, float64(attempts))) * time.Minute
			r.Set("attempts", attempts)
			r.Set("last_error", err.Error())
			r.Set("next_retry", time.Now().Add(backoff).UTC().Format("2006-01-02 15:04:05"))
			if saveErr := app.Save(r); saveErr != nil {
				log.Println("ap-delivery: save error:", saveErr)
			}
			app.Logger().Error("ap-delivery: failed", "inbox", inboxURL, "attempts", attempts, "error", err.Error())
		}
	}
}
