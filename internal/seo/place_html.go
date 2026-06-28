package seo

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type placeEvent struct {
	Title     string
	URL       string
	StartDate string
}

// GeneratePlaceHTML creates a complete HTML page for a place listing page.
func GeneratePlaceHTML(app core.App, placeID, baseURL string) ([]byte, error) {
	instanceName := "Gather"
	if settings, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	place, err := app.FindRecordById("places", placeID)
	if err != nil {
		return nil, err
	}

	placeName := place.GetString("name")
	address := place.GetString("address")
	placeURL := fmt.Sprintf("%s/place/%s", baseURL, placeID)
	title := fmt.Sprintf("%s - %s", placeName, instanceName)
	description := fmt.Sprintf("Upcoming events at %s", placeName)
	if address != "" {
		description += fmt.Sprintf(", %s", address)
	}
	description += "."

	today := time.Now().Format("2006-01-02 15:04:05")
	events, _ := app.FindRecordsByFilter(
		"events",
		"status = 'published' && start_datetime >= {:today} && place = {:placeId}",
		"start_datetime",
		50,
		0,
		map[string]any{"today": today, "placeId": placeID},
	)

	items := make([]placeEvent, 0, len(events))
	for _, ev := range events {
		slug := ev.GetString("slug")
		if slug == "" {
			slug = ev.Id
		}
		dateStr := ""
		if t := ev.GetDateTime("start_datetime").Time(); !t.IsZero() {
			dateStr = t.Local().Format("Monday 2 January, 3:04 PM MST")
		}
		items = append(items, placeEvent{
			Title:     ev.GetString("title"),
			URL:       fmt.Sprintf("%s/event/%s", baseURL, slug),
			StartDate: dateStr,
		})
	}

	loc := place.GetGeoPoint("location")

	return []byte(buildPlaceHTML(title, instanceName, description, placeURL, baseURL, placeName, address, loc.Lat, loc.Lon, items)), nil
}

func buildPlaceHTML(title, instanceName, description, placeURL, baseURL, placeName, address string, lat, lon float64, items []placeEvent) string {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("<html lang=\"en\">\n<head>\n")
	b.WriteString("  <meta charset=\"UTF-8\">\n")
	b.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	b.WriteString(fmt.Sprintf("  <title>%s</title>\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <link rel=\"canonical\" href=\"%s\">\n", htmlEscape(placeURL)))
	b.WriteString(fmt.Sprintf("  <meta name=\"description\" content=\"%s\">\n", htmlEscape(description)))

	b.WriteString("  <meta property=\"og:type\" content=\"website\">\n")
	b.WriteString(fmt.Sprintf("  <meta property=\"og:site_name\" content=\"%s\">\n", htmlEscape(instanceName)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:title\" content=\"%s\">\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:url\" content=\"%s\">\n", htmlEscape(placeURL)))
	b.WriteString(fmt.Sprintf("  <meta property=\"og:description\" content=\"%s\">\n", htmlEscape(description)))
	b.WriteString("  <meta name=\"twitter:card\" content=\"summary\">\n")
	b.WriteString(fmt.Sprintf("  <meta name=\"twitter:title\" content=\"%s\">\n", htmlEscape(title)))
	b.WriteString(fmt.Sprintf("  <meta name=\"twitter:description\" content=\"%s\">\n", htmlEscape(description)))

	// Place JSON-LD
	type geoLD struct {
		Type      string  `json:"@type"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	type addressLD struct {
		Type          string `json:"@type"`
		StreetAddress string `json:"streetAddress,omitempty"`
	}
	type placeLD struct {
		Context     string     `json:"@context"`
		Type        string     `json:"@type"`
		Name        string     `json:"name"`
		URL         string     `json:"url"`
		Address     *addressLD `json:"address,omitempty"`
		Geo         *geoLD     `json:"geo,omitempty"`
	}
	ld := placeLD{
		Context: "https://schema.org",
		Type:    "Place",
		Name:    placeName,
		URL:     placeURL,
	}
	if address != "" {
		ld.Address = &addressLD{Type: "PostalAddress", StreetAddress: address}
	}
	if lat != 0 || lon != 0 {
		ld.Geo = &geoLD{Type: "GeoCoordinates", Latitude: lat, Longitude: lon}
	}
	if ldJSON, err := json.Marshal(ld); err == nil {
		b.WriteString("  <script type=\"application/ld+json\">\n  ")
		b.WriteString(string(ldJSON))
		b.WriteString("\n  </script>\n")
	}

	b.WriteString("</head>\n<body>\n")
	b.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", htmlEscape(placeName)))
	if address != "" {
		b.WriteString(fmt.Sprintf("  <p>%s</p>\n", htmlEscape(address)))
	}

	if len(items) > 0 {
		b.WriteString("  <h2>Upcoming events</h2>\n  <ul>\n")
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
		b.WriteString("  <p>No upcoming events at this venue.</p>\n")
	}

	b.WriteString("</body>\n</html>\n")
	return b.String()
}
