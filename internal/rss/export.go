package rss

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type RSS struct {
	XMLName  xml.Name `xml:"rss"`
	Version  string   `xml:"version,attr"`
	AtomNS   string   `xml:"xmlns:atom,attr"`
	Channel  Channel  `xml:"channel"`
}

type Channel struct {
	Title         string   `xml:"title"`
	Link          string   `xml:"link"`
	Description   string   `xml:"description"`
	Language      string   `xml:"language"`
	LastBuildDate string   `xml:"lastBuildDate"`
	AtomLink      AtomLink `xml:"atom:link"`
	Items         []Item   `xml:"item"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func eventURL(baseURL string, event *core.Record) string {
	if slug := event.GetString("slug"); slug != "" {
		return fmt.Sprintf("%s/event/%s", baseURL, slug)
	}
	return fmt.Sprintf("%s/event/%s", baseURL, event.Id)
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
		pubDate := event.GetDateTime("start_datetime").Time()
		if pubDate.IsZero() {
			pubDate = event.GetDateTime("created").Time()
		}
		url := eventURL(baseURL, event)
		items = append(items, Item{
			Title:       event.GetString("title"),
			Link:        url,
			Description: event.GetString("description"),
			PubDate:     pubDate.Format(time.RFC1123Z),
			GUID:        url,
		})
	}

	feedURL := baseURL + "/feed.rss"
	rss := RSS{
		Version: "2.0",
		AtomNS:  "http://www.w3.org/2005/Atom",
		Channel: Channel{
			Title:         instanceName,
			Link:          baseURL,
			Description:   instanceDesc,
			Language:      "en",
			LastBuildDate: time.Now().UTC().Format(time.RFC1123Z),
			AtomLink: AtomLink{
				Href: feedURL,
				Rel:  "self",
				Type: "application/rss+xml",
			},
			Items: items,
		},
	}

	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}
