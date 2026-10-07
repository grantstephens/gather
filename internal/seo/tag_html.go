package seo

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type tagEvent struct {
	Title     string
	URL       string
	StartDate string
}

// GenerateTagHTML creates a complete HTML page for a tag listing page.
func GenerateTagHTML(app core.App, tagName, baseURL string) ([]byte, error) {
	instanceName := "Gather"
	if settings, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	tag, err := app.FindFirstRecordByFilter(
		"tags",
		"name = {:name} && status = 'approved'",
		map[string]any{"name": tagName},
	)
	if err != nil {
		return nil, err
	}

	tagURL := fmt.Sprintf("%s/tag/%s", baseURL, tagName)

	today := time.Now().Format("2006-01-02 15:04:05")
	events, _ := app.FindRecordsByFilter(
		"events",
		"status = 'published' && start_datetime >= {:today} && tags ?~ {:tagId}",
		"start_datetime",
		50,
		0,
		map[string]any{"today": today, "tagId": tag.Id},
	)

	if len(events) == 0 {
		return nil, fmt.Errorf("no upcoming events for tag")
	}
	title := tagPageTitle(tagName, instanceName)
	description := tagIntro(tagName, len(events))

	items := make([]tagEvent, 0, len(events))
	for _, ev := range events {
		slug := ev.GetString("slug")
		if slug == "" {
			slug = ev.Id
		}
		dateStr := ""
		if t := ev.GetDateTime("start_datetime").Time(); !t.IsZero() {
			dateStr = t.Local().Format("Monday 2 January, 3:04 PM MST")
		}
		items = append(items, tagEvent{
			Title:     ev.GetString("title"),
			URL:       fmt.Sprintf("%s/event/%s", baseURL, slug),
			StartDate: dateStr,
		})
	}

	return []byte(buildTagHTML(title, instanceName, description, tagURL, baseURL, tagName, items)), nil
}

// tagPageTitle: "Strathearn Arts events in Perthshire - Perthshire Events".
func tagPageTitle(tagName, instanceName string) string {
	heading := humanizeTag(tagName) + " events"
	if region := regionFromInstance(instanceName); region != strings.TrimSpace(instanceName) && region != "" {
		heading += " in " + region
	}
	return heading + " - " + instanceName
}

// tagIntro is the meta description and opening paragraph of a tag page.
func tagIntro(tagName string, n int) string {
	noun := "events"
	if n == 1 {
		noun = "event"
	}
	return fmt.Sprintf("%d upcoming %s %s — dates, venues and booking details.", n, strings.ToLower(humanizeTag(tagName)), noun)
}

func buildTagHTML(title, instanceName, description, tagURL, baseURL, tagName string, items []tagEvent) string {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("<html lang=\"en\">\n<head>\n")
	b.WriteString("  <meta charset=\"UTF-8\">\n")
	b.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	b.WriteString(fmt.Sprintf("  <title>%s</title>\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <link rel=\"canonical\" href=\"%s\">\n", htmlEscape(tagURL)))
	b.WriteString(fmt.Sprintf("  <meta name=\"description\" content=\"%s\">\n", htmlEscape(description)))

	b.WriteString("  <meta property=\"og:type\" content=\"website\">\n")
	b.WriteString(fmt.Sprintf("  <meta property=\"og:site_name\" content=\"%s\">\n", htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:title\" content=\"%s\">\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:url\" content=\"%s\">\n", htmlEscape(tagURL)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:description\" content=\"%s\">\n", htmlEscape(description)))
	b.WriteString("  <meta name=\"twitter:card\" content=\"summary\">\n")
	b.WriteString(fmt.Sprintf("  <meta name=\"twitter:title\" content=\"%s\">\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <meta name=\"twitter:description\" content=\"%s\">\n", htmlEscape(description)))

	type breadcrumbItem struct {
		Type     string `json:"@type"`
		Position int    `json:"position"`
		Name     string `json:"name"`
		Item     string `json:"item"`
	}
	type breadcrumbList struct {
		Context         string           `json:"@context"`
		Type            string           `json:"@type"`
		ItemListElement []breadcrumbItem `json:"itemListElement"`
	}
	ld := breadcrumbList{
		Context: "https://schema.org",
		Type:    "BreadcrumbList",
		ItemListElement: []breadcrumbItem{
			{Type: "ListItem", Position: 1, Name: instanceName, Item: baseURL},
			{Type: "ListItem", Position: 2, Name: humanizeTag(tagName) + " events", Item: tagURL},
		},
	}
	if ldJSON, err := json.Marshal(ld); err == nil {
		b.WriteString("  <script type=\"application/ld+json\">\n  ")
		b.WriteString(string(ldJSON))
		b.WriteString("\n  </script>\n")
	}

	links := make([]Link, 0, len(items))
	for _, it := range items {
		links = append(links, Link{Text: it.Title, URL: it.URL})
	}
	if list := itemListJSONLD(links); list != "" {
		b.WriteString("  <script type=\"application/ld+json\">\n  " + list + "\n  </script>\n")
	}

	b.WriteString("</head>\n<body>\n")
	b.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", htmlEscape(humanizeTag(tagName))))
	b.WriteString(fmt.Sprintf("  <p>%s</p>\n", htmlEscape(description)))

	if len(items) > 0 {
		b.WriteString("  <ul>\n")
		for _, ev := range items {
			if ev.StartDate != "" {
				b.WriteString(fmt.Sprintf("    <li><a href=\"%s\">%s</a> &mdash; %s</li>\n",
					htmlEscape(ev.URL), htmlEscape(ev.Title), htmlEscape(ev.StartDate)))
			} else {
				b.WriteString(fmt.Sprintf("    <li><a href=\"%s\">%s</a></li>\n",
					htmlEscape(ev.URL), htmlEscape(ev.Title)))
			}
		}
		b.WriteString("  </ul>\n")
	} else {
		b.WriteString("  <p>No upcoming events with this tag.</p>\n")
	}

	b.WriteString("</body>\n</html>\n")
	return b.String()
}
