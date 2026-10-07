package seo

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// SitemapEntry holds a URL and its last modified date for sitemap generation
type SitemapEntry struct {
	URL     string
	LastMod string
}

// GenerateSitemap lists every live URL: upcoming events and ones that ended
// within the grace period (no cap), visible custom pages, places and tags that
// have upcoming events, and picks posts that are not retired.
func GenerateSitemap(app core.App, baseURL string) ([]byte, error) {
	now := time.Now()
	cutoff := now.Add(-(RetireAfter + 24*time.Hour)).UTC().Format("2006-01-02 15:04:05")
	// Cheap SQL pre-filter (keeps long-running and recurring events), then the exact rule.
	events, err := AllRecords(app, "events",
		"status = 'published' && (start_datetime >= {:cut} || end_datetime >= {:cut} || recurrence_rule != '')",
		"-start_datetime", dbx.Params{"cut": cutoff})
	if err != nil {
		return nil, err
	}

	var entries []SitemapEntry
	for _, ev := range events {
		if EventLifecycle(ev, now) == Retired {
			continue
		}
		path := ev.GetString("slug")
		if path == "" {
			path = ev.Id
		}
		entries = append(entries, SitemapEntry{URL: fmt.Sprintf("%s/event/%s", baseURL, path), LastMod: lastMod(ev)})
	}

	pages, _ := AllRecords(app, "pages", "", "sort_order,title")
	for _, pg := range pages {
		entries = append(entries, SitemapEntry{URL: fmt.Sprintf("%s/%s", baseURL, pg.GetString("slug")), LastMod: lastMod(pg)})
	}
	for _, tag := range ActiveTags(app, now) {
		entries = append(entries, SitemapEntry{URL: fmt.Sprintf("%s/tag/%s", baseURL, tag.GetString("name")), LastMod: lastMod(tag)})
	}
	for _, place := range ActivePlaces(app, now) {
		entries = append(entries, SitemapEntry{URL: fmt.Sprintf("%s/place/%s", baseURL, PlaceRef(place)), LastMod: lastMod(place)})
	}

	picks, _ := AllRecords(app, "picks", "hidden = false", "-start_date")
	entries = append(entries, SitemapEntry{URL: baseURL + "/picks", LastMod: newestMod(picks, now)})
	for _, post := range picks {
		if PickLifecycle(app, post, now) == Retired {
			continue
		}
		entries = append(entries, SitemapEntry{URL: fmt.Sprintf("%s/picks/%s", baseURL, post.GetString("slug")), LastMod: lastMod(post)})
	}

	return []byte(buildSitemapXML(baseURL, entries)), nil
}

func lastMod(r *core.Record) string {
	if t := r.GetDateTime("updated").Time(); t.Year() >= 2000 {
		return t.Format("2006-01-02")
	}
	return time.Now().Format("2006-01-02")
}

func newestMod(recs []*core.Record, now time.Time) string {
	if len(recs) > 0 {
		return lastMod(recs[0])
	}
	return now.Format("2006-01-02")
}

func buildSitemapXML(baseURL string, entries []SitemapEntry) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")

	today := time.Now().Format("2006-01-02")

	// Google ignores <changefreq> and <priority>; only an accurate <lastmod> is used.
	writeURL := func(loc, lastmod string) {
		b.WriteString("  <url>\n")
		b.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", html.EscapeString(loc)))
		b.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", lastmod))
		b.WriteString("  </url>\n")
	}

	writeURL(baseURL+"/", today)
	for _, e := range entries {
		writeURL(e.URL, e.LastMod)
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
Disallow: /search
Disallow: /login
Disallow: /feed
Disallow: /ics
Disallow: /federation
Disallow: /embed

Sitemap: %s/sitemap.xml
Agentmap: %s/.well-known/ard.json
`, baseURL, baseURL)
}
