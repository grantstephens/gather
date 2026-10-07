package seo

import (
	"testing"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func TestResolvePlace(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	places, _ := app.FindCollectionByNameOrId("places")
	mk := func(name, slug string) *core.Record {
		p := core.NewRecord(places)
		p.Set("name", name)
		p.Set("slug", slug)
		p.Set("status", "approved")
		if err := app.SaveNoValidate(p); err != nil {
			t.Fatal(err)
		}
		return p
	}
	withSlug, noSlug := mk("Birnam Arts", "birnam-arts"), mk("Legacy Hall", "")

	if p, to := ResolvePlace(app, "birnam-arts"); p == nil || p.Id != withSlug.Id || to != "" {
		t.Errorf("slug hit must resolve with no redirect, got %v %q", p, to)
	}
	if p, to := ResolvePlace(app, withSlug.Id); p == nil || p.Id != withSlug.Id || to != "/place/birnam-arts" {
		t.Errorf("id hit with a slug must redirect to the slug, got %v %q", p, to)
	}
	if p, to := ResolvePlace(app, noSlug.Id); p == nil || p.Id != noSlug.Id || to != "" {
		t.Errorf("id hit without a slug must not redirect, got %v %q", p, to)
	}
	if p, to := ResolvePlace(app, "no-such-place"); p != nil || to != "" {
		t.Errorf("unknown ref resolves to nothing, got %v %q", p, to)
	}
}

func TestPlaceRefPrefersSlug(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	places, _ := app.FindCollectionByNameOrId("places")
	p := core.NewRecord(places)
	p.Set("slug", "birnam-arts")
	if got := PlaceRef(p); got != "birnam-arts" {
		t.Errorf("PlaceRef = %q", got)
	}
	q := core.NewRecord(places)
	if got := PlaceRef(q); got != q.Id {
		t.Errorf("PlaceRef without slug must be the id, got %q", got)
	}
}
