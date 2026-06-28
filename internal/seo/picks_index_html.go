package seo

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type picksListItem struct {
	Title     string
	URL       string
	Blurb     string
	StartDate string
}

// GeneratePicksIndexHTML creates a complete HTML page for the picks listing page.
func GeneratePicksIndexHTML(app core.App, baseURL string) ([]byte, error) {
	instanceName := "Gather"
	if settings, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	picksURL := fmt.Sprintf("%s/picks", baseURL)
	title := fmt.Sprintf("Weekend Picks - %s", instanceName)
	description := "Curated picks of the best events happening this weekend and beyond."

	records, _ := app.FindRecordsByFilter("picks", "hidden = false", "-start_date", 50, 0)

	items := make([]picksListItem, 0, len(records))
	for _, p := range records {
		slug := p.GetString("slug")
		if slug == "" {
			slug = p.Id
		}
		blurb := truncateText(stripMarkdown(p.GetString("blurb")), 160)
		dateStr := ""
		if sd := p.GetString("start_date"); sd != "" {
			if t, err := time.Parse("2006-01-02", sd); err == nil {
				dateStr = t.Format("2 January 2006")
			}
		}
		items = append(items, picksListItem{
			Title:     p.GetString("title"),
			URL:       fmt.Sprintf("%s/picks/%s", baseURL, slug),
			Blurb:     blurb,
			StartDate: dateStr,
		})
	}

	return []byte(buildPicksIndexHTML(title, instanceName, description, picksURL, baseURL, items)), nil
}

func buildPicksIndexHTML(title, instanceName, description, picksURL, baseURL string, items []picksListItem) string {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("<html lang=\"en\">\n<head>\n")
	b.WriteString("  <meta charset=\"UTF-8\">\n")
	b.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	b.WriteString(fmt.Sprintf("  <title>%s</title>\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <link rel=\"canonical\" href=\"%s\">\n", htmlEscape(picksURL)))
	b.WriteString(fmt.Sprintf("  <meta name=\"description\" content=\"%s\">\n", htmlEscape(description)))

	b.WriteString("  <meta property=\"og:type\" content=\"website\">\n")
	b.WriteString(fmt.Sprintf("  <meta property=\"og:site_name\" content=\"%s\">\n", htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:title\" content=\"%s\">\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:url\" content=\"%s\">\n", htmlEscape(picksURL)))
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
			{Type: "ListItem", Position: 2, Name: "Picks", Item: picksURL},
		},
	}
	if ldJSON, err := json.Marshal(ld); err == nil {
		b.WriteString("  <script type=\"application/ld+json\">\n  ")
		b.WriteString(string(ldJSON))
		b.WriteString("\n  </script>\n")
	}

	b.WriteString("</head>\n<body>\n")
	b.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <p>%s</p>\n", htmlEscape(description)))

	if len(items) > 0 {
		b.WriteString("  <ul>\n")
		for _, item := range items {
			b.WriteString("    <li>\n")
			b.WriteString(fmt.Sprintf("      <a href=\"%s\"><strong>%s</strong></a>", htmlEscape(item.URL), htmlEscape(item.Title)))
			if item.StartDate != "" {
				b.WriteString(fmt.Sprintf(" &mdash; %s", htmlEscape(item.StartDate)))
			}
			if item.Blurb != "" {
				b.WriteString(fmt.Sprintf("\n      <br>%s", htmlEscape(item.Blurb)))
			}
			b.WriteString("\n    </li>\n")
		}
		b.WriteString("  </ul>\n")
	}

	b.WriteString("</body>\n</html>\n")
	return b.String()
}
