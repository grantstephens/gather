package seo

import (
	"strings"
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func TestPlacePageTitle(t *testing.T) {
	if got := placePageTitle("Explorers Garden Pavilion", "Pitlochry", "Perthshire Events"); got != "Events at Explorers Garden Pavilion, Pitlochry - Perthshire Events" {
		t.Errorf("got %q", got)
	}
	if got := placePageTitle("Pitlochry Festival Theatre", "Pitlochry", "Perthshire Events"); got != "Events at Pitlochry Festival Theatre - Perthshire Events" {
		t.Errorf("a city already in the name is not repeated, got %q", got)
	}
	if got := placePageTitle("The Space", "", "Gather"); got != "Events at The Space - Gather" {
		t.Errorf("no city, no trailing comma: %q", got)
	}
}

func TestPlaceIntro(t *testing.T) {
	next := time.Date(2026, 10, 10, 18, 0, 0, 0, time.UTC)
	got := placeIntro("Birnam Arts", "Station Rd", 3, next)
	if got != "3 upcoming events at Birnam Arts, Station Rd. Next: Sat 10 Oct." {
		t.Errorf("got %q", got)
	}
	if got := placeIntro("Birnam Arts", "", 1, next); got != "1 upcoming event at Birnam Arts. Next: Sat 10 Oct." {
		t.Errorf("got %q", got)
	}
}

func TestBuildPlaceHTMLIntroAndItemList(t *testing.T) {
	items := []placeEvent{{Title: "Hamlet", URL: "https://x/event/hamlet", StartDate: "Friday 9 October"}}
	html := buildPlaceHTML("Events at Birnam Arts - Gather", "Gather", "1 upcoming event at Birnam Arts. Next: Fri 9 Oct.",
		"https://x/place/birnam-arts", "https://x", "Birnam Arts", "Station Rd", 0, 0, items)
	for _, want := range []string{
		"<h1>Birnam Arts</h1>", "<p>1 upcoming event at Birnam Arts. Next: Fri 9 Oct.</p>",
		`"@type":"ItemList"`, `"name":"Hamlet"`, `href="https://x/event/hamlet"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in:\n%s", want, html)
		}
	}
}

// Datetimes are stored in UTC. The place/tag pages must compare against UTC "now":
// otherwise, on a server whose TZ is ahead of UTC, an event starting soon makes
// the event page link to a place page that then 404s.
func TestPlaceAndTagPagesExistWhenTheServerClockIsNotUTC(t *testing.T) {
	old := time.Local
	time.Local = time.FixedZone("ahead", 5*3600)
	defer func() { time.Local = old }()

	app := api.NewTestApp(t)
	defer app.Cleanup()
	places, _ := app.FindCollectionByNameOrId("places")
	tags, _ := app.FindCollectionByNameOrId("tags")
	pl := core.NewRecord(places)
	pl.Set("name", "Soon Hall")
	pl.Set("status", "approved")
	tg := core.NewRecord(tags)
	tg.Set("name", "soon")
	tg.Set("status", "approved")
	for _, r := range []*core.Record{pl, tg} {
		if err := app.SaveNoValidate(r); err != nil {
			t.Fatal(err)
		}
	}
	mkEvent(t, app, "Starts in two hours", time.Now().Add(2*time.Hour), func(r *core.Record) {
		r.Set("place", pl.Id)
		r.Set("tags", []string{tg.Id})
	})
	if _, err := GeneratePlaceHTML(app, pl.Id, "https://x"); err != nil {
		t.Errorf("place page must exist for an event starting in two hours: %v", err)
	}
	if _, err := GenerateTagHTML(app, "soon", "https://x"); err != nil {
		t.Errorf("tag page must exist for an event starting in two hours: %v", err)
	}
	if !strings.Contains(string(mustHome(t, app)), "Starts in two hours") {
		t.Errorf("homepage must list it too")
	}
}

func mustHome(t *testing.T, app core.App) []byte {
	t.Helper()
	b, err := GenerateHomeHTML(app, "https://x")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
