package seo

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildLLMsTxtStartsWithSingleH1(t *testing.T) {
	result := BuildLLMsTxt("https://example.com", "Example Events", "A test instance.")
	lines := strings.Split(strings.TrimLeft(result, "\n"), "\n")
	if !strings.HasPrefix(lines[0], "# ") {
		t.Errorf("llms.txt must start with a single H1, got first line: %q", lines[0])
	}
	h1Count := strings.Count(result, "\n# ") + 1
	if h1Count != 1 {
		t.Errorf("llms.txt must contain exactly one H1, found %d", h1Count)
	}
}

func TestBuildLLMsTxtUsesAbsoluteURLs(t *testing.T) {
	result := BuildLLMsTxt("https://example.com", "Example Events", "A test instance.")
	for _, link := range []string{
		"](https://example.com/)",
		"](https://example.com/api/search)",
		"](https://example.com/feed.rss)",
		"](https://example.com/feed.ics)",
	} {
		if !strings.Contains(result, link) {
			t.Errorf("llms.txt should contain absolute link %q, got:\n%s", link, result)
		}
	}
	// Relative links break parsing per the llms.txt spec checklist.
	if strings.Contains(result, "](/") {
		t.Errorf("llms.txt should not contain relative links, got:\n%s", result)
	}
}

func TestBuildLLMsTxtFallsBackWhenDescriptionEmpty(t *testing.T) {
	result := BuildLLMsTxt("https://example.com", "Example Events", "")
	if strings.Contains(result, "\n> \n") {
		t.Error("llms.txt should never emit an empty blockquote line")
	}
	if !strings.Contains(result, "> A community events calendar.") {
		t.Errorf("expected fallback description, got:\n%s", result)
	}
}

func TestBuildAICatalogRootHasOnlySpecFields(t *testing.T) {
	data, err := BuildAICatalogJSON("https://example.com", "Example Events")
	if err != nil {
		t.Fatalf("BuildAICatalogJSON error: %v", err)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// The predecessor schema Lighthouse currently validates against is
	// additionalProperties:false at root — only specVersion/host/entries
	// are recognized (see GoogleChrome/lighthouse#17251).
	allowed := map[string]bool{"specVersion": true, "host": true, "entries": true}
	for key := range root {
		if !allowed[key] {
			t.Errorf("ai-catalog.json root has unrecognized field %q, which fails the predecessor schema's additionalProperties:false", key)
		}
	}
	if _, ok := root["specVersion"]; !ok {
		t.Error("ai-catalog.json must have specVersion at root")
	}
	if _, ok := root["entries"]; !ok {
		t.Error("ai-catalog.json must have entries at root")
	}
}

func TestBuildAICatalogEntriesHaveRequiredFields(t *testing.T) {
	catalog := BuildAICatalog("https://example.com", "Example Events")
	if len(catalog.Entries) == 0 {
		t.Fatal("expected at least one catalog entry")
	}
	for _, e := range catalog.Entries {
		if e.Identifier == "" || !strings.HasPrefix(e.Identifier, "urn:air:") {
			t.Errorf("entry identifier must be a urn:air: URN, got %q", e.Identifier)
		}
		if e.DisplayName == "" {
			t.Errorf("entry %q missing displayName", e.Identifier)
		}
		if e.Type == "" {
			t.Errorf("entry %q missing type", e.Identifier)
		}
		if e.URL == "" {
			t.Errorf("entry %q missing url", e.Identifier)
		}
		if !strings.HasPrefix(e.URL, "https://example.com") {
			t.Errorf("entry %q url should be absolute under the base URL, got %q", e.Identifier, e.URL)
		}
	}
}

func TestBuildAICatalogHostIdentifierIsDIDWeb(t *testing.T) {
	catalog := BuildAICatalog("https://example.com", "Example Events")
	if catalog.Host.Identifier != "did:web:example.com" {
		t.Errorf("expected did:web:example.com, got %q", catalog.Host.Identifier)
	}
}
