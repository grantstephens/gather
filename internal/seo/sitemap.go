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

// GenerateSitemap fetches all published events and returns sitemap XML bytes
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

	entries := make([]SitemapEntry, 0, len(events))
	for _, event := range events {
		lastMod := event.GetDateTime("updated").Time().Format("2006-01-02")
		if lastMod == "" {
			lastMod = time.Now().Format("2006-01-02")
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
