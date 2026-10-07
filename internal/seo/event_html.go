package seo

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// GenerateEventHTML renders the full bot-facing page for an event.
func GenerateEventHTML(app core.App, event *core.Record, baseURL string) ([]byte, error) {
	page, err := BuildEventPage(app, event, baseURL, time.Now())
	if err != nil {
		return nil, err
	}
	return []byte(RenderEventHTML(page)), nil
}

// buildEventHTML renders a bare upcoming event page from plain strings. It is
// kept as a small, database-free entry point for tests of the basic layout.
func buildEventHTML(title, instanceName, description, eventURL, metaTags, jsonLD string) string {
	return RenderEventHTML(&EventPage{
		Title: title, InstanceName: instanceName, URL: eventURL,
		Description: description, MetaDescription: description,
		State: Upcoming, MetaTags: metaTags, JSONLD: jsonLD,
	})
}
