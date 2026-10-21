package activitypub

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

const testBase = "https://example.com"

func createPlace(t *testing.T, app *tests.TestApp, name, address, city string, lat, lon float64, status string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("places")
	if err != nil {
		t.Fatalf("find places collection: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("name", name)
	r.Set("address", address)
	r.Set("city", city)
	r.Set("country_code", "GB")
	r.Set("location", types.GeoPoint{Lat: lat, Lon: lon})
	r.Set("status", status)
	if err := app.Save(r); err != nil {
		t.Fatalf("save place: %v", err)
	}
	return r
}

func createTag(t *testing.T, app *tests.TestApp, name, status string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		t.Fatalf("find tags collection: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("name", name)
	r.Set("status", status)
	if err := app.Save(r); err != nil {
		t.Fatalf("save tag: %v", err)
	}
	return r
}

func setTimezone(t *testing.T, app *tests.TestApp, tz string) {
	t.Helper()
	s := createSettings(t, app, "Test Instance", "")
	s.Set("timezone", tz)
	if err := app.Save(s); err != nil {
		t.Fatalf("save timezone: %v", err)
	}
}

func apEvent(t *testing.T, app core.App, ev *core.Record) APEvent {
	t.Helper()
	out, err := eventToAPEvent(app, ev, testBase)
	if err != nil {
		t.Fatalf("eventToAPEvent: %v", err)
	}
	return out
}

func TestEventToAPEvent_CoreFields(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	start := time.Date(2026, 7, 4, 18, 30, 0, 0, time.UTC) // 19:30 BST
	ev := createPublishedEvent(t, app, "Summer Gig", start)

	got := apEvent(t, app, ev)

	if got.Type != "Event" {
		t.Errorf("type = %q, want Event", got.Type)
	}
	if want := testBase + "/ap/events/" + ev.Id; got.ID != want {
		t.Errorf("id = %q, want %q", got.ID, want)
	}
	if got.AttributedTo != testBase+"/ap/actor" {
		t.Errorf("attributedTo = %q", got.AttributedTo)
	}
	if got.Name != "Summer Gig" {
		t.Errorf("name = %q (must be plain text, not HTML-escaped)", got.Name)
	}
	if got.StartTime != "2026-07-04T19:30:00+01:00" {
		t.Errorf("startTime = %q, want local time with offset", got.StartTime)
	}
	if got.Timezone != "Europe/London" {
		t.Errorf("timezone = %q", got.Timezone)
	}
	if got.JoinMode != "external" || got.ExternalParticipationURL != got.URL {
		t.Errorf("joinMode = %q, externalParticipationUrl = %q, url = %q", got.JoinMode, got.ExternalParticipationURL, got.URL)
	}
	if got.Status != "CONFIRMED" {
		t.Errorf("status = %q, want CONFIRMED", got.Status)
	}
	if len(got.To) != 1 || got.To[0] != "https://www.w3.org/ns/activitystreams#Public" {
		t.Errorf("to = %v", got.To)
	}
}

func TestEventToAPEvent_NameIsNotHTMLEscaped(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	ev := createPublishedEvent(t, app, "Rock & Roll <Night>", time.Now().Add(time.Hour))

	if got := apEvent(t, app, ev).Name; got != "Rock & Roll <Night>" {
		t.Errorf("name = %q", got)
	}
}

func TestEventToAPEvent_CancelledEvent(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	ev := createPublishedEvent(t, app, "Called Off", time.Now().Add(time.Hour))
	ev.Set("status", "cancelled")

	if got := apEvent(t, app, ev).Status; got != "CANCELLED" {
		t.Errorf("status = %q, want CANCELLED", got)
	}
}

func TestEventToAPEvent_EndTime(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	start := time.Date(2026, 1, 10, 19, 0, 0, 0, time.UTC)
	ev := createPublishedEvent(t, app, "No End", start)

	if got := apEvent(t, app, ev); got.EndTime != "" {
		t.Errorf("endTime = %q, want omitted when the event has no end", got.EndTime)
	}

	ev.Set("end_datetime", start.Add(2*time.Hour))
	if got := apEvent(t, app, ev).EndTime; got != "2026-01-10T21:00:00Z" {
		t.Errorf("endTime = %q", got)
	}
}

func TestEventToAPEvent_LocationUsesFloatCoordinates(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	place := createPlace(t, app, "Whole Degrees Hall", "1 Test St", "Perth", 56, -3, "approved")
	ev := createPublishedEvent(t, app, "At The Hall", time.Now().Add(time.Hour))
	ev.Set("place", place.Id)

	got := apEvent(t, app, ev)
	if got.Location == nil {
		t.Fatal("location missing")
	}
	b, err := json.Marshal(got.Location)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	// Mobilizon silently drops coordinates that arrive as JSON integers.
	if !strings.Contains(s, `"latitude":56.0`) || !strings.Contains(s, `"longitude":-3.0`) {
		t.Errorf("coordinates must marshal as floats, got %s", s)
	}
	if got.Location.Type != "Place" || got.Location.Name != "Whole Degrees Hall" {
		t.Errorf("place = %+v", got.Location)
	}
	if got.Location.Address == nil || got.Location.Address.Type != "PostalAddress" ||
		got.Location.Address.StreetAddress != "1 Test St" ||
		got.Location.Address.AddressLocality != "Perth" ||
		got.Location.Address.AddressCountry != "GB" {
		t.Errorf("address = %+v", got.Location.Address)
	}
}

func TestEventToAPEvent_NoLocationWithoutApprovedPlace(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")

	none := createPublishedEvent(t, app, "Nowhere", time.Now().Add(time.Hour))
	if apEvent(t, app, none).Location != nil {
		t.Error("event without a place must have no location")
	}

	pending := createPlace(t, app, "Pending Pub", "", "Perth", 56.39, -3.43, "pending")
	ev := createPublishedEvent(t, app, "At Pending", time.Now().Add(time.Hour))
	ev.Set("place", pending.Id)
	if apEvent(t, app, ev).Location != nil {
		t.Error("an unapproved place must not be federated")
	}
}

func TestEventToAPEvent_ContentStripsBoilerplateAndLinksBack(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	ev := createPublishedEvent(t, app, "Talk", time.Now().Add(time.Hour))
	ev.Set("description", "A good talk.<br><br>Cost: £5<br>Source: https://example.org/talk")
	ev.Set("slug", "talk-1")

	got := apEvent(t, app, ev)

	if strings.Contains(got.Content, "Cost:") || strings.Contains(got.Content, "Source:") {
		t.Errorf("boilerplate not stripped: %q", got.Content)
	}
	if !strings.Contains(got.Content, "A good talk.") {
		t.Errorf("description lost: %q", got.Content)
	}
	link := testBase + "/event/talk-1"
	if !strings.Contains(got.Content, `<a href="`+link+`">`+link+`</a>`) {
		t.Errorf("content must link back to %s, got %q", link, got.Content)
	}
	if got.URL != link {
		t.Errorf("url = %q, want %q", got.URL, link)
	}
}

func TestEventToAPEvent_Tags(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	music := createTag(t, app, "live music", "approved")
	pending := createTag(t, app, "secret", "pending")
	long := createTag(t, app, strings.Repeat("x", 41), "approved")
	ev := createPublishedEvent(t, app, "Tagged", time.Now().Add(time.Hour))
	ev.Set("tags", []string{music.Id, pending.Id, long.Id})

	got := apEvent(t, app, ev)

	if len(got.Tag) != 1 {
		t.Fatalf("tags = %+v, want only the approved tag of <= 40 chars", got.Tag)
	}
	if got.Tag[0].Type != "Hashtag" || got.Tag[0].Name != "#livemusic" {
		t.Errorf("tag = %+v, want Hashtag #livemusic", got.Tag[0])
	}
}

func TestEventToAPEvent_ImageAttachmentIsNamed(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	ev := createPublishedEvent(t, app, "Pictured", time.Now().Add(time.Hour))
	ev.Set("image", "poster_abc123.jpg")

	got := apEvent(t, app, ev)

	if len(got.Attachment) != 1 {
		t.Fatalf("attachment = %+v", got.Attachment)
	}
	a := got.Attachment[0]
	// Mobilizon crashes the whole Create if an attachment has no name.
	if a.Name == "" || a.Type != "Document" {
		t.Errorf("attachment must be a named Document, got %+v", a)
	}
	if !strings.HasSuffix(a.URL, ".webp") || !strings.HasPrefix(a.URL, testBase+"/api/files/") {
		t.Errorf("url = %q, want a .webp under /api/files/", a.URL)
	}
}

func TestEventToAPEvent_ContextHasMobilizonTerms(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	ev := createPublishedEvent(t, app, "Ctx", time.Now().Add(time.Hour))

	b, err := json.Marshal(apEvent(t, app, ev))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`https://www.w3.org/ns/activitystreams`, `https://mobilizon.org/ns#`, `mz:joinMode`} {
		if !strings.Contains(s, want) {
			t.Errorf("serialised event missing %q: %s", want, s)
		}
	}
}

// jsonKind names the JSON type of a decoded value so shapes can be compared
// without caring about the values themselves.
func jsonKind(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "null"
}

// subsetOf fails the test for every key in got that a real Mobilizon object
// (want) does not have, or that has a different JSON type. Mobilizon emits some
// keys as null; those are not compared by type.
func subsetOf(t *testing.T, path string, got, want map[string]any) {
	t.Helper()
	for k, gv := range got {
		wv, ok := want[k]
		if !ok {
			t.Errorf("%s.%s is not a key a real Mobilizon object has", path, k)
			continue
		}
		if wv != nil && jsonKind(gv) != jsonKind(wv) {
			t.Errorf("%s.%s is a %s, real Mobilizon sends a %s", path, k, jsonKind(gv), jsonKind(wv))
		}
	}
}

func TestEventToAPEvent_MatchesRealMobilizonShape(t *testing.T) {
	raw, err := os.ReadFile("testdata/mobilizon_event.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden map[string]any
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}

	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	place := createPlace(t, app, "Golden Hall", "1 Test St", "Perth", 56.39, -3.43, "approved")
	tag := createTag(t, app, "gig", "approved")
	ev := createPublishedEvent(t, app, "Golden sample", time.Now().Add(time.Hour))
	ev.Set("place", place.Id)
	ev.Set("tags", []string{tag.Id})
	ev.Set("end_datetime", time.Now().Add(2*time.Hour))

	b, err := json.Marshal(apEvent(t, app, ev))
	if err != nil {
		t.Fatal(err)
	}
	var ours map[string]any
	if err := json.Unmarshal(b, &ours); err != nil {
		t.Fatal(err)
	}

	ctx := func(m map[string]any) map[string]any {
		for _, e := range m["@context"].([]any) {
			if obj, ok := e.(map[string]any); ok {
				return obj
			}
		}
		t.Fatal("@context has no term-definition object")
		return nil
	}
	gctx, octx := ctx(golden), ctx(ours)
	if octx["mz"] != gctx["mz"] {
		t.Errorf("mz namespace = %v, real Mobilizon uses %v", octx["mz"], gctx["mz"])
	}
	if j, ok := octx["joinMode"].(map[string]any); !ok || j["@id"] != "mz:joinMode" {
		t.Errorf("joinMode term = %v", octx["joinMode"])
	}

	delete(ours, "@context")
	subsetOf(t, "event", ours, golden)

	subsetOf(t, "location", ours["location"].(map[string]any), golden["location"].(map[string]any))
	subsetOf(t, "location.address",
		ours["location"].(map[string]any)["address"].(map[string]any),
		golden["location"].(map[string]any)["address"].(map[string]any))
	subsetOf(t, "tag[0]", ours["tag"].([]any)[0].(map[string]any), golden["tag"].([]any)[0].(map[string]any))
}

func TestEventObject_PublishedEventIsServedAsEvent(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	ev := createPublishedEvent(t, app, "Live One", time.Now().Add(time.Hour))

	body, status := EventObject(app, testBase, ev.Id)

	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "Event" || got["name"] != "Live One" {
		t.Errorf("body = %s", body)
	}
}

func TestEventObject_CancelledEventIsStillServed(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	ev := createPublishedEvent(t, app, "Called Off", time.Now().Add(time.Hour))
	ev.Set("status", "cancelled")
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}

	body, status := EventObject(app, testBase, ev.Id)

	if status != 200 || !strings.Contains(string(body), `"status":"CANCELLED"`) {
		t.Errorf("status = %d, body = %s", status, body)
	}
}

func TestEventObject_UnavailableEventsAreGoneTombstones(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()
	setTimezone(t, app, "Europe/London")
	draft := createPublishedEvent(t, app, "Not Public", time.Now().Add(time.Hour))
	draft.Set("status", "draft")
	if err := app.Save(draft); err != nil {
		t.Fatal(err)
	}

	for name, id := range map[string]string{"draft": draft.Id, "missing": "doesnotexist1"} {
		body, status := EventObject(app, testBase, id)
		if status != 410 {
			t.Errorf("%s: status = %d, want 410", name, status)
		}
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got["type"] != "Tombstone" || got["id"] != testBase+"/ap/events/"+id || got["formerType"] != "Event" {
			t.Errorf("%s: body = %s", name, body)
		}
		if strings.Contains(string(body), "Not Public") {
			t.Errorf("%s: tombstone leaks the event title", name)
		}
	}
}
