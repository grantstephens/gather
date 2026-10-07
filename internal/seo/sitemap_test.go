package seo

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func TestBuildSitemapXML(t *testing.T) {
	baseURL := "https://example.com"
	entries := []SitemapEntry{
		{URL: "https://example.com/event/abc123", LastMod: "2026-03-01"},
		{URL: "https://example.com/event/def456", LastMod: "2026-03-10"},
	}

	xml := buildSitemapXML(baseURL, entries)

	if !strings.Contains(xml, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("sitemap should start with XML declaration")
	}
	if !strings.Contains(xml, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`) {
		t.Error("sitemap should have urlset element with correct namespace")
	}
	if !strings.Contains(xml, "<loc>https://example.com/</loc>") {
		t.Error("sitemap should include home page URL")
	}
	if !strings.Contains(xml, "<loc>https://example.com/event/abc123</loc>") {
		t.Error("sitemap should include first event URL")
	}
	if !strings.Contains(xml, "<loc>https://example.com/event/def456</loc>") {
		t.Error("sitemap should include second event URL")
	}
	if !strings.Contains(xml, "</urlset>") {
		t.Error("sitemap should close urlset")
	}
}

func TestBuildRobotsTxt(t *testing.T) {
	baseURL := "https://example.com"
	result := BuildRobotsTxt(baseURL)

	if !strings.Contains(result, "User-agent: *") {
		t.Error("robots.txt should have User-agent wildcard")
	}
	if !strings.Contains(result, "Allow: /") {
		t.Error("robots.txt should allow all paths")
	}
	if !strings.Contains(result, "Sitemap: https://example.com/sitemap.xml") {
		t.Error("robots.txt should include Sitemap directive with full URL")
	}
}

func TestRobotsTxtDisallowsInternalPaths(t *testing.T) {
	result := BuildRobotsTxt("https://example.com")
	for _, path := range []string{"/api/", "/_/", "/feed", "/ics", "/federation", "/embed"} {
		if !strings.Contains(result, "Disallow: "+path) {
			t.Errorf("robots.txt should disallow %s", path)
		}
	}
}

func TestBuildSitemapXMLHasNoChangefreqOrPriority(t *testing.T) {
	xml := buildSitemapXML("https://example.com", []SitemapEntry{{URL: "https://example.com/event/a", LastMod: "2026-03-01"}})
	for _, banned := range []string{"<changefreq>", "<priority>"} {
		if strings.Contains(xml, banned) {
			t.Errorf("sitemap must not contain %s (Google ignores it; it only adds noise)", banned)
		}
	}
	if strings.Count(xml, "<lastmod>") != 2 {
		t.Errorf("home and the entry should both carry lastmod, got:\n%s", xml)
	}
}

func saveEvent(t *testing.T, app core.App, title string, start time.Time, mutate func(*core.Record)) *core.Record {
	t.Helper()
	events, _ := app.FindCollectionByNameOrId("events")
	ev := core.NewRecord(events)
	ev.Set("title", title)
	ev.Set("start_datetime", start)
	ev.Set("status", "published")
	if mutate != nil {
		mutate(ev)
	}
	if err := app.SaveNoValidate(ev); err != nil {
		t.Fatal(err)
	}
	return ev
}

func TestGenerateSitemapListsOnlyLiveURLsWithoutACap(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Now()

	for i := 0; i < 505; i++ {
		saveEvent(t, app, fmt.Sprintf("future %d", i), now.Add(time.Duration(i+1)*time.Hour), nil)
	}
	for i := 0; i < 10; i++ { // ended 3 days ago: still in grace
		saveEvent(t, app, fmt.Sprintf("grace %d", i), now.AddDate(0, 0, -3), nil)
	}
	for i := 0; i < 5; i++ { // ended a month ago: retired
		saveEvent(t, app, fmt.Sprintf("retired %d", i), now.AddDate(0, 0, -30), nil)
	}
	// long-running exhibition that started two months ago but runs on
	saveEvent(t, app, "exhibition", now.AddDate(0, 0, -60), func(r *core.Record) { r.Set("end_datetime", now.AddDate(0, 0, 20)) })
	// weekly series that started 60 days ago and never ends
	saveEvent(t, app, "weekly", now.AddDate(0, 0, -60), func(r *core.Record) { r.Set("recurrence_rule", "FREQ=WEEKLY;COUNT=500") })
	// drafts never appear
	saveEvent(t, app, "draft", now.Add(time.Hour), func(r *core.Record) { r.Set("status", "draft") })

	xml, err := GenerateSitemap(app, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Count(string(xml), "/event/")
	if want := 505 + 10 + 1 + 1; got != want {
		t.Errorf("event URLs = %d, want %d (no 500 cap, no retired, no drafts)", got, want)
	}
}

func TestGenerateSitemapListsCustomPagesAndSkipsRetiredPicks(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Now()

	pages, _ := app.FindCollectionByNameOrId("pages")
	for _, slug := range []string{"about", "contact"} {
		p := core.NewRecord(pages)
		p.Set("title", slug)
		p.Set("slug", slug)
		p.Set("content", "<p>x</p>")
		if err := app.SaveNoValidate(p); err != nil {
			t.Fatal(err)
		}
	}

	old := saveEvent(t, app, "old", now.AddDate(0, 0, -30), nil)
	fresh := saveEvent(t, app, "fresh", now.AddDate(0, 0, 3), nil)
	picks, _ := app.FindCollectionByNameOrId("picks")
	mkPick := func(slug string, ids ...string) {
		p := core.NewRecord(picks)
		p.Set("title", slug)
		p.Set("slug", slug)
		p.Set("events", ids)
		p.Set("hidden", false)
		if err := app.SaveNoValidate(p); err != nil {
			t.Fatal(err)
		}
	}
	mkPick("past-weekend", old.Id)
	mkPick("this-weekend", fresh.Id)
	mkPick("autumn-guide")

	xml, err := GenerateSitemap(app, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	out := string(xml)
	for _, want := range []string{"<loc>https://example.com/about</loc>", "<loc>https://example.com/contact</loc>",
		"<loc>https://example.com/picks</loc>", "/picks/this-weekend<", "/picks/autumn-guide<"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s", want)
		}
	}
	for _, banned := range []string{"/picks/past-weekend<"} {
		if strings.Contains(out, banned) {
			t.Errorf("must not list %s", banned)
		}
	}
}

// Every URL the sitemap lists must actually render for a bot. The old code
// filtered pages on a "hidden" field that does not exist, so every custom page
// 404'd for crawlers while still being listed.
func TestEveryCustomPageInTheSitemapRendersForBots(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	pages, _ := app.FindCollectionByNameOrId("pages")
	p := core.NewRecord(pages)
	p.Set("title", "About")
	p.Set("slug", "about")
	p.Set("content", "<p>About us</p>")
	if err := app.SaveNoValidate(p); err != nil {
		t.Fatal(err)
	}
	html, err := GeneratePageHTML(app, "about", "https://example.com")
	if err != nil {
		t.Fatalf("custom page must render for bots: %v", err)
	}
	if !strings.Contains(string(html), "About us") {
		t.Errorf("page body missing")
	}
}
