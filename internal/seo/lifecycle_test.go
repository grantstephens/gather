package seo

import (
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func TestLifecycleOf(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	h := func(d time.Duration) time.Time { return now.Add(d) }
	day := 24 * time.Hour
	cases := []struct {
		name       string
		start, end time.Time
		want       State
	}{
		{"future", h(day), time.Time{}, Upcoming},
		{"in progress (no end, within 3h)", h(-2 * time.Hour), time.Time{}, Upcoming},
		{"just ended", h(-5 * time.Hour), time.Time{}, Ended},
		{"ended 6 days ago", h(-6 * day), time.Time{}, Ended},
		{"ended 8 days ago", h(-8 * day), time.Time{}, Retired},
		{"multi-day exhibition still running", h(-60 * day), h(10 * day), Upcoming},
		{"multi-day exhibition ended 3 days ago", h(-60 * day), h(-3 * day), Ended},
		{"multi-day exhibition ended 9 days ago", h(-60 * day), h(-9 * day), Retired},
		{"end before start is ignored", h(-8 * day), h(-9 * day), Retired},
		{"zero start is never retired", time.Time{}, time.Time{}, Upcoming},
	}
	for _, c := range cases {
		if got := LifecycleOf(c.start, c.end, now); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func recordWith(t *testing.T, app core.App, start time.Time, rule string) *core.Record {
	t.Helper()
	events, _ := app.FindCollectionByNameOrId("events")
	ev := core.NewRecord(events)
	ev.Set("title", "Recurring")
	ev.Set("start_datetime", start)
	ev.Set("recurrence_rule", rule)
	return ev
}

func TestEventLifecycleRecurring(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Now()
	start := now.AddDate(0, 0, -14)

	if got := EventLifecycle(recordWith(t, app, start, "FREQ=WEEKLY;COUNT=500"), now); got != Upcoming {
		t.Errorf("open-ended weekly series must stay upcoming, got %v", got)
	}
	if got := EventLifecycle(recordWith(t, app, now.AddDate(0, 0, -20), "FREQ=WEEKLY;COUNT=2"), now); got != Retired {
		t.Errorf("a series whose last date was 13 days ago is retired, got %v", got)
	}
	// finished yesterday: judged by its LAST occurrence, not its first date
	if got := EventLifecycle(recordWith(t, app, now.AddDate(0, 0, -15), "FREQ=WEEKLY;COUNT=3"), now); got != Ended {
		t.Errorf("series whose last date was yesterday is ended (not retired), got %v", got)
	}
	if got := EventLifecycle(recordWith(t, app, now.AddDate(0, 0, 3), ""), now); got != Upcoming {
		t.Errorf("plain future event is upcoming, got %v", got)
	}
}

func TestPickLifecycle(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Now()
	events, _ := app.FindCollectionByNameOrId("events")
	mk := func(start time.Time) string {
		ev := core.NewRecord(events)
		ev.Set("title", "e")
		ev.Set("start_datetime", start)
		ev.Set("status", "published")
		if err := app.Save(ev); err != nil {
			t.Fatal(err)
		}
		return ev.Id
	}
	picks, _ := app.FindCollectionByNameOrId("picks")
	pick := func(ids ...string) *core.Record {
		p := core.NewRecord(picks)
		p.Set("title", "p")
		p.Set("slug", "p")
		p.Set("events", ids)
		return p
	}
	old, fresh := mk(now.AddDate(0, 0, -30)), mk(now.AddDate(0, 0, 3))
	if got := PickLifecycle(app, pick(old), now); got != Retired {
		t.Errorf("all events long past => retired, got %v", got)
	}
	if got := PickLifecycle(app, pick(old, fresh), now); got != Upcoming {
		t.Errorf("one upcoming event keeps the post alive, got %v", got)
	}
	if got := PickLifecycle(app, pick(), now); got != Upcoming {
		t.Errorf("evergreen post with no events is never retired, got %v", got)
	}
}
