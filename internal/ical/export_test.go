package ical

import (
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

func createTestEvent(t *testing.T, app *tests.TestApp, title, description string, start, end time.Time) *core.Record {
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
	user.Set("email", "ical-test@example.com")
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
	if !end.IsZero() {
		r.Set("end_datetime", end)
	}
	if err := app.Save(r); err != nil {
		t.Fatalf("save event: %v", err)
	}
	return r
}

func TestGenerateFeed_CalendarWrapper(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	s := string(out)

	if !strings.HasPrefix(s, "BEGIN:VCALENDAR\r\n") {
		t.Error("output should start with BEGIN:VCALENDAR")
	}
	if !strings.HasSuffix(s, "END:VCALENDAR\r\n") {
		t.Error("output should end with END:VCALENDAR")
	}
	if !strings.Contains(s, "VERSION:2.0\r\n") {
		t.Error("missing VERSION:2.0")
	}
	if !strings.Contains(s, "PRODID:") {
		t.Error("missing PRODID")
	}
}

func TestGenerateFeed_EventFields(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	start := time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 15, 20, 0, 0, 0, time.UTC)
	ev := createTestEvent(t, app, "Test Event", "Some description", start, end)

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	s := string(out)

	if !strings.Contains(s, "BEGIN:VEVENT\r\n") {
		t.Error("missing BEGIN:VEVENT")
	}
	if !strings.Contains(s, "END:VEVENT\r\n") {
		t.Error("missing END:VEVENT")
	}
	if !strings.Contains(s, "SUMMARY:Test Event\r\n") {
		t.Error("missing SUMMARY")
	}
	if !strings.Contains(s, "DTSTART:20260615T180000Z\r\n") {
		t.Error("wrong DTSTART")
	}
	if !strings.Contains(s, "DTEND:20260615T200000Z\r\n") {
		t.Error("wrong DTEND")
	}
	if !strings.Contains(s, "DESCRIPTION:Some description\r\n") {
		t.Error("missing DESCRIPTION")
	}
	if !strings.Contains(s, "URL:https://example.com/event/"+ev.Id+"\r\n") {
		t.Errorf("missing or wrong URL, event id: %s", ev.Id)
	}
	if !strings.Contains(s, "UID:"+ev.Id+"@https://example.com\r\n") {
		t.Errorf("missing or wrong UID, event id: %s", ev.Id)
	}
}

func TestGenerateFeed_MissingEndDefaultsToOneHour(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	start := time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC)
	createTestEvent(t, app, "No End Event", "", start, time.Time{})

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	s := string(out)

	if !strings.Contains(s, "DTSTART:20260615T180000Z\r\n") {
		t.Error("wrong DTSTART")
	}
	// end should be start + 1 hour
	if !strings.Contains(s, "DTEND:20260615T190000Z\r\n") {
		t.Error("DTEND should default to start + 1 hour")
	}
}

func TestGenerateFeed_OnlyPublishedEvents(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	col, _ := app.FindCollectionByNameOrId("events")
	userCol, _ := app.FindCollectionByNameOrId("users")
	user := core.NewRecord(userCol)
	user.Set("email", "ical-draft@example.com")
	user.Set("password", "testpass123")
	user.Set("passwordConfirm", "testpass123")
	user.Set("role", "user")
	_ = app.Save(user)

	draft := core.NewRecord(col)
	draft.Set("title", "Draft Event")
	draft.Set("start_datetime", time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC))
	draft.Set("status", "draft")
	draft.Set("author", user.Id)
	_ = app.Save(draft)

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	s := string(out)

	if strings.Contains(s, "Draft Event") {
		t.Error("draft events should not appear in feed")
	}
}

func TestGenerateSingleEvent(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	start := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	ev := createTestEvent(t, app, "Single Export", "Details here", start, end)

	out, err := GenerateSingleEvent(app, "https://example.com", ev.Id)
	if err != nil {
		t.Fatalf("GenerateSingleEvent: %v", err)
	}
	s := string(out)

	if !strings.Contains(s, "BEGIN:VCALENDAR\r\n") {
		t.Error("missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(s, "SUMMARY:Single Export\r\n") {
		t.Error("missing SUMMARY")
	}
	if !strings.Contains(s, "DTSTART:20260620T100000Z\r\n") {
		t.Error("wrong DTSTART")
	}
	if !strings.Contains(s, "DTEND:20260620T120000Z\r\n") {
		t.Error("wrong DTEND")
	}
}

func TestGenerateSingleEvent_NotFound(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	_, err := GenerateSingleEvent(app, "https://example.com", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent event")
	}
}

func TestEscapeICS(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"semi;colon", "semi\\;colon"},
		{"com,ma", "com\\,ma"},
		{"new\nline", "new\\nline"},
		{"back\\slash", "back\\\\slash"},
	}
	for _, c := range cases {
		got := escapeICS(c.input)
		if got != c.want {
			t.Errorf("escapeICS(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestStripHTML(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"plain text", "plain text"},
		{"<p>paragraph</p>", "paragraph"},
		{"<b>bold</b> text", "bold text"},
		{"<a href=\"url\">link</a>", "link"},
		{"no tags here", "no tags here"},
	}
	for _, c := range cases {
		got := stripHTML(c.input)
		if got != c.want {
			t.Errorf("stripHTML(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestGenerateFeed_HTMLStrippedFromDescription(t *testing.T) {
	app := newTestApp(t)
	defer app.Cleanup()

	start := time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC)
	createTestEvent(t, app, "HTML Event", "<p>Rich <b>content</b></p>", start, time.Time{})

	out, err := GenerateFeed(app, "https://example.com", "")
	if err != nil {
		t.Fatalf("GenerateFeed: %v", err)
	}
	s := string(out)

	if strings.Contains(s, "<p>") || strings.Contains(s, "<b>") {
		t.Error("HTML tags should be stripped from DESCRIPTION")
	}
	if !strings.Contains(s, "DESCRIPTION:Rich content\r\n") {
		t.Error("stripped text should be in DESCRIPTION")
	}
}
