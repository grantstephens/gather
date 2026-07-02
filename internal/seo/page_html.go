package seo

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// GeneratePageHTML renders a custom page for bots.
// Returns an error if the page doesn't exist or is hidden.
func GeneratePageHTML(app core.App, slug, baseURL string) ([]byte, error) {
	instanceName := "Gather"
	if settings, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	page, err := app.FindFirstRecordByFilter(
		"pages",
		"slug = {:slug} && hidden = false",
		map[string]any{"slug": slug},
	)
	if err != nil {
		return nil, fmt.Errorf("page not found: %s", slug)
	}

	title := page.GetString("title")
	content := page.GetString("content")
	pageURL := fmt.Sprintf("%s/%s", baseURL, slug)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("  <meta charset=\"UTF-8\">\n")
	b.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	b.WriteString(fmt.Sprintf("  <title>%s - %s</title>\n", htmlEscape(title), htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf("  <link rel=\"canonical\" href=\"%s\">\n", htmlEscape(pageURL)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:type\" content=\"website\">\n"))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:site_name\" content=\"%s\">\n", htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:title\" content=\"%s - %s\">\n", htmlEscape(title), htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:url\" content=\"%s\">\n", htmlEscape(pageURL)))
	b.WriteString("</head>\n<body>\n")
	b.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", htmlEscape(title)))
	b.WriteString(content)
	b.WriteString("\n</body>\n</html>\n")

	return []byte(b.String()), nil
}
