package seo

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// GenerateEventHTML creates a complete HTML page with Schema.org JSON-LD and meta tags for an event
func GenerateEventHTML(app core.App, event *core.Record, baseURL string) ([]byte, error) {
	// Get instance name from settings
	instanceName := "Gather"
	settings, err := app.FindFirstRecordByFilter("settings", "")
	if err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	// Generate JSON-LD
	jsonLD, err := GenerateEventJSONLD(app, event, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JSON-LD: %w", err)
	}

	// Generate meta tags
	metaTags := GenerateMetaTags(app, event, baseURL)

	// Build HTML
	var html strings.Builder
	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString(`<html lang="en">`)
	html.WriteString("\n<head>\n")
	html.WriteString(`  <meta charset="UTF-8">`)
	html.WriteString("\n")
	html.WriteString(`  <meta name="viewport" content="width=device-width, initial-scale=1.0">`)
	html.WriteString("\n")

	// Title
	title := fmt.Sprintf("%s - %s", htmlEscape(event.GetString("title")), htmlEscape(instanceName))
	html.WriteString(fmt.Sprintf(`  <title>%s</title>`, title))
	html.WriteString("\n")

	// Meta tags
	html.WriteString("  ")
	html.WriteString(strings.ReplaceAll(strings.TrimSpace(metaTags), "\n", "\n  "))
	html.WriteString("\n")

	// JSON-LD script
	html.WriteString(`  <script type="application/ld+json">`)
	html.WriteString("\n")
	html.WriteString(string(jsonLD))
	html.WriteString("\n")
	html.WriteString(`  </script>`)
	html.WriteString("\n")

	// Client-side redirect (meta refresh)
	eventURL := fmt.Sprintf("/event/%s", event.Id)
	html.WriteString(fmt.Sprintf(`  <meta http-equiv="refresh" content="0;url=%s">`, eventURL))
	html.WriteString("\n")

	html.WriteString("</head>\n")
	html.WriteString("<body>\n")
	html.WriteString(`  <p>Redirecting to event page...</p>`)
	html.WriteString("\n")
	html.WriteString(fmt.Sprintf(`  <script>window.location.href = "%s";</script>`, eventURL))
	html.WriteString("\n")
	html.WriteString("</body>\n")
	html.WriteString("</html>\n")

	return []byte(html.String()), nil
}
