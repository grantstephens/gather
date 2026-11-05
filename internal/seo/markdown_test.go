package seo

import (
	"strings"
	"testing"
	"time"
)

func TestRenderMarkdownHTML(t *testing.T) {
	const base = "https://example.com"
	tests := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name:  "long structured blurb",
			input: sampleLongBlurb,
			contains: []string{
				"<h3>Poetry Brunch, Birnam Arts, Birnam</h3>",
				`<a href="https://example.com/event/poetry-brunch-l9lbpw21"`,
				`<a href="https://example.com/event/swan-lake-hmruxf7x"`,
				"<strong>Also on:</strong>",
			},
			notContains: []string{"###", "](/event"},
		},
		{
			name:        "script tag",
			input:       "Hi <script>alert(1)</script> there",
			notContains: []string{"<script", "alert(1)</script>"},
		},
		{
			name:        "img onerror",
			input:       `<img src=x onerror="alert(1)">`,
			notContains: []string{"<img", "onerror"},
		},
		{
			name:        "javascript href",
			input:       "[x](javascript:alert(1))",
			notContains: []string{"javascript:"},
		},
		{
			name:        "protocol-relative href",
			input:       "[x](//evil.com/p)",
			notContains: []string{"evil.com", "https://example.com//"},
		},
		{
			name:     "absolute https href kept",
			input:    "[x](https://other.org/a)",
			contains: []string{`href="https://other.org/a"`},
		},
		{name: "empty", input: "  \n ", notContains: []string{"<p>"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderMarkdownHTML(tt.input, base)
			for _, s := range tt.contains {
				if !strings.Contains(got, s) {
					t.Errorf("missing %q in %q", s, got)
				}
			}
			for _, s := range tt.notContains {
				if strings.Contains(got, s) {
					t.Errorf("unexpected %q in %q", s, got)
				}
			}
		})
	}
}

func TestRenderMarkdownHTMLWithRewrite(t *testing.T) {
	var seen []string
	got := RenderMarkdownHTMLWith("[a](/event/x) and [b](https://other.example/y)", "https://example.com", func(dest string) string {
		seen = append(seen, dest)
		return dest + "?rewritten=1"
	})
	if len(seen) != 2 || seen[0] != "https://example.com/event/x" || seen[1] != "https://other.example/y" {
		t.Errorf("rewrite should see absolute destinations, saw %v", seen)
	}
	if !strings.Contains(got, `href="https://example.com/event/x?rewritten=1"`) {
		t.Errorf("rewritten destination not rendered: %s", got)
	}
	if plain := RenderMarkdownHTML("[a](/event/x)", "https://example.com"); !strings.Contains(plain, `href="https://example.com/event/x"`) || strings.Contains(plain, "rewritten") {
		t.Errorf("RenderMarkdownHTML must stay a plain wrapper: %s", plain)
	}
}

func TestLongDateTimeExported(t *testing.T) {
	// 19:30 UTC on 10 Oct 2026 is 20:30 in London (BST).
	got := LongDateTime(time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC))
	if got != "Saturday 10 October 2026, 8:30 PM" {
		t.Errorf("LongDateTime = %q", got)
	}
}
