package seo

import (
	"strings"
	"testing"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func TestCustomPageHasMetaDescriptionFromItsContent(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	pages, _ := app.FindCollectionByNameOrId("pages")
	p := core.NewRecord(pages)
	p.Set("title", "About")
	p.Set("slug", "about")
	p.Set("content", "<h2>Who we are</h2><p>We are a <strong>community</strong> calendar for Perthshire &amp; Kinross.</p>")
	if err := app.SaveNoValidate(p); err != nil {
		t.Fatal(err)
	}
	html, err := GeneratePageHTML(app, "about", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	out := string(html)
	for _, want := range []string{
		`<meta name="description" content="Who we are We are a community calendar for Perthshire &amp; Kinross.">`,
		`<meta property="og:description"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, `content="<`) {
		t.Errorf("description must be plain text, got:\n%s", out)
	}
}

func TestCustomPageWithNoContentStillGetsADescription(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	pages, _ := app.FindCollectionByNameOrId("pages")
	p := core.NewRecord(pages)
	p.Set("title", "Contact")
	p.Set("slug", "contact")
	if err := app.SaveNoValidate(p); err != nil {
		t.Fatal(err)
	}
	html, _ := GeneratePageHTML(app, "contact", "https://example.com")
	if !strings.Contains(string(html), `<meta name="description" content="Contact - `) {
		t.Errorf("fallback description missing:\n%s", html)
	}
}
