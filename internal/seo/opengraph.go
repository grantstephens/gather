package seo

import (
	"fmt"
	"html"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// GenerateMetaTags creates Open Graph and Twitter Card meta tags for an event
func GenerateMetaTags(app core.App, event *core.Record, baseURL string) string {
	instanceName := "Gather"
	if settings, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	title := event.GetString("title")
	description := truncateText(stripMarkdown(event.GetString("description")), 200)
	eventURL := eventPageURL(baseURL, event)
	imageURL := ""
	if image := event.GetString("image"); image != "" {
		imageURL = fmt.Sprintf("%s/api/files/events/%s/%s", baseURL, event.Id, image)
	}

	return buildMetaTags(title, description, eventURL, imageURL, instanceName)
}

// buildMetaTags is the pure helper — accepts plain strings, testable without PocketBase
func buildMetaTags(title, description, eventURL, imageURL, instanceName string) string {
	var tags strings.Builder

	tags.WriteString(fmt.Sprintf(`<meta property="og:type" content="article">%s`, "\n"))
	tags.WriteString(fmt.Sprintf(`<meta property="og:site_name" content="%s">%s`, htmlEscape(instanceName), "\n"))
	tags.WriteString(fmt.Sprintf(`<meta property="og:title" content="%s">%s`, htmlEscape(title), "\n"))
	if description != "" {
		tags.WriteString(fmt.Sprintf(`<meta property="og:description" content="%s">%s`, htmlEscape(description), "\n"))
	}
	tags.WriteString(fmt.Sprintf(`<meta property="og:url" content="%s">%s`, htmlEscape(eventURL), "\n"))

	if imageURL != "" {
		tags.WriteString(fmt.Sprintf(`<meta property="og:image" content="%s">%s`, htmlEscape(imageURL), "\n"))
		tags.WriteString(fmt.Sprintf(`<meta property="og:image:alt" content="%s">%s`, htmlEscape(title), "\n"))
		tags.WriteString(fmt.Sprintf(`<meta property="og:image:width" content="800">%s`, "\n"))
		tags.WriteString(fmt.Sprintf(`<meta property="og:image:height" content="600">%s`, "\n"))
		tags.WriteString(fmt.Sprintf(`<meta name="twitter:card" content="summary_large_image">%s`, "\n"))
		tags.WriteString(fmt.Sprintf(`<meta name="twitter:image" content="%s">%s`, htmlEscape(imageURL), "\n"))
		tags.WriteString(fmt.Sprintf(`<meta name="twitter:image:alt" content="%s">%s`, htmlEscape(title), "\n"))
	} else {
		tags.WriteString(fmt.Sprintf(`<meta name="twitter:card" content="summary">%s`, "\n"))
	}
	tags.WriteString(fmt.Sprintf(`<meta name="twitter:title" content="%s">%s`, htmlEscape(title), "\n"))
	if description != "" {
		tags.WriteString(fmt.Sprintf(`<meta name="twitter:description" content="%s">%s`, htmlEscape(description), "\n"))
	}

	return tags.String()
}

// eventPageURL returns the canonical URL for an event, using slug if available.
func eventPageURL(baseURL string, event *core.Record) string {
	if s := event.GetString("slug"); s != "" {
		return fmt.Sprintf("%s/event/%s", baseURL, s)
	}
	return fmt.Sprintf("%s/event/%s", baseURL, event.Id)
}

// htmlEscape escapes HTML special characters to prevent XSS
func htmlEscape(s string) string {
	return html.EscapeString(s)
}

// truncateText truncates text to maxLen characters, adding "..." if truncated
func truncateText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLen {
		return text
	}

	// Truncate at word boundary
	truncated := text[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
