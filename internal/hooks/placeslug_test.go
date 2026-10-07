package hooks_test

import (
	"testing"

	"gather/internal/api"
	"gather/internal/hooks"
	"gather/internal/slug"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func newPlace(t *testing.T, app core.App, name string, mutate func(*core.Record)) *core.Record {
	t.Helper()
	places, _ := app.FindCollectionByNameOrId("places")
	p := core.NewRecord(places)
	p.Set("name", name)
	p.Set("status", "approved")
	if mutate != nil {
		mutate(p)
	}
	if err := app.Save(p); err != nil {
		t.Fatal(err)
	}
	got, _ := app.FindRecordById("places", p.Id)
	return got
}

func TestPlaceSlugHook(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	hooks.RegisterTextHooks(app)
	hooks.RegisterPlaceSlugHooks(app)

	a := newPlace(t, app, "The Space", nil)
	b := newPlace(t, app, "The Space", nil)
	if a.GetString("slug") != "the-space" || b.GetString("slug") != "the-space-2" {
		t.Errorf("slugs = %q, %q", a.GetString("slug"), b.GetString("slug"))
	}

	a.Set("name", "A Completely Different Name")
	if err := app.Save(a); err != nil {
		t.Fatal(err)
	}
	if got, _ := app.FindRecordById("places", a.Id); got.GetString("slug") != "the-space" {
		t.Errorf("slug must never change after creation, got %q", got.GetString("slug"))
	}

	if got := newPlace(t, app, "!!!", nil); got.GetString("slug") != "place" {
		t.Errorf("unusable name falls back to 'place', got %q", got.GetString("slug"))
	}
	// places can be created anonymously: a client must not be able to choose its slug
	if got := newPlace(t, app, "Whatever", func(p *core.Record) { p.Set("slug", "birnam-arts") }); got.GetString("slug") != "whatever" {
		t.Errorf("a client-supplied slug must be ignored, got %q", got.GetString("slug"))
	}
	if got := newPlace(t, app, "St John&#39;s Kirk", nil); got.GetString("slug") != "st-john-s-kirk" {
		t.Errorf("entities must be decoded before slugging, got %q", got.GetString("slug"))
	}
}

func TestBackfillPlaceSlugs(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()

	// hooks NOT registered: rows that predate the slug field
	first := newPlace(t, app, "Perth Theatre", nil)
	second := newPlace(t, app, "Perth Theatre", nil)
	keeper := newPlace(t, app, "Old", func(p *core.Record) { p.Set("slug", "keep-me") })
	if first.GetString("slug") != "" {
		t.Fatalf("precondition: slug should be empty, got %q", first.GetString("slug"))
	}

	if err := slug.BackfillPlaces(app); err != nil {
		t.Fatal(err)
	}
	get := func(id string) string { r, _ := app.FindRecordById("places", id); return r.GetString("slug") }
	if get(first.Id) != "perth-theatre" || get(second.Id) != "perth-theatre-2" {
		t.Errorf("slugs = %q, %q (oldest gets the clean slug)", get(first.Id), get(second.Id))
	}
	if get(keeper.Id) != "keep-me" {
		t.Errorf("existing slug must be untouched, got %q", get(keeper.Id))
	}
	if err := slug.BackfillPlaces(app); err != nil {
		t.Fatalf("must be idempotent: %v", err)
	}
}
