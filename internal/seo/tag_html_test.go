package seo

import (
	"strings"
	"testing"
)

func TestTagPageTitle(t *testing.T) {
	if got := tagPageTitle("strathearn-arts", "Perthshire Events"); got != "Strathearn Arts events in Perthshire - Perthshire Events" {
		t.Errorf("got %q", got)
	}
	if got := tagPageTitle("theatre", "Gather"); got != "Theatre events - Gather" {
		t.Errorf("got %q", got)
	}
}

func TestTagIntro(t *testing.T) {
	if got := tagIntro("theatre", 12); got != "12 upcoming theatre events — dates, venues and booking details." {
		t.Errorf("got %q", got)
	}
	if got := tagIntro("live_music", 1); got != "1 upcoming live music event — dates, venues and booking details." {
		t.Errorf("got %q", got)
	}
}

func TestBuildTagHTMLHeadingAndItemList(t *testing.T) {
	items := []tagEvent{{Title: "Hamlet", URL: "https://x/event/hamlet", StartDate: "Friday 9 October"}}
	html := buildTagHTML(tagPageTitle("strathearn-arts", "Perthshire Events"), "Perthshire Events",
		tagIntro("strathearn-arts", 1), "https://x/tag/strathearn-arts", "https://x", "strathearn-arts", items)
	for _, want := range []string{
		"<title>Strathearn Arts events in Perthshire - Perthshire Events</title>",
		"<h1>Strathearn Arts</h1>",
		`"@type":"ItemList"`, `"name":"Hamlet"`,
		`"name":"Strathearn Arts events"`, // breadcrumb uses the humanised name
		`href="https://x/event/hamlet"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in:\n%s", want, html)
		}
	}
}
