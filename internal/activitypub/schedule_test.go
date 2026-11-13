package activitypub

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

var london = func() *time.Location {
	l, err := time.LoadLocation("Europe/London")
	if err != nil {
		panic(err)
	}
	return l
}()

func TestPostAtIsNineAmLeadDaysBeforeTheEventDay(t *testing.T) {
	start := time.Date(2026, 12, 19, 19, 30, 0, 0, london)
	want := time.Date(2026, 12, 16, 9, 0, 0, 0, london)
	if got := PostAt(start, 3, london); !got.Equal(want) {
		t.Errorf("PostAt = %v, want %v", got, want)
	}
	// a late-evening UTC start that is already tomorrow locally uses the local day
	utcLate := time.Date(2026, 6, 19, 23, 30, 0, 0, time.UTC) // 00:30 on the 20th in London (BST)
	if got, want := PostAt(utcLate, 1, london), time.Date(2026, 6, 19, 9, 0, 0, 0, london); !got.Equal(want) {
		t.Errorf("PostAt (local day) = %v, want %v", got, want)
	}
}

func TestDueNow(t *testing.T) {
	start := time.Date(2026, 12, 19, 19, 30, 0, 0, london)
	cases := []struct {
		name string
		lead int
		now  time.Time
		want bool
	}{
		{"lead 0 is always immediate", 0, start.AddDate(0, 0, -30), true},
		{"far before the post time", 3, time.Date(2026, 12, 10, 12, 0, 0, 0, london), false},
		{"day of post time but before 9am", 3, time.Date(2026, 12, 16, 8, 59, 0, 0, london), false},
		{"exactly 9am", 3, time.Date(2026, 12, 16, 9, 0, 0, 0, london), true},
		{"already inside the window", 3, time.Date(2026, 12, 18, 12, 0, 0, 0, london), true},
	}
	for _, tc := range cases {
		if got := DueNow(start, tc.lead, tc.now, london); got != tc.want {
			t.Errorf("%s: DueNow = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func scheduleFixture(t *testing.T, lead int) *tests.TestApp {
	t.Helper()
	app := newTestApp(t)
	s := core.NewRecord(mustCollection(t, app, "settings"))
	s.Set("instance_name", "Test")
	s.Set("ap_event_lead_days", lead)
	if err := app.Save(s); err != nil {
		t.Fatal(err)
	}
	f := core.NewRecord(mustCollection(t, app, "ap_followers"))
	f.Set("actor_url", "https://remote.example/users/a")
	f.Set("inbox_url", "https://remote.example/inbox")
	if err := app.Save(f); err != nil {
		t.Fatal(err)
	}
	return app
}

func mustCollection(t *testing.T, app core.App, name string) *core.Collection {
	t.Helper()
	c, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mkScheduledEvent(t *testing.T, app core.App, title string, start time.Time, status string) *core.Record {
	t.Helper()
	e := core.NewRecord(mustCollection(t, app, "events"))
	e.Set("title", title)
	e.Set("start_datetime", start)
	e.Set("status", status)
	if err := app.SaveNoValidate(e); err != nil {
		t.Fatal(err)
	}
	return e
}

func queuedActivityTypes(t *testing.T, app core.App) []string {
	t.Helper()
	recs, err := app.FindRecordsByFilter("ap_delivery_queue", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range recs {
		var raw struct {
			Type   string
			Object struct{ Content string }
		}
		_ = json.Unmarshal([]byte(r.GetString("activity")), &raw)
		out = append(out, raw.Type+":"+raw.Object.Content)
	}
	return out
}

func TestAnnounceEventStampsOnceAndQueuesOneCreate(t *testing.T) {
	app := scheduleFixture(t, 3)
	defer app.Cleanup()
	ev := mkScheduledEvent(t, app, "Gig", time.Now().Add(48*time.Hour), "published")

	ok, err := AnnounceEvent(app, "https://example.com", ev, time.Now())
	if err != nil || !ok {
		t.Fatalf("first announce = %v, %v", ok, err)
	}
	ok, err = AnnounceEvent(app, "https://example.com", ev, time.Now())
	if err != nil || ok {
		t.Fatalf("second announce = %v, %v; must be a no-op", ok, err)
	}
	if n := len(queuedActivityTypes(t, app)); n != 1 {
		t.Errorf("queued %d deliveries, want exactly 1", n)
	}
	got, _ := app.FindRecordById("events", ev.Id)
	if got.GetDateTime("ap_posted_at").IsZero() {
		t.Error("ap_posted_at should be stamped")
	}
}

func TestPostDueEventsOnlyPostsEventsInsideTheWindow(t *testing.T) {
	app := scheduleFixture(t, 3)
	defer app.Cleanup()
	now := time.Date(2026, 12, 16, 10, 0, 0, 0, london) // after 9am on the post day
	due := mkScheduledEvent(t, app, "Due", time.Date(2026, 12, 19, 19, 0, 0, 0, london), "published")
	far := mkScheduledEvent(t, app, "Far", time.Date(2026, 12, 30, 19, 0, 0, 0, london), "published")
	past := mkScheduledEvent(t, app, "Past", time.Date(2026, 12, 1, 19, 0, 0, 0, london), "published")
	draft := mkScheduledEvent(t, app, "Draft", time.Date(2026, 12, 19, 19, 0, 0, 0, london), "draft")
	done := mkScheduledEvent(t, app, "Done", time.Date(2026, 12, 19, 20, 0, 0, 0, london), "published")
	if ok, err := StampPosted(app, done.Id, now); err != nil || !ok {
		t.Fatal(ok, err)
	}

	n, err := PostDueEvents(app, "https://example.com", now, london)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("posted %d events, want 1 (only the due one)", n)
	}
	for id, want := range map[string]bool{due.Id: true, far.Id: false, past.Id: false, draft.Id: false} {
		rec, _ := app.FindRecordById("events", id)
		if got := !rec.GetDateTime("ap_posted_at").IsZero(); got != want {
			t.Errorf("%s posted = %v, want %v", rec.GetString("title"), got, want)
		}
	}
	// idempotent: the next tick posts nothing new
	if n, _ := PostDueEvents(app, "https://example.com", now.Add(15*time.Minute), london); n != 0 {
		t.Errorf("second tick posted %d, want 0", n)
	}
}
