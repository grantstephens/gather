package activitypub

import (
	"time"

	"gather/internal/recurrence"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// EventLeadDays is how many days before an event it is posted to the Fediverse
// (settings.ap_event_lead_days). 0 means "as soon as it is published".
func EventLeadDays(app core.App) int {
	rec, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		return 0
	}
	if n := rec.GetInt("ap_event_lead_days"); n > 0 {
		return n
	}
	return 0
}

// PostAt is 09:00 (in loc) on the day `leadDays` before the event's local day.
func PostAt(start time.Time, leadDays int, loc *time.Location) time.Time {
	local := start.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day()-leadDays, 9, 0, 0, 0, loc)
}

// DueNow reports whether an event starting at `start` should already have been
// posted. With no lead time, or no known start, that is always true.
func DueNow(start time.Time, leadDays int, now time.Time, loc *time.Location) bool {
	if leadDays <= 0 || start.IsZero() {
		return true
	}
	return !now.Before(PostAt(start, leadDays, loc))
}

// StampPosted atomically marks the event as posted. It succeeds once, and goes
// around the record save so `updated` (sitemap lastmod, JSON-LD) is untouched.
func StampPosted(app core.App, eventID string, at time.Time) (bool, error) {
	res, err := app.DB().Update(
		"events",
		dbx.Params{"ap_posted_at": stampTime(at)},
		dbx.HashExp{"id": eventID, "ap_posted_at": ""},
	).Execute()
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func stampTime(at time.Time) string {
	dt, _ := types.ParseDateTime(at.UTC())
	return dt.String()
}

// ClearPosted records that followers no longer have the post (a Delete was sent).
func ClearPosted(app core.App, eventID string) error {
	_, err := app.DB().Update("events", dbx.Params{"ap_posted_at": ""}, dbx.HashExp{"id": eventID}).Execute()
	return err
}

// IsPosted reports whether followers currently have this event's post, read
// from the database rather than the (possibly stale) in-memory record.
func IsPosted(app core.App, eventID string) bool {
	rec, err := app.FindRecordById("events", eventID)
	return err == nil && !rec.GetDateTime("ap_posted_at").IsZero()
}

// AnnounceEvent posts the event to followers once: it stamps first (so a crash
// or a concurrent caller can never double-post) and then queues the Create.
// It returns false when the event was already posted.
func AnnounceEvent(app core.App, baseURL string, ev *core.Record, now time.Time) (bool, error) {
	ok, err := StampPosted(app, ev.Id, now)
	if err != nil || !ok {
		return false, err
	}
	if err := QueueDeliveryToFollowers(app, CreateActivityForEvent(ev, baseURL, "Create")); err != nil {
		return true, err
	}
	return true, nil
}

// nextStart is the date the post timing is based on: the next occurrence for a
// recurring event, otherwise the start. The zero time means "nothing upcoming".
func nextStart(ev *core.Record, now time.Time, leadDays int) time.Time {
	start := ev.GetDateTime("start_datetime").Time()
	rule := ev.GetString("recurrence_rule")
	if rule == "" {
		if start.Before(now) {
			return time.Time{}
		}
		return start
	}
	times, err := recurrence.Expand(rule, start, now, now.AddDate(0, 0, leadDays+2))
	if err != nil {
		return time.Time{}
	}
	times = recurrence.FilterExceptions(times, ev.GetString("recurrence_exceptions"))
	if len(times) == 0 {
		return time.Time{}
	}
	return times[0]
}

// PostDueEvents is the scheduler tick: it posts every published, not yet posted
// event whose post time has arrived. It returns how many were posted.
func PostDueEvents(app core.App, baseURL string, now time.Time, loc *time.Location) (int, error) {
	lead := EventLeadDays(app)
	cutoff := now.UTC().Format("2006-01-02 15:04:05")
	events, err := app.FindRecordsByFilter(
		"events",
		"status = 'published' && ap_posted_at = '' && (start_datetime > {:now} || recurrence_rule != '')",
		"start_datetime", 500, 0,
		dbx.Params{"now": cutoff},
	)
	if err != nil {
		return 0, err
	}
	posted := 0
	for _, ev := range events {
		start := nextStart(ev, now, lead)
		if start.IsZero() || !DueNow(start, lead, now, loc) {
			continue
		}
		ok, err := AnnounceEvent(app, baseURL, ev, now)
		if err != nil {
			app.Logger().Error("ap-event-schedule: announce failed", "event_id", ev.Id, "error", err)
		}
		if ok {
			posted++
		}
	}
	if posted > 0 {
		app.Logger().Info("ap-event-schedule: posted events", "count", posted)
	}
	return posted, nil
}

// PublishEvent is called when an event becomes published (or is edited while
// still unposted). With no lead time it posts straight away; otherwise only if
// the post time has already arrived, and the scheduler picks it up later.
// Events that are already over are never posted. It is a no-op when the event
// was already posted.
func PublishEvent(app core.App, baseURL string, ev *core.Record, now time.Time, loc *time.Location) (bool, error) {
	lead := EventLeadDays(app)
	if lead > 0 {
		start := nextStart(ev, now, lead)
		if start.IsZero() || !DueNow(start, lead, now, loc) {
			return false, nil
		}
	}
	return AnnounceEvent(app, baseURL, ev, now)
}
