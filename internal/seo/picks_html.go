package seo

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type picksEvent struct {
	Title     string
	URL       string
	StartDate string
}

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

	published := picks.GetDateTime("created").Time()
	modified := picks.GetDateTime("updated").Time()
	if published.IsZero() {
		published = time.Now()
	}
	if modified.IsZero() {
		modified = published
	}

	// Fetch all events in the picks for body listing and image
	imageURL := ""
	imageType := "image/webp"
	var events []picksEvent
	for _, eventID := range picks.GetStringSlice("events") {
		ev, err := app.FindRecordById("events", eventID)
		if err != nil {
			continue
		}

		evSlug := ev.GetString("slug")
		if evSlug == "" {
			evSlug = ev.Id
		}
		evURL := fmt.Sprintf("%s/event/%s", baseURL, evSlug)

		startTime := ev.GetDateTime("start_datetime").Time()
		dateStr := ""
		if !startTime.IsZero() {
			dateStr = startTime.Format("Monday 2 January, 3:04 PM")
		}

		events = append(events, picksEvent{
			Title:     ev.GetString("title"),
			URL:       evURL,
			StartDate: dateStr,
		})

		if imageURL == "" {
			if image := ev.GetString("image"); image != "" {
				imageURL = fmt.Sprintf("%s/api/files/events/%s/%s", baseURL, ev.Id, image)
				ext := strings.ToLower(filepath.Ext(image))
				switch ext {
				case ".jpg", ".jpeg":
					imageType = "image/jpeg"
				case ".png":
					imageType = "image/png"
				case ".gif":
					imageType = "image/gif"
				default:
					imageType = "image/webp"
				}
			}
		}
	}

	return []byte(buildPicksHTML(
		title, instanceName, description, picksURL, baseURL,
		imageURL, imageType, published, modified, events,
	)), nil
}

func buildPicksHTML(
	title, instanceName, description, picksURL, baseURL,
	imageURL, imageType string,
	published, modified time.Time,
	events []picksEvent,
) string {
	var b strings.Builder

	publishedISO := published.UTC().Format(time.RFC3339)
	modifiedISO := modified.UTC().Format(time.RFC3339)

	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("<html lang=\"en\">\n<head>\n")
	b.WriteString("  <meta charset=\"UTF-8\">\n")
	b.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	b.WriteString(fmt.Sprintf("  <title>%s - %s</title>\n", htmlEscape(title), htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf("  <link rel=\"canonical\" href=\"%s\">\n", htmlEscape(picksURL)))
	if description != "" {
		b.WriteString(fmt.Sprintf("  <meta name=\"description\" content=\"%s\">\n", htmlEscape(truncateText(description, 160))))
	}

	// Open Graph
	b.WriteString("  <meta property=\"og:type\" content=\"article\">\n")
	b.WriteString(fmt.Sprintf("  <meta property=\"og:site_name\" content=\"%s\">\n", htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:title\" content=\"%s\">\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:url\" content=\"%s\">\n", htmlEscape(picksURL)))
	if description != "" {
		b.WriteString(fmt.Sprintf("  <meta property=\"og:description\" content=\"%s\">\n", htmlEscape(truncateText(description, 200))))
	}
	b.WriteString(fmt.Sprintf("  <meta property=\"article:published_time\" content=\"%s\">\n", publishedISO))
	b.WriteString(fmt.Sprintf("  <meta property=\"article:modified_time\" content=\"%s\">\n", modifiedISO))
	if imageURL != "" {
		b.WriteString(fmt.Sprintf("  <meta property=\"og:image\" content=\"%s\">\n", htmlEscape(imageURL)))
		b.WriteString(fmt.Sprintf("  <meta property=\"og:image:type\" content=\"%s\">\n", imageType))
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

	// JSON-LD
	type publisher struct {
		Type string `json:"@type"`
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	type articleLD struct {
		Context      string    `json:"@context"`
		Type         string    `json:"@type"`
		Headline     string    `json:"headline"`
		Description  string    `json:"description,omitempty"`
		URL          string    `json:"url"`
		DatePublished string   `json:"datePublished"`
		DateModified  string   `json:"dateModified"`
		Image        string    `json:"image,omitempty"`
		Publisher    publisher `json:"publisher"`
	}
	ld := articleLD{
		Context:       "https://schema.org",
		Type:          "Article",
		Headline:      title,
		Description:   description,
		URL:           picksURL,
		DatePublished: publishedISO,
		DateModified:  modifiedISO,
		Image:         imageURL,
		Publisher:     publisher{Type: "Organization", Name: instanceName, URL: baseURL},
	}
	if ldJSON, err := json.Marshal(ld); err == nil {
		b.WriteString("  <script type=\"application/ld+json\">\n  ")
		b.WriteString(string(ldJSON))
		b.WriteString("\n  </script>\n")
	}

	b.WriteString("</head>\n<body>\n")
	b.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", htmlEscape(title)))
	if description != "" {
		b.WriteString(fmt.Sprintf("  <p>%s</p>\n", htmlEscape(description)))
	}
	if len(events) > 0 {
		b.WriteString("  <h2>This weekend's picks</h2>\n  <ul>\n")
		for _, ev := range events {
			if ev.StartDate != "" {
				b.WriteString(fmt.Sprintf("    <li><a href=\"%s\">%s</a> &mdash; %s</li>\n",
					htmlEscape(ev.URL), htmlEscape(ev.Title), htmlEscape(ev.StartDate)))
			} else {
				b.WriteString(fmt.Sprintf("    <li><a href=\"%s\">%s</a></li>\n",
					htmlEscape(ev.URL), htmlEscape(ev.Title)))
			}
		}
		b.WriteString("  </ul>\n")
	}
	b.WriteString(fmt.Sprintf("  <p><a href=\"%s\">Read the full picks post</a></p>\n", htmlEscape(picksURL)))
	b.WriteString("</body>\n</html>\n")
	return b.String()
}
