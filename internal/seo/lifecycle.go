package seo

import (
	"time"

	"gather/internal/recurrence"

	"github.com/pocketbase/pocketbase/core"
)

// State is where an event (or picks post) is in its life as a search result.
type State int

const (
	Upcoming State = iota // not over yet: indexable, in the sitemap
	Ended                 // over, within the grace period: indexable, "ended" banner
	Retired               // over for longer than RetireAfter: noindex, not in the sitemap
)

const (
	// RetireAfter is how long an ended event stays indexable.
	RetireAfter     = 7 * 24 * time.Hour
	defaultDuration = 3 * time.Hour
)

// LifecycleOf classifies a single occurrence. Ended-at is end, else start+3h.
func LifecycleOf(start, end, now time.Time) State {
	if start.IsZero() {
		return Upcoming
	}
	endedAt := start.Add(defaultDuration)
	if !end.IsZero() && end.After(start) {
		endedAt = end
	}
	switch {
	case now.Before(endedAt):
		return Upcoming
	case now.Before(endedAt.Add(RetireAfter)):
		return Ended
	default:
		return Retired
	}
}

// EventLifecycle is LifecycleOf for a record. A recurring record counts as
// upcoming while it still has a future occurrence.
func EventLifecycle(rec *core.Record, now time.Time) State {
	start := rec.GetDateTime("start_datetime").Time()
	end := rec.GetDateTime("end_datetime").Time()
	if rule := rec.GetString("recurrence_rule"); rule != "" && !start.IsZero() {
		times, err := recurrence.Expand(rule, start, now, now.AddDate(2, 0, 0))
		if err == nil {
			times = recurrence.FilterExceptions(times, rec.GetString("recurrence_exceptions"))
			if len(times) > 0 {
				return Upcoming
			}
			// Finished: judge by the last occurrence, not the first date.
			past, perr := recurrence.Expand(rule, start, start, now)
			if perr == nil {
				past = recurrence.FilterExceptions(past, rec.GetString("recurrence_exceptions"))
				if len(past) > 0 {
					last := past[len(past)-1]
					occEnd := time.Time{}
					if !end.IsZero() && end.After(start) {
						occEnd = last.Add(end.Sub(start))
					}
					return LifecycleOf(last, occEnd, now)
				}
			}
		}
	}
	return LifecycleOf(start, end, now)
}

// PickLifecycle: a picks post is as alive as its liveliest event. A post with no
// resolvable events is never retired (seasonal/evergreen posts).
func PickLifecycle(app core.App, pick *core.Record, now time.Time) State {
	best, found := Retired, false
	for _, id := range pick.GetStringSlice("events") {
		ev, err := app.FindRecordById("events", id)
		if err != nil {
			continue
		}
		found = true
		if s := EventLifecycle(ev, now); s < best {
			best = s
		}
	}
	if !found {
		return Upcoming
	}
	return best
}
