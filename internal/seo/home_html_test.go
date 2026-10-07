package seo

import (
	"strings"
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func TestBuildHomeHTML(t *testing.T) {
	result := buildHomeHTML("Perthshire Events", "Community calendar for Perthshire", "https://example.com/api/files/settings/1/favicon.webp", "https://example.com", []HomeEvent{
		{Slug: "abc123", Title: "Summer Fest", StartDate: "Saturday, 1 June 2026"},
		{Slug: "def456", Title: "Folk Night", StartDate: "Sunday, 2 June 2026"},
	}, HomeBrowse{})

	if !strings.Contains(result, "<title>What&#39;s on in Perthshire - Perthshire Events</title>") {
		t.Errorf("home title should lead with what the site is for, got:\n%s", result)
	}
	if !strings.Contains(result, "<h1>What&#39;s on in Perthshire</h1>") {
		t.Error("home h1 should say what's on in the region")
	}
	if !strings.Contains(result, `<meta name="description"`) {
		t.Error("home HTML should contain meta description tag")
	}
	if !strings.Contains(result, "Community calendar for Perthshire") {
		t.Error("home HTML should contain instance description")
	}
	if !strings.Contains(result, `href="https://example.com/event/abc123"`) {
		t.Error("home HTML should link to events")
	}
	if !strings.Contains(result, "Summer Fest") {
		t.Error("home HTML should contain event titles")
	}
	if !strings.Contains(result, `og:title`) {
		t.Error("home HTML should contain og:title")
	}
	if !strings.Contains(result, `og:site_name`) {
		t.Error("home HTML should contain og:site_name")
	}
	if !strings.Contains(result, `rel="canonical"`) {
		t.Error("home HTML should contain canonical tag")
	}
	if !strings.Contains(result, `type="application/rss+xml"`) {
		t.Error("home HTML should contain RSS link")
	}
	if !strings.Contains(result, `og:image`) {
		t.Error("home HTML should contain og:image when favicon is set")
	}
}

func TestBuildHomeHTMLNoEvents(t *testing.T) {
	result := buildHomeHTML("Gather", "", "", "https://example.com", nil, HomeBrowse{})
	if !strings.Contains(result, "<title>Gather</title>") {
		t.Error("home HTML should still render title with no events")
	}
	if !strings.Contains(result, "No upcoming events") {
		t.Error("home HTML should show no-events message")
	}
}

func TestBuildHomeHTMLXSSEscape(t *testing.T) {
	result := buildHomeHTML(`<script>alert("xss")</script>`, "", "", "https://example.com", nil, HomeBrowse{})
	if strings.Contains(result, "<script>alert") {
		t.Error("home HTML should escape XSS in instance name")
	}
}

func TestBuildHomeHTMLNonEventsInstanceKeepsPlainTitle(t *testing.T) {
	result := buildHomeHTML("Gather", "", "", "https://example.com", nil, HomeBrowse{})
	if !strings.Contains(result, "<title>Gather</title>") || !strings.Contains(result, "<h1>Gather</h1>") {
		t.Errorf("an instance name without 'Events' keeps the plain title:\n%s", result)
	}
}

func TestBuildHomeHTMLLinksEveryVenueCategoryAndPick(t *testing.T) {
	result := buildHomeHTML("Perthshire Events", "", "", "https://example.com", nil, HomeBrowse{
		Places: []Link{{Text: "Pitlochry Festival Theatre", URL: "https://example.com/place/pitlochry-festival-theatre"}},
		Tags:   []Link{{Text: "Strathearn Arts", URL: "https://example.com/tag/strathearn-arts"}},
		Picks:  []Link{{Text: "What's on 9-11 October", URL: "https://example.com/picks/whats-on-perthshire-9-11-october-2026"}},
	})
	for _, want := range []string{
		`<h2>Browse by venue</h2>`, `href="https://example.com/place/pitlochry-festival-theatre">Pitlochry Festival Theatre</a>`,
		`<h2>Browse by category</h2>`, `href="https://example.com/tag/strathearn-arts">Strathearn Arts</a>`,
		`<h2>Weekend picks</h2>`, `href="https://example.com/picks/whats-on-perthshire-9-11-october-2026">What&#39;s on 9-11 October</a>`,
	} {
		if !strings.Contains(result, want) {
			t.Errorf("missing %q in:\n%s", want, result)
		}
	}
}

func TestBuildHomeHTMLOmitsEmptyBrowseSections(t *testing.T) {
	result := buildHomeHTML("Perthshire Events", "", "", "https://example.com", nil, HomeBrowse{})
	for _, banned := range []string{"Browse by venue", "Browse by category", "Weekend picks"} {
		if strings.Contains(result, banned) {
			t.Errorf("empty section %q must be omitted", banned)
		}
	}
}

func TestGenerateHomeHTMLBrowseListsOnlyLivePages(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Now()

	places, _ := app.FindCollectionByNameOrId("places")
	tags, _ := app.FindCollectionByNameOrId("tags")
	mk := func(col *core.Collection, name string) *core.Record {
		r := core.NewRecord(col)
		r.Set("name", name)
		r.Set("status", "approved")
		if err := app.SaveNoValidate(r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	live, quiet := mk(places, "Live Hall"), mk(places, "Quiet Hall")
	liveTag, deadTag := mk(tags, "theatre"), mk(tags, "ghost")

	upcoming := mkEvent(t, app, "Upcoming", now.Add(48*time.Hour), func(r *core.Record) {
		r.Set("place", live.Id)
		r.Set("tags", []string{liveTag.Id})
	})
	past := mkEvent(t, app, "Past", now.AddDate(0, 0, -30), func(r *core.Record) {
		r.Set("place", quiet.Id)
		r.Set("tags", []string{deadTag.Id})
	})

	picks, _ := app.FindCollectionByNameOrId("picks")
	mkPick := func(slug string, ids ...string) {
		p := core.NewRecord(picks)
		p.Set("title", slug)
		p.Set("slug", slug)
		p.Set("hidden", false)
		p.Set("events", ids)
		if err := app.SaveNoValidate(p); err != nil {
			t.Fatal(err)
		}
	}
	mkPick("this-weekend", upcoming.Id)
	mkPick("old-weekend", past.Id)

	html, err := GenerateHomeHTML(app, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	out := string(html)
	for _, want := range []string{
		"Live Hall", "/place/" + PlaceRef(live), "/tag/theatre", ">Theatre</a>", "/picks/this-weekend",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, banned := range []string{"Quiet Hall", "/tag/ghost", "/picks/old-weekend"} {
		if strings.Contains(out, banned) {
			t.Errorf("must not link %q (its page would 404 or is retired)", banned)
		}
	}
}
