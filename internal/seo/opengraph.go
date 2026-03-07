package seo

import (
	"fmt"
	"html"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// GenerateMetaTags creates Open Graph and Twitter Card meta tags for an event
func GenerateMetaTags(app core.App, event *core.Record, baseURL string) string {
	var tags strings.Builder

	// Open Graph tags
	tags.WriteString(fmt.Sprintf(`<meta property="og:type" content="event">%s`, "\n"))
	tags.WriteString(fmt.Sprintf(`<meta property="og:title" content="%s">%s`, htmlEscape(event.GetString("title")), "\n"))

	// Truncate description to 200 chars
	description := truncateText(stripMarkdown(event.GetString("description")), 200)
	if description != "" {
		tags.WriteString(fmt.Sprintf(`<meta property="og:description" content="%s">%s`, htmlEscape(description), "\n"))
	}

	eventURL := fmt.Sprintf("%s/event/%s", baseURL, event.Id)
	tags.WriteString(fmt.Sprintf(`<meta property="og:url" content="%s">%s`, htmlEscape(eventURL), "\n"))

	// Image tags
	if image := event.GetString("image"); image != "" {
		imageURL := fmt.Sprintf("%s/api/files/events/%s/%s", baseURL, event.Id, image)
		tags.WriteString(fmt.Sprintf(`<meta property="og:image" content="%s">%s`, htmlEscape(imageURL), "\n"))
		tags.WriteString(fmt.Sprintf(`<meta property="og:image:width" content="800">%s`, "\n"))
		tags.WriteString(fmt.Sprintf(`<meta property="og:image:height" content="600">%s`, "\n"))
	}

	// Twitter Card tags
	if image := event.GetString("image"); image != "" {
		tags.WriteString(fmt.Sprintf(`<meta name="twitter:card" content="summary_large_image">%s`, "\n"))
		imageURL := fmt.Sprintf("%s/api/files/events/%s/%s", baseURL, event.Id, image)
		tags.WriteString(fmt.Sprintf(`<meta name="twitter:image" content="%s">%s`, htmlEscape(imageURL), "\n"))
	} else {
		tags.WriteString(fmt.Sprintf(`<meta name="twitter:card" content="summary">%s`, "\n"))
	}

	tags.WriteString(fmt.Sprintf(`<meta name="twitter:title" content="%s">%s`, htmlEscape(event.GetString("title")), "\n"))
	if description != "" {
		tags.WriteString(fmt.Sprintf(`<meta name="twitter:description" content="%s">%s`, htmlEscape(description), "\n"))
	}

	return tags.String()
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
