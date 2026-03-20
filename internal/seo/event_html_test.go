package seo

import (
	"strings"
	"testing"
)

func TestBuildEventHTMLContainsCanonical(t *testing.T) {
	html := buildEventHTML("My Event", "Gather", "A description", "https://example.com/event/abc", "<meta>", "{}")
	if !strings.Contains(html, `<link rel="canonical" href="https://example.com/event/abc">`) {
		t.Errorf("event HTML should contain canonical tag, got:\n%s", html)
	}
}

func TestBuildEventHTMLNoMetaRefresh(t *testing.T) {
	html := buildEventHTML("My Event", "Gather", "", "https://example.com/event/abc", "", "{}")
	if strings.Contains(html, `http-equiv="refresh"`) {
		t.Error("event HTML should not contain meta-refresh redirect")
	}
	if strings.Contains(html, `window.location.href`) {
		t.Error("event HTML should not contain JS redirect")
	}
}

func TestBuildEventHTMLContainsTitle(t *testing.T) {
	html := buildEventHTML("Summer Fest", "Perthshire Events", "Great fun", "https://example.com/event/abc", "", "{}")
	if !strings.Contains(html, "<title>Summer Fest - Perthshire Events</title>") {
		t.Errorf("event HTML title should be 'Event - Site', got:\n%s", html)
	}
}

func TestBuildEventHTMLContainsDescription(t *testing.T) {
	html := buildEventHTML("Event", "Gather", "Some description text", "https://example.com/event/abc", "", "{}")
	if !strings.Contains(html, "Some description text") {
		t.Error("event HTML body should contain description text")
	}
}

func TestBuildEventHTMLXSSEscape(t *testing.T) {
	html := buildEventHTML(`<script>alert("xss")</script>`, "Gather", "", "https://example.com/event/abc", "", "{}")
	if strings.Contains(html, "<script>alert") {
		t.Error("event HTML should escape XSS in title")
	}
}

func TestGenerateMetaTagsContainsSiteName(t *testing.T) {
	tags := buildMetaTags("Test Event", "A description", "https://example.com/event/1", "", "My Site")
	if !strings.Contains(tags, `<meta property="og:site_name" content="My Site">`) {
		t.Errorf("meta tags should contain og:site_name, got:\n%s", tags)
	}
	if !strings.Contains(tags, `<meta property="og:title" content="Test Event">`) {
		t.Errorf("meta tags should contain og:title, got:\n%s", tags)
	}
}
