package seo

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// HomeEvent is a minimal event for the bot-facing home page listing
type HomeEvent struct {
	Slug      string
	Title     string
	StartDate string
}

// HomeBrowse holds the crawlable navigation shown under the event list: every
// venue and category that has a live page, and the latest picks posts.
type HomeBrowse struct {
	Places, Tags, Picks []Link
}

// homeHeading returns the page heading and <title>. For an instance named
// "<Region> Events" it leads with "What's on in <Region>"; otherwise it keeps
// the instance name.
func homeHeading(instanceName string) (title, h1 string) {
	region := regionFromInstance(instanceName)
	if region != strings.TrimSpace(instanceName) && region != "" {
		h1 = "What's on in " + region
		return h1 + " - " + instanceName, h1
	}
	return instanceName, instanceName
}

// GenerateHomeHTML fetches upcoming events and returns server-rendered HTML for bots
func GenerateHomeHTML(app core.App, baseURL string) ([]byte, error) {
	instanceName := "Gather"
	description := ""
	ogImage := ""
	if settings, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
		description = settings.GetString("instance_description")
		if favicon := settings.GetString("favicon"); favicon != "" {
			ogImage = fmt.Sprintf("%s/api/files/%s/%s/%s", baseURL, settings.Collection().Id, settings.Id, favicon)
		}
	}

	today := time.Now().Format("2006-01-02")
	events, _ := app.FindRecordsByFilter(
		"events",
		"status = 'published' && start_datetime >= {:today}",
		"start_datetime",
		50,
		0,
		map[string]any{"today": today},
	)

	homeEvents := make([]HomeEvent, 0, len(events))
	for _, e := range events {
		startTime := e.GetDateTime("start_datetime").Time()
		slug := e.GetString("slug")
		if slug == "" {
			slug = e.Id
		}
		homeEvents = append(homeEvents, HomeEvent{
			Slug:      slug,
			Title:     e.GetString("title"),
			StartDate: startTime.Format("Monday, 2 January 2006"),
		})
	}

	now := time.Now()
	var browse HomeBrowse
	for _, p := range ActivePlaces(app, now) {
		browse.Places = append(browse.Places, Link{Text: p.GetString("name"), URL: baseURL + "/place/" + PlaceRef(p)})
	}
	for _, t := range ActiveTags(app, now) {
		browse.Tags = append(browse.Tags, Link{Text: humanizeTag(t.GetString("name")), URL: baseURL + "/tag/" + t.GetString("name")})
	}
	picks, _ := app.FindRecordsByFilter("picks", "hidden = false", "-start_date", 30, 0)
	for _, pk := range picks {
		if len(browse.Picks) >= 4 {
			break
		}
		if PickLifecycle(app, pk, now) != Retired {
			browse.Picks = append(browse.Picks, Link{Text: pk.GetString("title"), URL: baseURL + "/picks/" + pk.GetString("slug")})
		}
	}

	return []byte(buildHomeHTML(instanceName, description, ogImage, baseURL, homeEvents, browse)), nil
}

func buildHomeHTML(instanceName, description, ogImage, baseURL string, events []HomeEvent, browse HomeBrowse) string {
	var b strings.Builder
	title, h1 := homeHeading(instanceName)

	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString(`<html lang="en">` + "\n<head>\n")
	b.WriteString(`  <meta charset="UTF-8">` + "\n")
	b.WriteString(`  <meta name="viewport" content="width=device-width, initial-scale=1.0">` + "\n")
	b.WriteString(fmt.Sprintf("  <title>%s</title>\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf(`  <link rel="canonical" href="%s/">`, htmlEscape(baseURL)) + "\n")

	if description != "" {
		b.WriteString(fmt.Sprintf(`  <meta name="description" content="%s">`, htmlEscape(truncateText(description, 160))) + "\n")
	}

	b.WriteString(`  <meta property="og:type" content="website">` + "\n")
	b.WriteString(fmt.Sprintf(`  <meta property="og:site_name" content="%s">`, htmlEscape(instanceName)) + "\n")
	b.WriteString(fmt.Sprintf(`  <meta property="og:title" content="%s">`, htmlEscape(title)) + "\n")
	b.WriteString(fmt.Sprintf(`  <meta property="og:url" content="%s/">`, htmlEscape(baseURL)) + "\n")
	if description != "" {
		b.WriteString(fmt.Sprintf(`  <meta property="og:description" content="%s">`, htmlEscape(truncateText(description, 200))) + "\n")
	}
	if ogImage != "" {
		b.WriteString(fmt.Sprintf(`  <meta property="og:image" content="%s">`, htmlEscape(ogImage)) + "\n")
	}
	b.WriteString(fmt.Sprintf(`  <link rel="alternate" type="application/rss+xml" title="%s" href="%s/feed.rss">`, htmlEscape(instanceName), baseURL) + "\n")
	if websiteLD, err := json.Marshal(map[string]string{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     instanceName,
		"url":      baseURL + "/",
	}); err == nil {
		b.WriteString("  <script type=\"application/ld+json\">\n  " + string(websiteLD) + "\n  </script>\n")
	}
	b.WriteString("</head>\n<body>\n")
	b.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", htmlEscape(h1)))
	if description != "" {
		b.WriteString(fmt.Sprintf("  <p>%s</p>\n", htmlEscape(description)))
	}

	if len(events) > 0 {
		b.WriteString("  <h2>Upcoming Events</h2>\n  <ul>\n")
		for _, e := range events {
			b.WriteString(fmt.Sprintf(
				"    <li><a href=\"%s/event/%s\">%s &mdash; %s</a></li>\n",
				baseURL, htmlEscape(e.Slug), htmlEscape(e.Title), htmlEscape(e.StartDate),
			))
		}
		b.WriteString("  </ul>\n")
	} else {
		b.WriteString("  <p>No upcoming events.</p>\n")
	}

	writeBrowse := func(heading string, links []Link) {
		if len(links) == 0 {
			return
		}
		b.WriteString(fmt.Sprintf("  <h2>%s</h2>\n  <ul>\n", htmlEscape(heading)))
		for _, l := range links {
			b.WriteString(fmt.Sprintf("    <li><a href=\"%s\">%s</a></li>\n", htmlEscape(l.URL), htmlEscape(l.Text)))
		}
		b.WriteString("  </ul>\n")
	}
	writeBrowse("Browse by venue", browse.Places)
	writeBrowse("Browse by category", browse.Tags)
	writeBrowse("Weekend picks", browse.Picks)

	b.WriteString("</body>\n</html>\n")
	return b.String()
}
