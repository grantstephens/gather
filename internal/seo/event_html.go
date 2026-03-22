package seo

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// GenerateEventHTML creates a complete HTML page with Schema.org JSON-LD and meta tags for an event
func GenerateEventHTML(app core.App, event *core.Record, baseURL string) ([]byte, error) {
	instanceName := "Gather"
	settings, err := app.FindFirstRecordByFilter("settings", "")
	if err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	jsonLD, err := GenerateEventJSONLD(app, event, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JSON-LD: %w", err)
	}

	metaTags := GenerateMetaTags(app, event, baseURL)
	eventURL := eventPageURL(baseURL, event)
	description := truncateText(stripMarkdown(event.GetString("description")), 300)

	return []byte(buildEventHTML(
		event.GetString("title"),
		instanceName,
		description,
		eventURL,
		metaTags,
		string(jsonLD),
	)), nil
}

func buildEventHTML(title, instanceName, description, eventURL, metaTags, jsonLD string) string {
	var html strings.Builder
	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString(`<html lang="en">` + "\n<head>\n")
	html.WriteString(`  <meta charset="UTF-8">` + "\n")
	html.WriteString(`  <meta name="viewport" content="width=device-width, initial-scale=1.0">` + "\n")
	html.WriteString(fmt.Sprintf(`  <title>%s - %s</title>`, htmlEscape(title), htmlEscape(instanceName)) + "\n")
	html.WriteString(fmt.Sprintf(`  <link rel="canonical" href="%s">`, htmlEscape(eventURL)) + "\n")
	if description != "" {
		html.WriteString(fmt.Sprintf(`  <meta name="description" content="%s">`, htmlEscape(truncateText(description, 160))) + "\n")
	}
	if strings.TrimSpace(metaTags) != "" {
		html.WriteString("  ")
		html.WriteString(strings.ReplaceAll(strings.TrimSpace(metaTags), "\n", "\n  "))
		html.WriteString("\n")
	}
	html.WriteString(`  <script type="application/ld+json">` + "\n")
	html.WriteString(jsonLD + "\n")
	html.WriteString(`  </script>` + "\n")
	html.WriteString("</head>\n<body>\n")
	html.WriteString(fmt.Sprintf(`  <h1>%s</h1>`, htmlEscape(title)) + "\n")
	if description != "" {
		html.WriteString(fmt.Sprintf(`  <p>%s</p>`, htmlEscape(description)) + "\n")
	}
	html.WriteString(fmt.Sprintf(`  <p><a href="%s">View full event details</a></p>`, htmlEscape(eventURL)) + "\n")
	html.WriteString("</body>\n</html>\n")
	return html.String()
}
