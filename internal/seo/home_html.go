package seo

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// HomeEvent is a minimal event for the bot-facing home page listing
type HomeEvent struct {
	ID        string
	Title     string
	StartDate string
}

// GenerateHomeHTML fetches upcoming events and returns server-rendered HTML for bots
func GenerateHomeHTML(app core.App, baseURL string) ([]byte, error) {
	instanceName := "Gather"
	description := ""
	if settings, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
		description = settings.GetString("instance_description")
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
		homeEvents = append(homeEvents, HomeEvent{
			ID:        e.Id,
			Title:     e.GetString("title"),
			StartDate: startTime.Format("Monday, 2 January 2006"),
		})
	}

	return []byte(buildHomeHTML(instanceName, description, baseURL, homeEvents)), nil
}

func buildHomeHTML(instanceName, description, baseURL string, events []HomeEvent) string {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString(`<html lang="en">` + "\n<head>\n")
	b.WriteString(`  <meta charset="UTF-8">` + "\n")
	b.WriteString(`  <meta name="viewport" content="width=device-width, initial-scale=1.0">` + "\n")
	b.WriteString(fmt.Sprintf("  <title>%s</title>\n", htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf(`  <link rel="canonical" href="%s/">`, htmlEscape(baseURL)) + "\n")

	if description != "" {
		b.WriteString(fmt.Sprintf(`  <meta name="description" content="%s">`, htmlEscape(truncateText(description, 160))) + "\n")
	}

	b.WriteString(`  <meta property="og:type" content="website">` + "\n")
	b.WriteString(fmt.Sprintf(`  <meta property="og:site_name" content="%s">`, htmlEscape(instanceName)) + "\n")
	b.WriteString(fmt.Sprintf(`  <meta property="og:title" content="%s">`, htmlEscape(instanceName)) + "\n")
	b.WriteString(fmt.Sprintf(`  <meta property="og:url" content="%s/">`, htmlEscape(baseURL)) + "\n")
	if description != "" {
		b.WriteString(fmt.Sprintf(`  <meta property="og:description" content="%s">`, htmlEscape(truncateText(description, 200))) + "\n")
	}
	b.WriteString(fmt.Sprintf(`  <link rel="alternate" type="application/rss+xml" title="%s" href="%s/feed.rss">`, htmlEscape(instanceName), baseURL) + "\n")
	b.WriteString("</head>\n<body>\n")
	b.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", htmlEscape(instanceName)))
	if description != "" {
		b.WriteString(fmt.Sprintf("  <p>%s</p>\n", htmlEscape(description)))
	}

	if len(events) > 0 {
		b.WriteString("  <h2>Upcoming Events</h2>\n  <ul>\n")
		for _, e := range events {
			b.WriteString(fmt.Sprintf(
				"    <li><a href=\"%s/event/%s\">%s &mdash; %s</a></li>\n",
				baseURL, htmlEscape(e.ID), htmlEscape(e.Title), htmlEscape(e.StartDate),
			))
		}
		b.WriteString("  </ul>\n")
	} else {
		b.WriteString("  <p>No upcoming events.</p>\n")
	}

	b.WriteString("</body>\n</html>\n")
	return b.String()
}
