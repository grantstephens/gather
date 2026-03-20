package seo

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

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

	urls := make([]string, 0, len(events))
	for _, event := range events {
		urls = append(urls, fmt.Sprintf("%s/event/%s", baseURL, event.Id))
	}

	return []byte(buildSitemapXML(baseURL, urls)), nil
}

func buildSitemapXML(baseURL string, eventURLs []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")

	today := time.Now().Format("2006-01-02")

	writeURL := func(loc, changefreq, priority string) {
		b.WriteString("  <url>\n")
		b.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", html.EscapeString(loc)))
		b.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", today))
		b.WriteString(fmt.Sprintf("    <changefreq>%s</changefreq>\n", changefreq))
		b.WriteString(fmt.Sprintf("    <priority>%s</priority>\n", priority))
		b.WriteString("  </url>\n")
	}

	writeURL(baseURL+"/", "daily", "1.0")
	for _, u := range eventURLs {
		writeURL(u, "weekly", "0.8")
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
