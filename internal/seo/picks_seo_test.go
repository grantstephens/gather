package seo

import (
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func TestPicksIndexTitle(t *testing.T) {
	if got := picksIndexTitle("Perthshire Events"); got != "What's on in Perthshire: weekend picks - Perthshire Events" {
		t.Errorf("got %q", got)
	}
	if got := picksIndexTitle("Gather"); got != "Weekend Picks - Gather" {
		t.Errorf("got %q", got)
	}
}

func TestPickIsRetired(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Now()
	events, _ := app.FindCollectionByNameOrId("events")
	mkEvent := func(start time.Time) string {
		ev := core.NewRecord(events)
		ev.Set("title", "e")
		ev.Set("start_datetime", start)
		ev.Set("status", "published")
		if err := app.SaveNoValidate(ev); err != nil {
			t.Fatal(err)
		}
		return ev.Id
	}
	picks, _ := app.FindCollectionByNameOrId("picks")
	mkPick := func(slug string, hidden bool, ids ...string) {
		p := core.NewRecord(picks)
		p.Set("title", slug)
		p.Set("slug", slug)
		p.Set("hidden", hidden)
		p.Set("events", ids)
		if err := app.SaveNoValidate(p); err != nil {
			t.Fatal(err)
		}
	}
	old, fresh := mkEvent(now.AddDate(0, 0, -30)), mkEvent(now.AddDate(0, 0, 3))
	mkPick("past", false, old)
	mkPick("current", false, fresh)
	mkPick("evergreen", false)
	mkPick("secret", true, old)

	for slug, want := range map[string]bool{"past": true, "current": false, "evergreen": false, "secret": false, "missing": false} {
		if got := PickIsRetired(app, slug, now); got != want {
			t.Errorf("PickIsRetired(%q) = %v, want %v", slug, got, want)
		}
	}
}
