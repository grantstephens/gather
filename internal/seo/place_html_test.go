package seo

import (
	"strings"
	"testing"
	"time"
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
