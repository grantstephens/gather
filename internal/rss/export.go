package rss

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate,omitempty"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func GenerateFeed(app core.App, baseURL string, filter string) ([]byte, error) {
	// Get settings
	settings, _ := app.FindFirstRecordByFilter("settings", "id != ''")

	instanceName := "Gather"
	instanceDesc := "Community Events"
	if settings != nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
		if desc := settings.GetString("instance_description"); desc != "" {
			instanceDesc = desc
		}
	}

	// Build filter
	fullFilter := "status = 'published'"
	if filter != "" {
		fullFilter += " && " + filter
	}

	// Get events
	events, err := app.FindRecordsByFilter(
		"events",
		fullFilter,
		"-start_datetime",
		50,
		0,
	)
	if err != nil {
		return nil, err
	}

	// Build items
	items := make([]Item, 0, len(events))
	for _, event := range events {
		items = append(items, Item{
			Title:       event.GetString("title"),
			Link:        fmt.Sprintf("%s/event/%s", baseURL, event.Id),
			Description: event.GetString("description"),
			PubDate:     event.GetDateTime("created").Time().Format(time.RFC1123Z),
			GUID:        fmt.Sprintf("%s/event/%s", baseURL, event.Id),
		})
	}

	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       instanceName,
			Link:        baseURL,
			Description: instanceDesc,
			Items:       items,
		},
	}

	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}
