package seo

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// SitemapEntry holds a URL and its last modified date for sitemap generation
type SitemapEntry struct {
	URL     string
	LastMod string
}

// GenerateSitemap fetches all published events, approved tags/places, and
// custom pages, then returns sitemap XML bytes.
func GenerateSitemap(app core.App, baseURL string) ([]byte, error) {
	events, err := app.FindRecordsByFilter(
		"events",
		"status = 'published'",
		"-start_datetime",
		500,
		0,
	)
	if err != nil {
		return nil, err
	}

	entries := make([]SitemapEntry, 0, len(events)+50)
	for _, event := range events {
		updatedTime := event.GetDateTime("updated").Time()
		var lastMod string
		if updatedTime.IsZero() {
			lastMod = time.Now().Format("2006-01-02")
		} else {
			lastMod = updatedTime.Format("2006-01-02")
		}
		urlPath := event.GetString("slug")
		if urlPath == "" {
			urlPath = event.Id
		}
		entries = append(entries, SitemapEntry{
			URL:     fmt.Sprintf("%s/event/%s", baseURL, urlPath),
			LastMod: lastMod,
		})
	}

	// Custom pages
	pages, _ := app.FindRecordsByFilter("pages", "", "sort_order,title", 100, 0)
	for _, page := range pages {
		updatedTime := page.GetDateTime("updated").Time()
		lastMod := time.Now().Format("2006-01-02")
		if !updatedTime.IsZero() {
			lastMod = updatedTime.Format("2006-01-02")
		}
		entries = append(entries, SitemapEntry{
			URL:     fmt.Sprintf("%s/%s", baseURL, page.GetString("slug")),
			LastMod: lastMod,
		})
	}

	// Approved tags
	tags, _ := app.FindRecordsByFilter("tags", "status = 'approved'", "name", 200, 0)
	for _, tag := range tags {
		entries = append(entries, SitemapEntry{
			URL:     fmt.Sprintf("%s/tag/%s", baseURL, tag.GetString("name")),
			LastMod: time.Now().Format("2006-01-02"),
		})
	}

	// Approved places
	places, _ := app.FindRecordsByFilter("places", "status = 'approved'", "name", 200, 0)
	for _, place := range places {
		entries = append(entries, SitemapEntry{
			URL:     fmt.Sprintf("%s/place/%s", baseURL, place.Id),
			LastMod: time.Now().Format("2006-01-02"),
		})
	}

	return []byte(buildSitemapXML(baseURL, entries)), nil
}

func buildSitemapXML(baseURL string, entries []SitemapEntry) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")

	today := time.Now().Format("2006-01-02")

	writeURL := func(loc, lastmod, changefreq, priority string) {
		b.WriteString("  <url>\n")
		b.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", html.EscapeString(loc)))
		b.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", lastmod))
		b.WriteString(fmt.Sprintf("    <changefreq>%s</changefreq>\n", changefreq))
		b.WriteString(fmt.Sprintf("    <priority>%s</priority>\n", priority))
		b.WriteString("  </url>\n")
	}

	writeURL(baseURL+"/", today, "daily", "1.0")
	for _, e := range entries {
		writeURL(e.URL, e.LastMod, "weekly", "0.8")
	}

	b.WriteString("</urlset>\n")
	return b.String()
}

// BuildRobotsTxt returns robots.txt content with sitemap pointer
func BuildRobotsTxt(baseURL string) string {
	return fmt.Sprintf(`User-agent: *
Allow: /
Disallow: /api/
Disallow: /_/

Sitemap: %s/sitemap.xml
`, baseURL)
}
