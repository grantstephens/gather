package hooks_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"gather/internal/api"
	"gather/internal/hooks"
	"gather/internal/textutil"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func newEvent(t *testing.T, app core.App, title, description string) *core.Record {
	t.Helper()
	events, err := app.FindCollectionByNameOrId("events")
	if err != nil {
		t.Fatal(err)
	}
	ev := core.NewRecord(events)
	ev.Set("title", title)
	ev.Set("description", description)
	ev.Set("start_datetime", time.Now().Add(24*time.Hour))
	ev.Set("status", "draft")
	return ev
}

func TestTextHooksDecodeOnSave(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	hooks.RegisterTextHooks(app)

	ev := newEvent(t, app, "Beatrix Potter&#039;s Walk", "Tom &amp;amp; Jerry<br><br>Cost: £5")
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}
	got, _ := app.FindRecordById("events", ev.Id)
	if got.GetString("title") != "Beatrix Potter's Walk" {
		t.Errorf("title = %q", got.GetString("title"))
	}
	if got.GetString("description") != "Tom & Jerry<br><br>Cost: £5" {
		t.Errorf("description = %q (markup must be preserved)", got.GetString("description"))
	}

	got.Set("title", "St John&#39;s Kirk Concert")
	if err := app.Save(got); err != nil {
		t.Fatal(err)
	}
	again, _ := app.FindRecordById("events", ev.Id)
	if again.GetString("title") != "St John's Kirk Concert" {
		t.Errorf("update title = %q", again.GetString("title"))
	}
}

func TestTextHooksDecodePlacesAndTags(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	hooks.RegisterTextHooks(app)

	places, _ := app.FindCollectionByNameOrId("places")
	p := core.NewRecord(places)
	p.Set("name", "St John&#39;s Kirk")
	p.Set("status", "approved")
	if err := app.Save(p); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.FindRecordById("places", p.Id); got.GetString("name") != "St John's Kirk" {
		t.Errorf("place name = %q", got.GetString("name"))
	}

	tags, _ := app.FindCollectionByNameOrId("tags")
	tg := core.NewRecord(tags)
	tg.Set("name", "rock&amp;roll")
	tg.Set("status", "approved")
	if err := app.Save(tg); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.FindRecordById("tags", tg.Id); got.GetString("name") != "rock&roll" {
		t.Errorf("tag name = %q", got.GetString("name"))
	}
}

func TestDecodeAllExistingRewritesOldRowsButNotSlugs(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()

	// hooks NOT registered: simulates rows scraped before the fix
	ev := newEvent(t, app, "Potter&#039;s Walk", "plain")
	ev.Set("slug", "potter-039-s-walk-abc12345")
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.FindRecordById("events", ev.Id); got.GetString("title") != "Potter&#039;s Walk" {
		t.Fatalf("precondition: title should still be raw, got %q", got.GetString("title"))
	}

	if err := textutil.DecodeAllExisting(app); err != nil {
		t.Fatal(err)
	}
	got, _ := app.FindRecordById("events", ev.Id)
	if got.GetString("title") != "Potter's Walk" {
		t.Errorf("title = %q", got.GetString("title"))
	}
	if got.GetString("slug") != "potter-039-s-walk-abc12345" {
		t.Errorf("slug must never change, got %q", got.GetString("slug"))
	}
	if err := textutil.DecodeAllExisting(app); err != nil {
		t.Fatalf("must be idempotent: %v", err)
	}
}

func TestCostAndSourceAreParsedFromBoilerplate(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	hooks.RegisterTextHooks(app)

	ev := newEvent(t, app, "Concert", "Lovely.<br><br>Cost: £5<br>Source: https://x.test/e")
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}
	got, _ := app.FindRecordById("events", ev.Id)
	if got.GetString("cost") != "£5" || got.GetString("source_url") != "https://x.test/e" {
		t.Errorf("cost=%q source_url=%q", got.GetString("cost"), got.GetString("source_url"))
	}

	// an explicit value is never overwritten by the boilerplate
	ev2 := newEvent(t, app, "Concert 2", "Lovely.<br><br>Cost: £5")
	ev2.Set("cost", "Free for members")
	if err := app.Save(ev2); err != nil {
		t.Fatal(err)
	}
	got2, _ := app.FindRecordById("events", ev2.Id)
	if got2.GetString("cost") != "Free for members" {
		t.Errorf("explicit cost overwritten: %q", got2.GetString("cost"))
	}
}

func TestBackfillCostSource(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()

	ev := newEvent(t, app, "Old", "x<br><br>Cost: Free<br>Source: https://old.test/e")
	if err := app.Save(ev); err != nil { // hooks not registered: fields stay empty
		t.Fatal(err)
	}
	if got, _ := app.FindRecordById("events", ev.Id); got.GetString("cost") != "" {
		t.Fatalf("precondition: cost should be empty, got %q", got.GetString("cost"))
	}
	if err := textutil.BackfillCostSource(app); err != nil {
		t.Fatal(err)
	}
	got, _ := app.FindRecordById("events", ev.Id)
	if got.GetString("cost") != "Free" || got.GetString("source_url") != "https://old.test/e" {
		t.Errorf("cost=%q source_url=%q", got.GetString("cost"), got.GetString("source_url"))
	}
}

func TestLongCostLineDoesNotStopTheEventSaving(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	hooks.RegisterTextHooks(app)

	long := strings.Repeat("Adults £10, children free. ", 20) // ~540 chars
	ev := newEvent(t, app, "Concert", "Nice.<br><br>Cost: "+long)
	if err := app.Save(ev); err != nil {
		t.Fatalf("a long Cost: line must never make the save fail: %v", err)
	}
	got, _ := app.FindRecordById("events", ev.Id)
	if n := utf8.RuneCountInString(got.GetString("cost")); n == 0 || n > 200 {
		t.Errorf("cost should be kept but capped at 200 characters, got %d", n)
	}
	if !utf8.ValidString(got.GetString("cost")) {
		t.Errorf("cost truncation must not split a multi-byte character")
	}
}
