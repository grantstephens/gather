package seo

import (
	"strings"
	"testing"
)

func TestBuildHomeHTML(t *testing.T) {
	result := buildHomeHTML("Perthshire Events", "Community calendar for Perthshire", "https://example.com/logo.webp", "https://example.com", []HomeEvent{
		{Slug: "abc123", Title: "Summer Fest", StartDate: "Saturday, 1 June 2026"},
		{Slug: "def456", Title: "Folk Night", StartDate: "Sunday, 2 June 2026"},
	})

	if !strings.Contains(result, "<title>Perthshire Events</title>") {
		t.Error("home HTML should contain instance name as title")
	}
	if !strings.Contains(result, `<meta name="description"`) {
		t.Error("home HTML should contain meta description tag")
	}
	if !strings.Contains(result, "Community calendar for Perthshire") {
		t.Error("home HTML should contain instance description")
	}
	if !strings.Contains(result, `href="https://example.com/event/abc123"`) {
		t.Error("home HTML should link to events")
	}
	if !strings.Contains(result, "Summer Fest") {
		t.Error("home HTML should contain event titles")
	}
	if !strings.Contains(result, `og:title`) {
		t.Error("home HTML should contain og:title")
	}
	if !strings.Contains(result, `og:site_name`) {
		t.Error("home HTML should contain og:site_name")
	}
	if !strings.Contains(result, `rel="canonical"`) {
		t.Error("home HTML should contain canonical tag")
	}
	if !strings.Contains(result, `type="application/rss+xml"`) {
		t.Error("home HTML should contain RSS link")
	}
}

func TestBuildHomeHTMLNoEvents(t *testing.T) {
	result := buildHomeHTML("Gather", "", "", "https://example.com", nil)
	if !strings.Contains(result, "<title>Gather</title>") {
		t.Error("home HTML should still render title with no events")
	}
	if !strings.Contains(result, "No upcoming events") {
		t.Error("home HTML should show no-events message")
	}
}

func TestBuildHomeHTMLXSSEscape(t *testing.T) {
	result := buildHomeHTML(`<script>alert("xss")</script>`, "", "", "https://example.com", nil)
	if strings.Contains(result, "<script>alert") {
		t.Error("home HTML should escape XSS in instance name")
	}
}
