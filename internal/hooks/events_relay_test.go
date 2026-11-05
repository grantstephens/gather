package hooks_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"gather/internal/hooks"
	"gather/internal/testutil"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestMain(m *testing.M) {
	code := m.Run()
	testutil.Cleanup()
	os.Exit(code)
}

func relayFixture(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := testutil.NewApp()
	if err != nil {
		t.Fatal(err)
	}
	settings, _ := app.FindCollectionByNameOrId("settings")
	s := core.NewRecord(settings)
	s.Set("instance_name", "Test")
	if err := app.Save(s); err != nil {
		t.Fatal(err)
	}
	col, _ := app.FindCollectionByNameOrId("ap_relay_followers")
	f := core.NewRecord(col)
	f.Set("actor_url", "https://mz.example/relay")
	f.Set("inbox_url", "https://mz.example/inbox")
	if err := app.Save(f); err != nil {
		t.Fatal(err)
	}
	hooks.RegisterEventHooks(app, "https://example.com")
	return app
}

func queuedTypes(t *testing.T, app core.App) map[string]int {
	t.Helper()
	time.Sleep(300 * time.Millisecond) // hooks queue in a goroutine
	recs, err := app.FindRecordsByFilter("ap_delivery_queue", "id != ''", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	for _, r := range recs {
		var a struct{ Type string }
		_ = json.Unmarshal([]byte(r.GetString("activity")), &a)
		out[a.Type]++
	}
	return out
}

func saveEvent(t *testing.T, app core.App, status string) *core.Record {
	t.Helper()
	col, _ := app.FindCollectionByNameOrId("events")
	e := core.NewRecord(col)
	e.Set("title", "Gig")
	e.Set("start_datetime", time.Now().Add(48*time.Hour))
	e.Set("status", status)
	if err := app.Save(e); err != nil {
		t.Fatal(err)
	}
	// reload: a just-created record carries a stale Original(), which the update hook reads
	got, err := app.FindRecordById("events", e.Id)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestEventHooks_PublishedEventGoesToRelayFollowers(t *testing.T) {
	app := relayFixture(t)
	defer app.Cleanup()

	saveEvent(t, app, "published")

	got := queuedTypes(t, app)
	if got["Create"] != 1 || got["Announce"] != 1 {
		t.Errorf("queued = %v, want one Create and one Announce for the relay follower", got)
	}
}

func TestEventHooks_CancellingSendsUpdateNotDelete(t *testing.T) {
	app := relayFixture(t)
	defer app.Cleanup()
	e := saveEvent(t, app, "published")
	queuedTypes(t, app) // let the Create settle

	e.Set("status", "cancelled")
	if err := app.Save(e); err != nil {
		t.Fatal(err)
	}

	// /ap/events/{id} still serves cancelled events (status CANCELLED), so a Delete
	// would be ignored by Mobilizon when it re-fetches.
	got := queuedTypes(t, app)
	if got["Update"] != 1 || got["Delete"] != 0 {
		t.Errorf("queued = %v, want one Update and no Delete", got)
	}
}

func TestEventHooks_UnpublishingToDraftSendsDelete(t *testing.T) {
	app := relayFixture(t)
	defer app.Cleanup()
	e := saveEvent(t, app, "published")
	queuedTypes(t, app)

	e.Set("status", "draft")
	if err := app.Save(e); err != nil {
		t.Fatal(err)
	}

	if got := queuedTypes(t, app); got["Delete"] != 1 {
		t.Errorf("queued = %v, want a Delete", got)
	}
}

func TestEventHooks_CancelledThenDraftedSendsDelete(t *testing.T) {
	app := relayFixture(t)
	defer app.Cleanup()
	e := saveEvent(t, app, "cancelled")
	e.Set("status", "draft")
	if err := app.Save(e); err != nil {
		t.Fatal(err)
	}

	if got := queuedTypes(t, app); got["Delete"] != 1 {
		t.Errorf("queued = %v, want a Delete (Mobilizon still has the cancelled event)", got)
	}
}

func TestEventHooks_CancelledThenRepublishedSendsUpdate(t *testing.T) {
	app := relayFixture(t)
	defer app.Cleanup()
	e := saveEvent(t, app, "cancelled")
	e.Set("status", "published")
	if err := app.Save(e); err != nil {
		t.Fatal(err)
	}

	got := queuedTypes(t, app)
	if got["Update"] != 1 || got["Create"] != 0 {
		t.Errorf("queued = %v, want one Update and no Create", got)
	}
}

func TestEventHooks_DeletingCancelledEventSendsDelete(t *testing.T) {
	app := relayFixture(t)
	defer app.Cleanup()
	e := saveEvent(t, app, "cancelled")
	if err := app.Delete(e); err != nil {
		t.Fatal(err)
	}

	if got := queuedTypes(t, app); got["Delete"] != 1 {
		t.Errorf("queued = %v, want a Delete", got)
	}
}

func TestEventHooks_PlaceFollowersGetTheEventToo(t *testing.T) {
	app := relayFixture(t)
	defer app.Cleanup()
	// drop the relay follower so only the place follower can explain the queue
	relays, _ := app.FindRecordsByFilter("ap_relay_followers", "id != ''", "", 0, 0)
	for _, r := range relays {
		if err := app.Delete(r); err != nil {
			t.Fatal(err)
		}
	}
	places, _ := app.FindCollectionByNameOrId("places")
	p := core.NewRecord(places)
	p.Set("name", "Theatre")
	p.Set("city", "Pitlochry")
	p.Set("status", "approved")
	if err := app.Save(p); err != nil {
		t.Fatal(err)
	}
	col, _ := app.FindCollectionByNameOrId("ap_subject_followers")
	f := core.NewRecord(col)
	f.Set("subject", "place:"+p.Id)
	f.Set("actor_url", "https://mz.example/u")
	f.Set("inbox_url", "https://mz.example/u/inbox")
	if err := app.Save(f); err != nil {
		t.Fatal(err)
	}

	events, _ := app.FindCollectionByNameOrId("events")
	e := core.NewRecord(events)
	e.Set("title", "Gig")
	e.Set("start_datetime", time.Now().Add(48*time.Hour))
	e.Set("status", "published")
	e.Set("place", p.Id)
	if err := app.Save(e); err != nil {
		t.Fatal(err)
	}

	got := queuedTypes(t, app)
	if got["Create"] != 1 || got["Announce"] != 0 {
		t.Errorf("queued = %v, want just a Create (from the place) for the place follower", got)
	}
}

func TestEventHooks_DraftIsNotFederated(t *testing.T) {
	app := relayFixture(t)
	defer app.Cleanup()

	saveEvent(t, app, "draft")

	if got := queuedTypes(t, app); len(got) != 0 {
		t.Errorf("queued = %v, want nothing", got)
	}
}

// apFixture has one ordinary ActivityPub follower and the given lead time.
func apFixture(t *testing.T, lead int) *tests.TestApp {
	t.Helper()
	app, err := testutil.NewApp()
	if err != nil {
		t.Fatal(err)
	}
	settings, _ := app.FindCollectionByNameOrId("settings")
	s := core.NewRecord(settings)
	s.Set("instance_name", "Test")
	s.Set("ap_event_lead_days", lead)
	if err := app.Save(s); err != nil {
		t.Fatal(err)
	}
	col, _ := app.FindCollectionByNameOrId("ap_followers")
	f := core.NewRecord(col)
	f.Set("actor_url", "https://remote.example/users/a")
	f.Set("inbox_url", "https://remote.example/inbox")
	if err := app.Save(f); err != nil {
		t.Fatal(err)
	}
	hooks.RegisterEventHooks(app, "https://example.com")
	return app
}

func saveEventAt(t *testing.T, app core.App, status string, start time.Time) *core.Record {
	t.Helper()
	col, _ := app.FindCollectionByNameOrId("events")
	e := core.NewRecord(col)
	e.Set("title", "Gig")
	e.Set("start_datetime", start)
	e.Set("status", status)
	if err := app.Save(e); err != nil {
		t.Fatal(err)
	}
	got, err := app.FindRecordById("events", e.Id)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// laterEdit simulates a separate, later request: let the background post
// finish, then load the record fresh (a stale copy would write the old,
// empty ap_posted_at back over the stamp).
func laterEdit(t *testing.T, app core.App, id string) *core.Record {
	t.Helper()
	time.Sleep(300 * time.Millisecond)
	rec, err := app.FindRecordById("events", id)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func posted(t *testing.T, app core.App, id string) bool {
	t.Helper()
	rec, err := app.FindRecordById("events", id)
	if err != nil {
		t.Fatal(err)
	}
	return !rec.GetDateTime("ap_posted_at").IsZero()
}

func TestEventHooks_FarOffEventIsNotPostedOnPublish(t *testing.T) {
	app := apFixture(t, 3)
	defer app.Cleanup()
	ev := saveEventAt(t, app, "published", time.Now().Add(10*24*time.Hour))
	if got := queuedTypes(t, app); len(got) != 0 {
		t.Errorf("an event 10 days away must wait for its post time, queued %v", got)
	}
	if posted(t, app, ev.Id) {
		t.Error("must not be marked posted")
	}
}

func TestEventHooks_EventInsideTheWindowIsPostedOnPublish(t *testing.T) {
	app := apFixture(t, 3)
	defer app.Cleanup()
	ev := saveEventAt(t, app, "published", time.Now().Add(36*time.Hour))
	if got := queuedTypes(t, app); got["Create"] != 1 {
		t.Errorf("want one Create, got %v", got)
	}
	if !posted(t, app, ev.Id) {
		t.Error("should be marked posted")
	}
}

func TestEventHooks_ZeroLeadPostsImmediately(t *testing.T) {
	app := apFixture(t, 0)
	defer app.Cleanup()
	saveEventAt(t, app, "published", time.Now().Add(30*24*time.Hour))
	if got := queuedTypes(t, app); got["Create"] != 1 {
		t.Errorf("lead 0 keeps the old behaviour, got %v", got)
	}
}

func TestEventHooks_EditOfUnpostedEventIsSilent(t *testing.T) {
	app := apFixture(t, 3)
	defer app.Cleanup()
	ev := saveEventAt(t, app, "published", time.Now().Add(10*24*time.Hour))
	ev.Set("title", "Renamed")
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}
	if got := queuedTypes(t, app); len(got) != 0 {
		t.Errorf("nothing to update on the Fediverse yet, queued %v", got)
	}
}

func TestEventHooks_EditOfPostedEventSendsUpdate(t *testing.T) {
	app := apFixture(t, 3)
	defer app.Cleanup()
	ev := saveEventAt(t, app, "published", time.Now().Add(36*time.Hour))
	ev = laterEdit(t, app, ev.Id)
	ev.Set("title", "Renamed")
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}
	if got := queuedTypes(t, app); got["Update"] != 1 {
		t.Errorf("want one Update, got %v", got)
	}
}

func TestEventHooks_UnpublishingAPostedEventSendsDeleteAndClearsTheMarker(t *testing.T) {
	app := apFixture(t, 3)
	defer app.Cleanup()
	ev := saveEventAt(t, app, "published", time.Now().Add(36*time.Hour))
	ev = laterEdit(t, app, ev.Id)
	ev.Set("status", "cancelled")
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}
	if got := queuedTypes(t, app); got["Delete"] != 1 {
		t.Errorf("want one Delete, got %v", got)
	}
	if posted(t, app, ev.Id) {
		t.Error("followers no longer have it, so it must read as not posted")
	}
}

func TestEventHooks_UnpublishingAnUnpostedEventIsSilent(t *testing.T) {
	app := apFixture(t, 3)
	defer app.Cleanup()
	ev := saveEventAt(t, app, "published", time.Now().Add(10*24*time.Hour))
	ev.Set("status", "cancelled")
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}
	if got := queuedTypes(t, app); len(got) != 0 {
		t.Errorf("followers never saw it, so no Delete: queued %v", got)
	}
}
