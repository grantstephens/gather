package seo

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestBuildPicksHTML_LongBlurb(t *testing.T) {
	const base = "https://example.com"
	blurbHTML := renderMarkdownHTML(sampleLongBlurb, base)
	description := truncateText(StripMarkdownInline(sampleLongBlurb), 160)
	out := buildPicksHTML(
		"Weekend picks", "Gather", blurbHTML, description, base+"/picks/x", base,
		"", "image/webp", time.Now(), time.Now(),
		[]picksEvent{{Title: "Swan Lake", URL: base + "/event/swan-lake-hmruxf7x", Venue: "Pitlochry"}},
	)

	for _, want := range []string{
		"<h3>Poetry Brunch, Birnam Arts, Birnam</h3>",
		`<a href="https://example.com/event/poetry-brunch-l9lbpw21"`,
		"<h2>This weekend's picks</h2>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Count(out, "<p>") < 3 {
		t.Errorf("expected at least 3 <p> in body, got %d", strings.Count(out, "<p>"))
	}

	m := regexp.MustCompile(`<meta name="description" content="([^"]*)">`).FindStringSubmatch(out)
	if m == nil {
		t.Fatal("no meta description")
	}
	if len(m[1]) > 163 {
		t.Errorf("description too long: %d", len(m[1]))
	}
	for _, bad := range []string{"###", "](", "**", "\n"} {
		if strings.Contains(m[1], bad) {
			t.Errorf("description contains %q: %q", bad, m[1])
		}
	}
	if !strings.HasPrefix(m[1], "Birnam Book Festival turns five this weekend") {
		t.Errorf("unexpected description: %q", m[1])
	}
}
