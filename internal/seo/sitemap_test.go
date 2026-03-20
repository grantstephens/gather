package seo

import (
	"strings"
	"testing"
)

func TestBuildSitemapXML(t *testing.T) {
	baseURL := "https://example.com"
	eventURLs := []string{
		"https://example.com/event/abc123",
		"https://example.com/event/def456",
	}

	xml := buildSitemapXML(baseURL, eventURLs)

	if !strings.Contains(xml, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("sitemap should start with XML declaration")
	}
	if !strings.Contains(xml, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`) {
		t.Error("sitemap should have urlset element with correct namespace")
	}
	if !strings.Contains(xml, "<loc>https://example.com/</loc>") {
		t.Error("sitemap should include home page URL")
	}
	if !strings.Contains(xml, "<loc>https://example.com/event/abc123</loc>") {
		t.Error("sitemap should include first event URL")
	}
	if !strings.Contains(xml, "<loc>https://example.com/event/def456</loc>") {
		t.Error("sitemap should include second event URL")
	}
	if !strings.Contains(xml, "</urlset>") {
		t.Error("sitemap should close urlset")
	}
}

func TestBuildRobotsTxt(t *testing.T) {
	baseURL := "https://example.com"
	result := BuildRobotsTxt(baseURL)

	if !strings.Contains(result, "User-agent: *") {
		t.Error("robots.txt should have User-agent wildcard")
	}
	if !strings.Contains(result, "Allow: /") {
		t.Error("robots.txt should allow all paths")
	}
	if !strings.Contains(result, "Sitemap: https://example.com/sitemap.xml") {
		t.Error("robots.txt should include Sitemap directive with full URL")
	}
}

func TestRobotsTxtDisallowsInternalPaths(t *testing.T) {
	result := BuildRobotsTxt("https://example.com")
	for _, path := range []string{"/api/", "/_/"} {
		if !strings.Contains(result, "Disallow: "+path) {
			t.Errorf("robots.txt should disallow %s", path)
		}
	}
}
