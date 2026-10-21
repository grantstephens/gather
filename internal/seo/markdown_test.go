package seo

import (
	"strings"
	"testing"
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
