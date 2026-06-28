package seo

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// GeneratePicksHTML creates a complete HTML page with meta tags for a picks post.
func GeneratePicksHTML(app core.App, slug, baseURL string) ([]byte, error) {
	instanceName := "Gather"
	if settings, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	picks, err := app.FindFirstRecordByFilter(
		"picks",
		"slug = {:slug} && hidden = false",
		map[string]any{"slug": slug},
	)
	if err != nil {
		return nil, err
	}

	title := picks.GetString("title")
	description := truncateText(stripMarkdown(picks.GetString("blurb")), 300)
	picksURL := fmt.Sprintf("%s/picks/%s", baseURL, slug)

	// Find the first event with an image for the OG image
	imageURL := ""
	for _, eventID := range picks.GetStringSlice("events") {
		event, err := app.FindRecordById("events", eventID)
		if err != nil {
			continue
		}
		if image := event.GetString("image"); image != "" {
			imageURL = fmt.Sprintf("%s/api/files/events/%s/%s", baseURL, event.Id, image)
			break
		}
	}

	return []byte(buildPicksHTML(title, instanceName, description, picksURL, imageURL)), nil
}

func buildPicksHTML(title, instanceName, description, picksURL, imageURL string) string {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("<html lang=\"en\">\n<head>\n")
	b.WriteString("  <meta charset=\"UTF-8\">\n")
	b.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	b.WriteString(fmt.Sprintf("  <title>%s - %s</title>\n", htmlEscape(title), htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf("  <link rel=\"canonical\" href=\"%s\">\n", htmlEscape(picksURL)))
	if description != "" {
		b.WriteString(fmt.Sprintf("  <meta name=\"description\" content=\"%s\">\n", htmlEscape(truncateText(description, 160))))
	}
	b.WriteString("  <meta property=\"og:type\" content=\"article\">\n")
	b.WriteString(fmt.Sprintf("  <meta property=\"og:site_name\" content=\"%s\">\n", htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:title\" content=\"%s\">\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:url\" content=\"%s\">\n", htmlEscape(picksURL)))
	if description != "" {
		b.WriteString(fmt.Sprintf("  <meta property=\"og:description\" content=\"%s\">\n", htmlEscape(truncateText(description, 200))))
	}
	if imageURL != "" {
		b.WriteString(fmt.Sprintf("  <meta property=\"og:image\" content=\"%s\">\n", htmlEscape(imageURL)))
		b.WriteString(fmt.Sprintf("  <meta property=\"og:image:alt\" content=\"%s\">\n", htmlEscape(title)))
		b.WriteString("  <meta property=\"og:image:width\" content=\"800\">\n")
		b.WriteString("  <meta property=\"og:image:height\" content=\"600\">\n")
		b.WriteString("  <meta name=\"twitter:card\" content=\"summary_large_image\">\n")
		b.WriteString(fmt.Sprintf("  <meta name=\"twitter:image\" content=\"%s\">\n", htmlEscape(imageURL)))
		b.WriteString(fmt.Sprintf("  <meta name=\"twitter:image:alt\" content=\"%s\">\n", htmlEscape(title)))
	} else {
		b.WriteString("  <meta name=\"twitter:card\" content=\"summary\">\n")
	}
	b.WriteString(fmt.Sprintf("  <meta name=\"twitter:title\" content=\"%s\">\n", htmlEscape(title)))
	if description != "" {
		b.WriteString(fmt.Sprintf("  <meta name=\"twitter:description\" content=\"%s\">\n", htmlEscape(truncateText(description, 200))))
	}
	b.WriteString("</head>\n<body>\n")
	b.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", htmlEscape(title)))
	if description != "" {
		b.WriteString(fmt.Sprintf("  <p>%s</p>\n", htmlEscape(description)))
	}
	b.WriteString(fmt.Sprintf("  <p><a href=\"%s\">Read the full picks post</a></p>\n", htmlEscape(picksURL)))
	b.WriteString("</body>\n</html>\n")
	return b.String()
}
