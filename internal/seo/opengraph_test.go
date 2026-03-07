package seo

import (
	"strings"
	"testing"
)

func TestHtmlEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: `<script>alert("xss")</script>`,
			want:  `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`,
		},
		{
			input: `Normal text`,
			want:  `Normal text`,
		},
		{
			input: `Text with "quotes" & ampersands`,
			want:  `Text with &#34;quotes&#34; &amp; ampersands`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := htmlEscape(tt.input); got != tt.want {
				t.Errorf("htmlEscape() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{
			name:   "Short text",
			text:   "Short text",
			maxLen: 100,
			want:   "Short text",
		},
		{
			name:   "Exact length",
			text:   "Exactly 20 chars now",
			maxLen: 20,
			want:   "Exactly 20 chars now",
		},
		{
			name:   "Truncate at word boundary",
			text:   "This is a very long description that needs to be truncated",
			maxLen: 30,
			want:   "This is a very long...",
		},
		{
			name:   "Very short maxLen",
			text:   "Some text here",
			maxLen: 5,
			want:   "Some...",
		},
		{
			name:   "Whitespace trimming",
			text:   "  Text with spaces  ",
			maxLen: 100,
			want:   "Text with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateText(tt.text, tt.maxLen); got != tt.want {
				t.Errorf("truncateText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateMetaTagsStructure(t *testing.T) {
	// This is a unit test that doesn't require PocketBase
	// We'll test the structure and escaping logic separately

	// Test that htmlEscape is applied correctly
	escaped := htmlEscape(`<script>alert("xss")</script>`)
	if strings.Contains(escaped, "<script>") {
		t.Error("htmlEscape failed to escape script tags")
	}

	// Test that truncateText works
	long := "This is a very long description that should definitely be truncated to exactly 50 characters maximum to ensure the truncation logic works properly and adds ellipsis when needed"
	truncated := truncateText(long, 50)
	if len(truncated) > 53 { // 50 + "..."
		t.Errorf("truncateText exceeded max length: got %d chars", len(truncated))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Error("truncateText should add ellipsis when text exceeds maxLen")
	}
}
