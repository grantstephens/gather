package cron

import (
	"log"
	"time"

	dbx "github.com/pocketbase/dbx"
	"gather/internal/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// Register registers all housekeeping cron jobs on the app.
func Register(app core.App, baseURL string) {
	// Clean up exhausted AP delivery queue records older than 7 days
	app.Cron().Add("ap-queue-cleanup", "0 3 * * *", func() {
		cutoff := time.Now().AddDate(0, 0, -7)
		records, err := app.FindRecordsByFilter(
			"ap_delivery_queue",
			"attempts >= 5 && created < {:cutoff}",
			"-created", 500, 0,
			dbx.Params{"cutoff": cutoff.UTC().Format("2006-01-02 15:04:05")},
		)
		if err != nil {
			log.Println("ap-queue-cleanup: query error:", err)
			return
		}
		for _, r := range records {
			if err := app.Delete(r); err != nil {
				log.Println("ap-queue-cleanup: delete error:", err)
			}
		}
		app.Logger().Info("ap-queue-cleanup: pruned exhausted records", "count", len(records))
	})

	// Announce this weekend's picks to ActivityPub on Monday and Tuesday mornings
	app.Cron().Add("picks-ap-announcement", "0 9 * * 1,2", func() {
		if baseURL == "" {
			return
		}
		settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
		if err != nil || !settings.GetBool("ap_enabled") {
			return
		}

		now := time.Now()
		// Upcoming Saturday (days until Saturday from Mon=5, Tue=4)
		daysUntilSat := (6 - int(now.Weekday()) + 7) % 7
		saturday := now.AddDate(0, 0, daysUntilSat)
		sunday := saturday.AddDate(0, 0, 1)

		satStr := saturday.Format("2006-01-02")
		sunStr := sunday.Format("2006-01-02 23:59:59")

		picks, err := app.FindRecordsByFilter(
			"picks",
			"hidden = false && ap_announced = false && start_date >= {:sat} && start_date <= {:sun}",
			"start_date", 10, 0,
			dbx.Params{"sat": satStr, "sun": sunStr},
		)
		if err != nil {
			log.Println("picks-ap-announcement: query error:", err)
			return
		}

		for _, p := range picks {
			activity := activitypub.CreateActivityForPicks(p, baseURL)
			if err := activitypub.QueueDeliveryToFollowers(app, activity); err != nil {
				log.Println("picks-ap-announcement: queue error:", err)
				continue
			}
			p.Set("ap_announced", true)
			if err := app.Save(p); err != nil {
				log.Println("picks-ap-announcement: save error:", err)
			}
		}
		if len(picks) > 0 {
			app.Logger().Info("picks-ap-announcement: announced picks", "count", len(picks))
		}
	})

	// Cancel events stuck in pending for more than 90 days (if require_moderation is enabled)
	app.Cron().Add("stale-pending-cleanup", "0 4 * * *", func() {
		// Check if moderation is enabled
		settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
		if err != nil || !settings.GetBool("require_moderation") {
			return
		}
		cutoff := time.Now().AddDate(0, 0, -90)
		records, err := app.FindRecordsByFilter(
			"events",
			"status = 'pending' && created < {:cutoff}",
			"-created", 500, 0,
			dbx.Params{"cutoff": cutoff.UTC().Format("2006-01-02 15:04:05")},
		)
		if err != nil {
			log.Println("stale-pending-cleanup: query error:", err)
			return
		}
		for _, r := range records {
			r.Set("status", "cancelled")
			if err := app.Save(r); err != nil {
				log.Println("stale-pending-cleanup: save error:", err)
			}
		}
		if len(records) > 0 {
			app.Logger().Info("stale-pending-cleanup: cancelled stale pending events", "count", len(records))
		}
	})
}
