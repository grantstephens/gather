package rss

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "gather/migrations"
)

func newTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return app
}

func createTestEvent(t *testing.T, app *tests.TestApp, title, description string, start time.Time, slug string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("events")
	if err != nil {
		t.Fatalf("find events collection: %v", err)
	}

	userCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}
	user := core.NewRecord(userCol)
	user.Set("email", "rss-test@example.com")
	user.Set("password", "testpass123")
	user.Set("passwordConfirm", "testpass123")
	user.Set("role", "user")
	_ = app.Save(user)

	r := core.NewRecord(col)
	r.Set("title", title)
	r.Set("description", description)
	r.Set("start_datetime", start)
	r.Set("status", "published")
	r.Set("author", user.Id)
	if slug != "" {
		r.Set("slug", slug)
	}
	if err := app.Save(r); err != nil {
		t.Fatalf("save event: %v", err)
	}
	return r
}

func parseRSS(t *testing.T, data []byte) RSS {
	t.Helper()
	// strip <?xml header before unmarshalling
	s := strings.TrimPrefix(string(data), xml.Header)
	var feed RSS
	if err := xml.Unmarshal([]byte(s), &feed); err != nil {
		t.Fatalf("unmarshal RSS: %v", err)
	}
	return feed
}

func TestGenerateFeed_ValidXML(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	if !strings.HasPrefix(string(out), xml.Header) {
		t.Error("output should start with XML declaration")
	}
	feed := parseRSS(t, out)
	if feed.Version != "2.0" {
		t.Errorf("expected RSS version 2.0, got %q", feed.Version)
	}
}

func TestGenerateFeed_ChannelFields(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	s := string(out)

	// Check raw output — Go's xml.Unmarshal doesn't round-trip namespace-prefixed
	// tags reliably, so we assert directly on the serialized form.
	if !strings.Contains(s, "<link>https://example.com</link>") {
		t.Error("missing channel link")
	}
	if !strings.Contains(s, "<language>en</language>") {
		t.Error("missing language")
	}
	if !strings.Contains(s, `href="https://example.com/feed.rss"`) {
		t.Error("missing atom:link href")
	}
	if !strings.Contains(s, `rel="self"`) {
		t.Error("missing atom:link rel=self")
	}
}

func TestGenerateFeed_ItemFields(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	start := time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC)
	ev := createTestEvent(t, app, "RSS Test Event", "Event details", start, "")

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	feed := parseRSS(t, out)

	if len(feed.Channel.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(feed.Channel.Items))
	}
	item := feed.Channel.Items[0]

	if item.Title != "RSS Test Event" {
		t.Errorf("item title = %q, want %q", item.Title, "RSS Test Event")
	}
	expectedURL := "https://example.com/event/" + ev.Id
	if item.Link != expectedURL {
		t.Errorf("item link = %q, want %q", item.Link, expectedURL)
	}
	if item.GUID != expectedURL {
		t.Errorf("item guid = %q, want %q", item.GUID, expectedURL)
	}
	if item.Description != "Event details" {
		t.Errorf("item description = %q", item.Description)
	}
	// pubDate should be the start_datetime formatted as RFC1123Z
	expectedPubDate := start.Format(time.RFC1123Z)
	if item.PubDate != expectedPubDate {
		t.Errorf("item pubDate = %q, want %q", item.PubDate, expectedPubDate)
	}
}

func TestGenerateFeed_SlugUsedInURL(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	start := time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC)
	createTestEvent(t, app, "Slug Event", "", start, "my-slug")

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	feed := parseRSS(t, out)

	if len(feed.Channel.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(feed.Channel.Items))
	}
	if feed.Channel.Items[0].Link != "https://example.com/event/my-slug" {
		t.Errorf("expected slug URL, got %q", feed.Channel.Items[0].Link)
	}
}

func TestGenerateFeed_IDUsedWhenNoSlug(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	start := time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC)
	ev := createTestEvent(t, app, "No Slug Event", "", start, "")

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	feed := parseRSS(t, out)

	if len(feed.Channel.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(feed.Channel.Items))
	}
	expectedURL := "https://example.com/event/" + ev.Id
	if feed.Channel.Items[0].Link != expectedURL {
		t.Errorf("expected ID URL %q, got %q", expectedURL, feed.Channel.Items[0].Link)
	}
}

func TestGenerateFeed_OnlyPublishedEvents(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	col, _ := app.FindCollectionByNameOrId("events")
	userCol, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(userCol)
	user.Set("email", "rss-draft@example.com")
	user.Set("password", "testpass123")
	user.Set("passwordConfirm", "testpass123")
	user.Set("role", "user")
	_ = app.Save(user)

	draft := core.NewRecord(col)
	draft.Set("title", "Draft RSS Event")
	draft.Set("start_datetime", time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC))
	draft.Set("status", "draft")
	draft.Set("author", user.Id)
	_ = app.Save(draft)

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	feed := parseRSS(t, out)

	if len(feed.Channel.Items) != 0 {
		t.Errorf("draft events should not appear in feed, got %d items", len(feed.Channel.Items))
	}
}

func TestGenerateFeed_FilterApplied(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	col, _ := app.FindCollectionByNameOrId("events")
	userCol, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(userCol)
	user.Set("email", "rss-filter@example.com")
	user.Set("password", "testpass123")
	user.Set("passwordConfirm", "testpass123")
	user.Set("role", "user")
	_ = app.Save(user)

	r1 := core.NewRecord(col)
	r1.Set("title", "Alpha Event")
	r1.Set("start_datetime", time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC))
	r1.Set("status", "published")
	r1.Set("author", user.Id)
	_ = app.Save(r1)

	r2 := core.NewRecord(col)
	r2.Set("title", "Beta Event")
	r2.Set("start_datetime", time.Date(2026, 6, 16, 18, 0, 0, 0, time.UTC))
	r2.Set("status", "published")
	r2.Set("author", user.Id)
	_ = app.Save(r2)

	out, err := GenerateFeed(app, "https://example.com", "title = 'Alpha Event'")
	if err != nil {
		t.Fatalf("GenerateFeed with filter: %v", err)
	}
	feed := parseRSS(t, out)

	if len(feed.Channel.Items) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(feed.Channel.Items))
	}
	if feed.Channel.Items[0].Title != "Alpha Event" {
		t.Errorf("expected Alpha Event, got %q", feed.Channel.Items[0].Title)
	}
}
