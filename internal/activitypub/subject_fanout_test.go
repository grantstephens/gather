package activitypub

import (
	"fmt"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func addSubjectFollower(t *testing.T, app *tests.TestApp, subject, actor, inbox, shared string) {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("ap_subject_followers")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(col)
	r.Set("subject", subject)
	r.Set("actor_url", actor)
	r.Set("inbox_url", inbox)
	r.Set("shared_inbox_url", shared)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
}

// eventAt creates a published event at a place.
func eventAt(t *testing.T, app *tests.TestApp, placeID string) *core.Record {
	t.Helper()
	ev := createPublishedEvent(t, app, "Summer Gig", time.Now().Add(48*time.Hour))
	ev.Set("place", placeID)
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}
	got, err := app.FindRecordById("events", ev.Id)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// queueSummary lists "inbox type<-actor-suffix" for each queued activity.
func queueSummary(t *testing.T, app core.App) []string {
	t.Helper()
	var out []string
	for _, q := range queuedActivities(t, app) {
		out = append(out, fmt.Sprintf("%s %s<-%s", q["_inbox"], q["type"], q["actor"]))
	}
	return out
}

func TestQueueSubjectEvent_PlaceFollowerGetsEventThenPlaceAnnounce(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	pid := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	addSubjectFollower(t, app, "place:"+pid, "https://a.example/u", "https://a.example/u/inbox", "https://a.example/inbox")
	ev := eventAt(t, app, pid)

	if err := QueueSubjectEvent(app, testBase, ev, "Create"); err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprint([]string{
		"https://a.example/inbox Create<-" + testBase + "/ap/actor",
		"https://a.example/inbox Announce<-" + testBase + "/ap/actor/place/" + pid,
	})
	if got := fmt.Sprint(queueSummary(t, app)); got != want {
		t.Errorf("queue = %v\nwant    %v", got, want)
	}
}

func TestQueueSubjectEvent_TownFollowerGetsTownAnnounce(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	pid := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	addSubjectFollower(t, app, "town:pitlochry", "https://a.example/u", "https://a.example/u/inbox", "")
	ev := eventAt(t, app, pid)

	if err := QueueSubjectEvent(app, testBase, ev, "Create"); err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprint([]string{
		"https://a.example/u/inbox Create<-" + testBase + "/ap/actor",
		"https://a.example/u/inbox Announce<-" + testBase + "/ap/actor/town/pitlochry",
	})
	if got := fmt.Sprint(queueSummary(t, app)); got != want {
		t.Errorf("queue = %v\nwant    %v", got, want)
	}
}

func TestQueueSubjectEvent_FollowingPlaceAndTownDeliversOnce(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	pid := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	addSubjectFollower(t, app, "place:"+pid, "https://a.example/u", "https://a.example/u/inbox", "")
	addSubjectFollower(t, app, "town:pitlochry", "https://a.example/u", "https://a.example/u/inbox", "")
	ev := eventAt(t, app, pid)

	if err := QueueSubjectEvent(app, testBase, ev, "Create"); err != nil {
		t.Fatal(err)
	}

	if got := queueSummary(t, app); len(got) != 2 {
		t.Errorf("queue = %v, want one Create and one Announce", got)
	}
}

func TestQueueSubjectEvent_UpdateAndDeleteAreSingleOwnerActivities(t *testing.T) {
	for _, typ := range []string{"Update", "Delete"} {
		app := newTestApp(t)
		createSettingsWithKeys(t, app)
		pid := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
		addSubjectFollower(t, app, "place:"+pid, "https://a.example/u", "https://a.example/u/inbox", "")
		ev := eventAt(t, app, pid)

		if err := QueueSubjectEvent(app, testBase, ev, typ); err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprint([]string{"https://a.example/u/inbox " + typ + "<-" + testBase + "/ap/actor"})
		if got := fmt.Sprint(queueSummary(t, app)); got != want {
			t.Errorf("%s: queue = %v, want %v", typ, got, want)
		}
		app.Cleanup()
	}
}

func TestQueueSubjectEvent_NoPlaceOrPendingPlaceQueuesNothing(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	pending := slugPlace(t, app, "Shady Bar", "Dunkeld", "shady", "pending")
	addSubjectFollower(t, app, "place:"+pending, "https://a.example/u", "https://a.example/u/inbox", "")
	addSubjectFollower(t, app, "town:dunkeld", "https://a.example/u", "https://a.example/u/inbox", "")

	for _, ev := range []*core.Record{eventAt(t, app, pending), eventAt(t, app, "")} {
		if err := QueueSubjectEvent(app, testBase, ev, "Create"); err != nil {
			t.Fatal(err)
		}
	}
	if n, _ := app.CountRecords("ap_delivery_queue"); n != 0 {
		t.Errorf("queue = %d, want 0", n)
	}
}

func TestQueueSubjectEvent_OtherPlacesFollowersAreLeftAlone(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	a := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	b := slugPlace(t, app, "Cinema", "Perth", "cinema", "approved")
	addSubjectFollower(t, app, "place:"+b, "https://a.example/u", "https://a.example/u/inbox", "")
	ev := eventAt(t, app, a)

	if err := QueueSubjectEvent(app, testBase, ev, "Create"); err != nil {
		t.Fatal(err)
	}
	if n, _ := app.CountRecords("ap_delivery_queue"); n != 0 {
		t.Errorf("queue = %d, want 0", n)
	}
}
