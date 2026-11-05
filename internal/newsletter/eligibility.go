package newsletter

import (
	"sort"
	"time"

	"gather/internal/seo"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// weekendWindow returns the Friday and Sunday (date-only, in now's location)
// of the weekend a newsletter sent at `now` is about: the coming one Mon-Thu,
// the current one Fri-Sun.
func weekendWindow(now time.Time) (fri, sun time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var offset int
	switch wd := int(today.Weekday()); {
	case wd >= 1 && wd <= 4: // Mon-Thu -> coming Friday
		offset = 5 - wd
	case wd == 5:
		offset = 0
	case wd == 6:
		offset = -1
	default: // Sunday
		offset = -2
	}
	fri = today.AddDate(0, 0, offset)
	sun = fri.AddDate(0, 0, 2)
	return fri, sun
}

// EligiblePick returns the picks post to email, or nil when there is none.
// Eligible: visible, not yet sent, dated inside the weekend window, and with at
// least one published, still-upcoming event. Earliest start_date wins.
func EligiblePick(app core.App, now time.Time) (*core.Record, error) {
	fri, sun := weekendWindow(now)
	// Plain date strings, never time.Time: SQLite would format a time.Time with
	// a zone offset and the string comparison would break.
	picks, err := app.FindRecordsByFilter(
		"picks",
		"hidden = false && newsletter_sent_at = '' && start_date >= {:fri} && start_date <= {:sun}",
		"start_date", 0, 0,
		dbx.Params{
			"fri": fri.Format("2006-01-02") + " 00:00:00",
			"sun": sun.Format("2006-01-02") + " 23:59:59",
		},
	)
	if err != nil {
		return nil, err
	}
	for _, p := range picks {
		evs, err := upcomingEvents(app, p, now)
		if err != nil {
			return nil, err
		}
		if len(evs) > 0 {
			return p, nil
		}
	}
	return nil, nil
}

// upcomingEvents returns the pick's published events that are still upcoming
// (recurring series count while they have a future occurrence), soonest first.
func upcomingEvents(app core.App, pick *core.Record, now time.Time) ([]*core.Record, error) {
	ids := pick.GetStringSlice("events")
	if len(ids) == 0 {
		return nil, nil
	}
	all, err := app.FindRecordsByIds("events", ids)
	if err != nil {
		return nil, err
	}
	var out []*core.Record
	for _, ev := range all {
		if ev.GetString("status") != "published" {
			continue
		}
		if seo.EventLifecycle(ev, now) != seo.Upcoming {
			continue
		}
		out = append(out, ev)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].GetDateTime("start_datetime").Time().Before(out[j].GetDateTime("start_datetime").Time())
	})
	return out, nil
}
