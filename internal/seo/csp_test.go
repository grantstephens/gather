package seo

import (
	"strings"
	"testing"
)

func TestExtractExternalOrigins(t *testing.T) {
	html := `<script defer src="https://plausible.io/js/script.js"></script>
<script src="https://cdn.example.com/tracker.js"></script>
<script src="/local/script.js"></script>`

	origins := ExtractExternalOrigins(html)
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d: %v", len(origins), origins)
	}
	if origins[0] != "https://plausible.io" {
		t.Errorf("expected https://plausible.io, got %s", origins[0])
	}
	if origins[1] != "https://cdn.example.com" {
		t.Errorf("expected https://cdn.example.com, got %s", origins[1])
	}
}

func TestExtractExternalOriginsDeduplicates(t *testing.T) {
	html := `<script src="https://plausible.io/js/script.js"></script>
<script src="https://plausible.io/js/other.js"></script>`

	origins := ExtractExternalOrigins(html)
	if len(origins) != 1 {
		t.Errorf("expected 1 origin (deduplicated), got %d: %v", len(origins), origins)
	}
}

func TestBuildCSPNoExtras(t *testing.T) {
	csp := BuildCSP(nil)
	if !strings.Contains(csp, "script-src 'self'") {
		t.Error("CSP should contain script-src 'self'")
	}
	if !strings.Contains(csp, "connect-src 'self'") {
		t.Error("CSP should contain connect-src 'self'")
	}
}

func TestBuildCSPWithExtras(t *testing.T) {
	csp := BuildCSP([]string{"https://plausible.io"})
	if !strings.Contains(csp, "https://plausible.io") {
		t.Errorf("CSP should contain extra origin, got: %s", csp)
	}
	if !strings.Contains(csp, "script-src") || !strings.Contains(csp, "connect-src") {
		t.Errorf("CSP should contain script-src and connect-src, got: %s", csp)
	}
}
