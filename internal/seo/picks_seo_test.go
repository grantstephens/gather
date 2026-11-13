package seo

import (
	"strings"
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

func TestPicksIndexAdvertisesEmailOnlyWhenOffered(t *testing.T) {
	items := []picksListItem{{Title: "x", URL: "https://s.test/picks/x"}}
	on := buildPicksIndexHTML("T", "Perthshire Events", "d", "https://s.test/picks", "https://s.test", items, true)
	for _, want := range []string{"What&#39;s on this weekend", `href="https://s.test/"`, "by email"} {
		if !strings.Contains(on, want) {
			t.Errorf("signup blurb missing %q in:\n%s", want, on)
		}
	}
	off := buildPicksIndexHTML("T", "Perthshire Events", "d", "https://s.test/picks", "https://s.test", items, false)
	if strings.Contains(off, "by email") {
		t.Error("blurb must not appear when the newsletter is unavailable")
	}
}

func TestGeneratePicksIndexUsesNewsletterHook(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	orig := NewsletterOffered
	defer func() { NewsletterOffered = orig }()

	NewsletterOffered = func(core.App, string) bool { return true }
	html, _ := GeneratePicksIndexHTML(app, "https://s.test")
	if !strings.Contains(string(html), "by email") {
		t.Error("hook true: blurb expected")
	}
	NewsletterOffered = func(core.App, string) bool { return false }
	html, _ = GeneratePicksIndexHTML(app, "https://s.test")
	if strings.Contains(string(html), "by email") {
		t.Error("hook false: no blurb")
	}
}
