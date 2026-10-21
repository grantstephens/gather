package hooks_test

import (
	"encoding/json"
	"testing"
	"time"

	"gather/internal/hooks"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "gather/migrations"
)

func relayFixture(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.RunAllMigrations(); err != nil {
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

func TestEventHooks_DraftIsNotFederated(t *testing.T) {
	app := relayFixture(t)
	defer app.Cleanup()

	saveEvent(t, app, "draft")

	if got := queuedTypes(t, app); len(got) != 0 {
		t.Errorf("queued = %v, want nothing", got)
	}
}
