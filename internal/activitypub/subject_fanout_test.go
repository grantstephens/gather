package activitypub

import (
	"fmt"
	"strings"
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

func TestSubjectEventObject_ServesTheGroupsCopyOnlyForItsOwnPublishedEvents(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	pid := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	other := slugPlace(t, app, "Cinema", "Perth", "cinema", "approved")
	ev := eventAt(t, app, pid)
	group := testBase + "/ap/actor/place/" + pid

	body, status := SubjectEventObject(app, testBase, "place", pid, ev.Id)
	if status != 200 {
		t.Fatalf("status = %d: %s", status, body)
	}
	obj := actorDoc(t, body)
	if obj["id"] != group+"/events/"+ev.Id || obj["attributedTo"] != group || obj["type"] != "Event" {
		t.Errorf("id/attributedTo/type = %v / %v / %v", obj["id"], obj["attributedTo"], obj["type"])
	}
	if town, status := SubjectEventObject(app, testBase, "town", "pitlochry", ev.Id); status != 200 {
		t.Errorf("town copy status = %d: %s", status, town)
	}

	if _, status := SubjectEventObject(app, testBase, "place", other, ev.Id); status != 410 {
		t.Errorf("event at another place: status = %d, want 410", status)
	}
	ev.Set("status", "draft")
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}
	body, status = SubjectEventObject(app, testBase, "place", pid, ev.Id)
	if status != 410 || strings.Contains(string(body), "Summer Gig") {
		t.Errorf("draft: status = %d body = %s", status, body)
	}
}

// The instance actor sends the activity (it is the event's organizer); the group is what the
// Event is attributedTo. Mobilizon only sets a group as owner when organizer and group differ.
func TestQueueSubjectEvent_PlaceFollowerGetsTheInstanceCreatingAPlaceOwnedCopy(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	pid := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	addSubjectFollower(t, app, "place:"+pid, "https://a.example/u", "https://a.example/u/inbox", "https://a.example/inbox")
	ev := eventAt(t, app, pid)

	if err := QueueSubjectEvent(app, testBase, ev, "Create"); err != nil {
		t.Fatal(err)
	}

	group := testBase + "/ap/actor/place/" + pid
	want := fmt.Sprint([]string{"https://a.example/inbox Create<-" + testBase + "/ap/actor"})
	if got := fmt.Sprint(queueSummary(t, app)); got != want {
		t.Errorf("queue = %v\nwant    %v", got, want)
	}
	obj := queuedActivities(t, app)[0]["object"].(map[string]any)
	if obj["type"] != "Event" || obj["id"] != group+"/events/"+ev.Id || obj["attributedTo"] != group || obj["actor"] != testBase+"/ap/actor" {
		t.Errorf("object id/attributedTo/actor = %v / %v / %v, want the group's own copy organised by the instance", obj["id"], obj["attributedTo"], obj["actor"])
	}
}

func TestQueueSubjectEvent_TownFollowerGetsATownOwnedCopy(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	createSettingsWithKeys(t, app)
	pid := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
	addSubjectFollower(t, app, "town:pitlochry", "https://a.example/u", "https://a.example/u/inbox", "")
	ev := eventAt(t, app, pid)

	if err := QueueSubjectEvent(app, testBase, ev, "Create"); err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprint([]string{"https://a.example/u/inbox Create<-" + testBase + "/ap/actor"})
	if got := fmt.Sprint(queueSummary(t, app)); got != want {
		t.Errorf("queue = %v\nwant    %v", got, want)
	}
	obj := queuedActivities(t, app)[0]["object"].(map[string]any)
	if obj["attributedTo"] != testBase+"/ap/actor/town/pitlochry" {
		t.Errorf("attributedTo = %v", obj["attributedTo"])
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

	q := queuedActivities(t, app)
	if len(q) != 1 {
		t.Fatalf("queue = %v, want a single Create", queueSummary(t, app))
	}
	if obj := q[0]["object"].(map[string]any); !strings.HasPrefix(obj["attributedTo"].(string), testBase+"/ap/actor/place/") {
		t.Errorf("attributedTo = %v, want the place copy (it wins over the town)", obj["attributedTo"])
	}
}

func TestQueueSubjectEvent_UpdateAndDeleteTargetTheGroupsCopy(t *testing.T) {
	for _, typ := range []string{"Update", "Delete"} {
		app := newTestApp(t)
		createSettingsWithKeys(t, app)
		pid := slugPlace(t, app, "Theatre", "Pitlochry", "pft", "approved")
		addSubjectFollower(t, app, "place:"+pid, "https://a.example/u", "https://a.example/u/inbox", "")
		ev := eventAt(t, app, pid)

		if err := QueueSubjectEvent(app, testBase, ev, typ); err != nil {
			t.Fatal(err)
		}
		group := testBase + "/ap/actor/place/" + pid
		want := fmt.Sprint([]string{"https://a.example/u/inbox " + typ + "<-" + testBase + "/ap/actor"})
		if got := fmt.Sprint(queueSummary(t, app)); got != want {
			t.Errorf("%s: queue = %v, want %v", typ, got, want)
		}
		if typ == "Delete" {
			if obj := queuedActivities(t, app)[0]["object"]; obj != group+"/events/"+ev.Id {
				t.Errorf("Delete object = %v, want the group's copy id", obj)
			}
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
