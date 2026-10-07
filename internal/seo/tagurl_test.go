package seo

import (
	"strings"
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func TestTagURLEscapesThePathSegment(t *testing.T) {
	cases := map[string]string{
		"walks":      "https://x/tag/walks",
		"C#":         "https://x/tag/C%23",
		"a/b":        "https://x/tag/a%2Fb",
		"live music": "https://x/tag/live%20music",
		"50%":        "https://x/tag/50%25",
	}
	for name, want := range cases {
		if got := tagURL("https://x", name); got != want {
			t.Errorf("tagURL(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestEveryTagLinkAndSitemapEntryIsEscaped(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Now()
	tags, _ := app.FindCollectionByNameOrId("tags")
	var ids []string
	for _, n := range []string{"C#", "a/b", "live music"} {
		tg := core.NewRecord(tags)
		tg.Set("name", n)
		tg.Set("status", "approved")
		if err := app.SaveNoValidate(tg); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, tg.Id)
	}
	ev := mkEvent(t, app, "Tagged", now.Add(48*time.Hour), func(r *core.Record) { r.Set("tags", ids) })

	sitemap, _ := GenerateSitemap(app, "https://x")
	home, _ := GenerateHomeHTML(app, "https://x")
	page, err := BuildEventPage(app, ev, "https://x", now)
	if err != nil {
		t.Fatal(err)
	}
	var eventLinks []string
	for _, l := range page.Tags {
		eventLinks = append(eventLinks, l.URL)
	}
	all := string(sitemap) + "\n" + string(home) + "\n" + strings.Join(eventLinks, "\n")
	for _, raw := range []string{"/tag/C#", "/tag/a/b", "/tag/live music"} {
		if strings.Contains(all, raw) {
			t.Errorf("unescaped tag URL %q leaked into output", raw)
		}
	}
	for _, want := range []string{"/tag/C%23", "/tag/a%2Fb", "/tag/live%20music"} {
		for label, doc := range map[string]string{"sitemap": string(sitemap), "home": string(home), "event page": strings.Join(eventLinks, "\n")} {
			if !strings.Contains(doc, want) {
				t.Errorf("%s is missing %q", label, want)
			}
		}
	}
}
