package activitypub

import (
	"encoding/json"
	"math"
	"time"

	dbx "github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const maxAttempts = 5

// ProcessDeliveryQueue picks up pending ap_delivery_queue records and delivers them.
func ProcessDeliveryQueue(app core.App) {
	privateKey, err := loadPrivateKey(app)
	if err != nil {
		app.Logger().Error("ap-delivery: failed to load private key", "error", err)
		return
	}

	nowStr := time.Now().UTC().Format("2006-01-02 15:04:05")
	records, err := app.FindRecordsByFilter(
		"ap_delivery_queue",
		"attempts < {:max} && next_retry <= {:now}",
		"next_retry", 50, 0,
		dbx.Params{"max": maxAttempts, "now": nowStr},
	)
	if err != nil {
		app.Logger().Error("ap-delivery: queue query error", "error", err)
		return
	}

	for _, r := range records {
		inboxURL := r.GetString("inbox_url")
		var activity Activity
		if err := json.Unmarshal([]byte(r.GetString("activity")), &activity); err != nil {
			app.Logger().Error("ap-delivery: unmarshal error", "inbox", inboxURL, "error", err)
			r.Set("attempts", maxAttempts)
			r.Set("last_error", "unmarshal error: "+err.Error())
			app.Save(r)
			continue
		}

		err := deliverActivityWithKey(privateKey, activity.Actor+"#main-key", activity, inboxURL)
		if err == nil {
			app.Logger().Info("ap-delivery: delivered", "inbox", inboxURL)
			if delErr := app.Delete(r); delErr != nil {
				app.Logger().Error("ap-delivery: delete error", "inbox", inboxURL, "error", delErr)
			}
		} else {
			attempts := r.GetInt("attempts") + 1
			backoff := time.Duration(5*math.Pow(2, float64(attempts))) * time.Minute
			r.Set("attempts", attempts)
			r.Set("last_error", err.Error())
			r.Set("next_retry", time.Now().Add(backoff).UTC().Format("2006-01-02 15:04:05"))
			if saveErr := app.Save(r); saveErr != nil {
				app.Logger().Error("ap-delivery: save error", "inbox", inboxURL, "error", saveErr)
			}
			app.Logger().Error("ap-delivery: failed", "inbox", inboxURL, "attempts", attempts, "error", err.Error())
		}
	}

	// Prune exhausted records
	exhausted, err := app.FindRecordsByFilter(
		"ap_delivery_queue",
		"attempts >= {:max}",
		"", 0, 0,
		dbx.Params{"max": maxAttempts},
	)
	if err != nil {
		app.Logger().Error("ap-delivery: prune query error", "error", err)
		return
	}
	for _, r := range exhausted {
		if delErr := app.Delete(r); delErr != nil {
			app.Logger().Error("ap-delivery: prune delete error", "error", delErr)
		}
	}
	if len(exhausted) > 0 {
		app.Logger().Info("ap-delivery: pruned exhausted records", "count", len(exhausted))
	}
}
