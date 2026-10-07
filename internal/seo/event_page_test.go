package seo

import (
	"strings"
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func basePage() *EventPage {
	return &EventPage{
		Title: "Beatrix Potter's Walk", InstanceName: "Perthshire Events", BaseURL: "https://site.test",
		URL:         "https://site.test/event/bp-1",
		Description: "Para one.\nPara two & <b>bold</b>.", MetaDescription: "Para one.",
		State: Upcoming, StartISO: "2026-10-10T19:30:00+01:00", DateText: "Saturday 10 October 2026, 7:30 PM",
		Venue:   &Link{Text: "Birnam Arts", URL: "https://site.test/place/birnam-arts"},
		Address: "Station Rd, Birnam",
		Tags:    []Link{{Text: "walks", URL: "https://site.test/tag/walks"}},
		Cost:    "£5", BookingURL: "https://src.test/book",
		MetaTags: `<meta property="og:title" content="x">`, JSONLD: `{"@type":"Event"}`,
	}
}

func TestRenderEventHTMLBasics(t *testing.T) {
	html := RenderEventHTML(basePage())
	for _, want := range []string{
		"<title>Beatrix Potter&#39;s Walk - Perthshire Events</title>",
		`<link rel="canonical" href="https://site.test/event/bp-1">`,
		`<h1>Beatrix Potter&#39;s Walk</h1>`,
		`<a href="https://site.test/place/birnam-arts">Birnam Arts</a>`,
		`<a href="https://site.test/tag/walks">walks</a>`,
		`<time datetime="2026-10-10T19:30:00+01:00">Saturday 10 October 2026, 7:30 PM</time>`,
		"<p>Para two &amp; &lt;b&gt;bold&lt;/b&gt;.</p>",
		`href="https://src.test/book"`, "£5", "application/ld+json",
		`<a href="https://site.test/">Home</a>`, `<a href="https://site.test/picks">Weekend picks</a>`,
		`<meta name="description" content="Para one.">`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, "&amp;#") || strings.Contains(html, "noindex") || strings.Contains(html, "has ended") {
		t.Errorf("unexpected content in an upcoming page")
	}
}

func TestRenderVenueWithoutLiveLinkIsPlainText(t *testing.T) {
	p := basePage()
	p.Venue = &Link{Text: "Birnam Arts"}
	html := RenderEventHTML(p)
	if strings.Contains(html, "/place/") || !strings.Contains(html, "Birnam Arts") {
		t.Errorf("a venue without a live page must render as text only:\n%s", html)
	}
}

func TestRenderSeriesTitleIsDateQualifiedAndListsOtherDates(t *testing.T) {
	p := basePage()
	p.SeriesLabel = "Sat 10 Oct"
	p.OtherDates = []Link{{Text: "Sat 17 Oct", URL: "https://site.test/event/bp-2"}}
	html := RenderEventHTML(p)
	if !strings.Contains(html, "<title>Beatrix Potter&#39;s Walk – Sat 10 Oct - Perthshire Events</title>") {
		t.Errorf("series title not date-qualified:\n%s", html)
	}
	if !strings.Contains(html, "Other dates") || !strings.Contains(html, `href="https://site.test/event/bp-2"`) {
		t.Errorf("other dates missing")
	}
}

func TestRenderEndedAndRetired(t *testing.T) {
	p := basePage()
	p.State = Ended
	p.Alternatives = []Link{{Text: "Next thing", URL: "https://site.test/event/next"}}
	html := RenderEventHTML(p)
	if !strings.Contains(html, "This event has ended") || !strings.Contains(html, "Next thing") {
		t.Errorf("ended banner/alternatives missing")
	}
	if strings.Contains(html, "noindex") || p.NoIndex() {
		t.Errorf("ended (within grace) stays indexable")
	}
	p.State = Retired
	if !p.NoIndex() || !strings.Contains(RenderEventHTML(p), `<meta name="robots" content="noindex,follow">`) {
		t.Errorf("retired must be noindex,follow")
	}
}

func TestRenderEscapesEverythingOnce(t *testing.T) {
	p := basePage()
	p.Title = `<script>alert("x")</script>`
	p.Venue = &Link{Text: `A&B "Hall"`, URL: `https://site.test/place/a"b`}
	html := RenderEventHTML(p)
	if strings.Contains(html, "<script>alert") || strings.Contains(html, `place/a"b`) {
		t.Errorf("unescaped content:\n%s", html)
	}
	if !strings.Contains(html, "A&amp;B &#34;Hall&#34;") {
		t.Errorf("venue text not escaped exactly once:\n%s", html)
	}
}

func TestDateRangeText(t *testing.T) {
	s := time.Date(2026, 10, 10, 18, 30, 0, 0, time.UTC) // 19:30 BST
	same := time.Date(2026, 10, 10, 20, 0, 0, 0, time.UTC)
	next := time.Date(2026, 10, 12, 20, 0, 0, 0, time.UTC)
	if got := dateRangeText(s, time.Time{}); got != "Saturday 10 October 2026, 7:30 PM" {
		t.Errorf("no end: %q", got)
	}
	if got := dateRangeText(s, same); got != "Saturday 10 October 2026, 7:30 PM – 9:00 PM" {
		t.Errorf("same day: %q", got)
	}
	if got := dateRangeText(s, next); got != "Saturday 10 October 2026, 7:30 PM – Monday 12 October 2026, 9:00 PM" {
		t.Errorf("multi day: %q", got)
	}
}

// ---- BuildEventPage (database-backed) ----

func mkEvent(t *testing.T, app core.App, title string, start time.Time, mutate func(*core.Record)) *core.Record {
	t.Helper()
	events, _ := app.FindCollectionByNameOrId("events")
	ev := core.NewRecord(events)
	ev.Set("title", title)
	ev.Set("description", "A reasonably long description that explains what the event is about in some detail.")
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

func TestBuildEventPage(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Now()
	places, _ := app.FindCollectionByNameOrId("places")
	tags, _ := app.FindCollectionByNameOrId("tags")

	place := core.NewRecord(places)
	place.Set("name", "Birnam Arts")
	place.Set("address", "Station Rd")
	place.Set("status", "approved")
	if err := app.SaveNoValidate(place); err != nil {
		t.Fatal(err)
	}
	quiet := core.NewRecord(places) // approved but nothing upcoming: its page would 404
	quiet.Set("name", "Quiet Hall")
	quiet.Set("status", "approved")
	if err := app.SaveNoValidate(quiet); err != nil {
		t.Fatal(err)
	}
	live, dead := core.NewRecord(tags), core.NewRecord(tags)
	live.Set("name", "walks")
	live.Set("status", "approved")
	dead.Set("name", "ghost-tag")
	dead.Set("status", "approved")
	for _, r := range []*core.Record{live, dead} {
		if err := app.SaveNoValidate(r); err != nil {
			t.Fatal(err)
		}
	}

	atVenue := func(r *core.Record) { r.Set("place", place.Id); r.Set("tags", []string{live.Id}) }
	main1 := mkEvent(t, app, "Potter Walk", now.Add(24*time.Hour), func(r *core.Record) {
		atVenue(r)
		r.Set("cost", "£5")
		r.Set("source_url", "https://src.test/book")
	})
	mkEvent(t, app, "Potter Walk", now.Add(8*24*time.Hour), atVenue)  // sibling
	mkEvent(t, app, "Potter Walk", now.Add(15*24*time.Hour), atVenue) // sibling
	mkEvent(t, app, "Other Show", now.Add(3*24*time.Hour), atVenue)   // more at venue

	page, err := BuildEventPage(app, main1, "https://site.test", now)
	if err != nil {
		t.Fatal(err)
	}
	if page.State != Upcoming || page.NoIndex() {
		t.Errorf("state = %v", page.State)
	}
	if page.Venue == nil || page.Venue.Text != "Birnam Arts" || page.Venue.URL != "https://site.test/place/"+PlaceRef(place) {
		t.Errorf("venue = %#v", page.Venue)
	}
	if len(page.Tags) != 1 || page.Tags[0].Text != "walks" {
		t.Errorf("only tags with a live page may be linked, got %#v", page.Tags)
	}
	if page.SeriesLabel == "" || len(page.OtherDates) != 2 {
		t.Errorf("series label %q, other dates %d (want 2)", page.SeriesLabel, len(page.OtherDates))
	}
	if len(page.MoreAtVenue) != 1 || page.MoreAtVenue[0].Text != "Other Show" {
		t.Errorf("more at venue = %#v", page.MoreAtVenue)
	}
	if page.Cost != "£5" || page.BookingURL != "https://src.test/book" {
		t.Errorf("cost/booking = %q / %q", page.Cost, page.BookingURL)
	}
	if !strings.Contains(page.JSONLD, `"offers"`) || !strings.Contains(page.MetaTags, "og:title") {
		t.Errorf("jsonld/meta not populated")
	}

	// a venue with no upcoming events renders as plain text, never a dead link
	lonely := mkEvent(t, app, "Lonely", now.Add(-5*24*time.Hour), func(r *core.Record) {
		r.Set("place", quiet.Id)
		r.Set("tags", []string{dead.Id}) // only ever used on a past event: its page would 404
	})
	lp, _ := BuildEventPage(app, lonely, "https://site.test", now)
	if len(lp.Tags) != 0 {
		t.Errorf("a tag with no upcoming events must not be linked, got %#v", lp.Tags)
	}
	if lp.Venue == nil || lp.Venue.URL != "" {
		t.Errorf("venue with no upcoming events must not be linked: %#v", lp.Venue)
	}
	if lp.State != Ended || len(lp.Alternatives) == 0 {
		t.Errorf("ended event needs state Ended (got %v) and alternatives (got %d)", lp.State, len(lp.Alternatives))
	}

	old := mkEvent(t, app, "Ancient", now.AddDate(0, 0, -20), nil)
	op, _ := BuildEventPage(app, old, "https://site.test", now)
	if op.State != Retired || !op.NoIndex() {
		t.Errorf("20-day-old event must be retired/noindex, got %v", op.State)
	}

	// picks post referencing the event
	picks, _ := app.FindCollectionByNameOrId("picks")
	pk := core.NewRecord(picks)
	pk.Set("title", "This weekend")
	pk.Set("slug", "this-weekend")
	pk.Set("hidden", false)
	pk.Set("events", []string{main1.Id})
	if err := app.SaveNoValidate(pk); err != nil {
		t.Fatal(err)
	}
	fp, _ := BuildEventPage(app, main1, "https://site.test", now)
	if fp.FeaturedIn == nil || fp.FeaturedIn.URL != "https://site.test/picks/this-weekend" {
		t.Errorf("featured in = %#v", fp.FeaturedIn)
	}
}
