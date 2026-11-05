package newsletter

import (
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"
)

func TestWeekendWindow(t *testing.T) {
	loc := time.UTC
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	d := func(l *time.Location, y int, m time.Month, day, h int) time.Time {
		return time.Date(y, m, day, h, 30, 0, 0, l)
	}
	cases := []struct {
		name     string
		now      time.Time
		fri, sun string
	}{
		{"wednesday", d(loc, 2026, 11, 4, 10), "2026-11-06", "2026-11-08"},
		{"monday", d(loc, 2026, 11, 9, 8), "2026-11-13", "2026-11-15"},
		{"thursday", d(loc, 2026, 11, 5, 23), "2026-11-06", "2026-11-08"},
		{"friday", d(loc, 2026, 11, 6, 9), "2026-11-06", "2026-11-08"},
		{"saturday", d(loc, 2026, 11, 7, 9), "2026-11-06", "2026-11-08"},
		{"sunday", d(loc, 2026, 11, 8, 20), "2026-11-06", "2026-11-08"},
		// UK clocks go back on Sunday 2026-10-25; the window must not drift.
		{"dst week wednesday", d(london, 2026, 10, 21, 12), "2026-10-23", "2026-10-25"},
		{"dst sunday", d(london, 2026, 10, 25, 12), "2026-10-23", "2026-10-25"},
		{"dst monday after", d(london, 2026, 10, 26, 12), "2026-10-30", "2026-11-01"},
	}
	for _, c := range cases {
		fri, sun := weekendWindow(c.now)
		if got := fri.Format("2006-01-02"); got != c.fri {
			t.Errorf("%s: fri = %s, want %s", c.name, got, c.fri)
		}
		if got := sun.Format("2006-01-02"); got != c.sun {
			t.Errorf("%s: sun = %s, want %s", c.name, got, c.sun)
		}
		if fri.Hour() != 0 || sun.Hour() != 0 {
			t.Errorf("%s: window dates must be date-only", c.name)
		}
	}
}

type pickSpec struct {
	slug   string
	start  string // "2006-01-02"
	hidden bool
	sent   bool
	events []string
}

func mkEvent(t *testing.T, app core.App, start, end time.Time, status, rule string) string {
	t.Helper()
	col, _ := app.FindCollectionByNameOrId("events")
	ev := core.NewRecord(col)
	ev.Set("title", "Event")
	ev.Set("start_datetime", start)
	if !end.IsZero() {
		ev.Set("end_datetime", end)
	}
	ev.Set("status", status)
	if rule != "" {
		ev.Set("recurrence_rule", rule)
	}
	if err := app.SaveNoValidate(ev); err != nil {
		t.Fatal(err)
	}
	return ev.Id
}

func mkPick(t *testing.T, app core.App, s pickSpec) *core.Record {
	t.Helper()
	col, _ := app.FindCollectionByNameOrId("picks")
	p := core.NewRecord(col)
	p.Set("title", s.slug)
	p.Set("slug", s.slug)
	p.Set("hidden", s.hidden)
	p.Set("events", s.events)
	p.Set("start_date", s.start+" 00:00:00.000Z")
	if s.sent {
		p.Set("newsletter_sent_at", time.Now())
	}
	if err := app.SaveNoValidate(p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEligiblePick(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC) // Wednesday
	future := mkEvent(t, app, now.AddDate(0, 0, 2), time.Time{}, "published", "")
	ended := mkEvent(t, app, now.AddDate(0, 0, -30), time.Time{}, "published", "")
	cancelled := mkEvent(t, app, now.AddDate(0, 0, 2), time.Time{}, "cancelled", "")
	draft := mkEvent(t, app, now.AddDate(0, 0, 2), time.Time{}, "draft", "")
	pending := mkEvent(t, app, now.AddDate(0, 0, 2), time.Time{}, "pending", "")
	series := mkEvent(t, app, now.AddDate(0, 0, -14), time.Time{}, "published", "FREQ=WEEKLY;COUNT=500")

	// ineligible
	mkPick(t, app, pickSpec{slug: "hidden", start: "2026-11-06", hidden: true, events: []string{future}})
	mkPick(t, app, pickSpec{slug: "already-sent", start: "2026-11-06", sent: true, events: []string{future}})
	mkPick(t, app, pickSpec{slug: "wrong-week", start: "2026-11-13", events: []string{future}})
	mkPick(t, app, pickSpec{slug: "thursday-dated", start: "2026-11-05", events: []string{future}})
	mkPick(t, app, pickSpec{slug: "all-ended", start: "2026-11-06", events: []string{ended}})
	mkPick(t, app, pickSpec{slug: "only-cancelled", start: "2026-11-06", events: []string{cancelled, draft, pending}})
	mkPick(t, app, pickSpec{slug: "no-events", start: "2026-11-06"})

	if got, err := EligiblePick(app, now); err != nil || got != nil {
		t.Fatalf("nothing should be eligible yet, got %v (err %v)", got, err)
	}

	// eligible: Sunday-dated, with a recurring series (first date long past)
	mkPick(t, app, pickSpec{slug: "sunday-dated", start: "2026-11-08", events: []string{future}})
	got, err := EligiblePick(app, now)
	if err != nil || got == nil || got.GetString("slug") != "sunday-dated" {
		t.Fatalf("sunday-dated post should be eligible, got %v (err %v)", got, err)
	}

	mkPick(t, app, pickSpec{slug: "recurring", start: "2026-11-06", events: []string{series}})
	got, _ = EligiblePick(app, now)
	if got == nil || got.GetString("slug") != "recurring" {
		t.Fatalf("earliest start_date wins and a recurring series with a future occurrence counts; got %v", got)
	}
}

func TestUpcomingEventsFiltersAndOrders(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	later := mkEvent(t, app, now.AddDate(0, 0, 3), time.Time{}, "published", "")
	sooner := mkEvent(t, app, now.AddDate(0, 0, 2), time.Time{}, "published", "")
	cancelled := mkEvent(t, app, now.AddDate(0, 0, 2), time.Time{}, "cancelled", "")
	ended := mkEvent(t, app, now.AddDate(0, 0, -2), time.Time{}, "published", "")
	pick := mkPick(t, app, pickSpec{slug: "p", start: "2026-11-06", events: []string{later, cancelled, ended, sooner}})

	evs, err := upcomingEvents(app, pick, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 || evs[0].Id != sooner || evs[1].Id != later {
		t.Errorf("want [sooner, later], got %d events", len(evs))
	}
}
