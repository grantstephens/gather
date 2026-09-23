package seo

import (
	"encoding/json"
	"fmt"
)

// AICatalogHost describes the publisher in an ai-catalog.json / ard.json manifest.
type AICatalogHost struct {
	DisplayName string `json:"displayName"`
	Identifier  string `json:"identifier"`
}

// AICatalogEntry describes a single agentic resource per the ARD spec
// (https://agenticresourcediscovery.org/spec/, §4.2). Only fields the spec
// actually defines are included, since the predecessor schema Lighthouse
// currently validates against rejects unrecognized properties.
type AICatalogEntry struct {
	Identifier            string   `json:"identifier"`
	DisplayName           string   `json:"displayName"`
	Type                  string   `json:"type"`
	URL                   string   `json:"url"`
	Description           string   `json:"description,omitempty"`
	RepresentativeQueries []string `json:"representativeQueries,omitempty"`
	Capabilities          []string `json:"capabilities,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
}

// AICatalog is the root manifest shape for both /.well-known/ai-catalog.json
// (predecessor path) and /.well-known/ard.json (current spec path). Keep the
// root strictly to specVersion/host/entries: the predecessor schema Lighthouse
// ships today is additionalProperties:false at root.
type AICatalog struct {
	SpecVersion string           `json:"specVersion"`
	Host        AICatalogHost    `json:"host"`
	Entries     []AICatalogEntry `json:"entries"`
}

// BuildAICatalog returns the ARD manifest describing this instance's public,
// machine-queryable resources (event/place search and RSS/iCal feeds) so AI
// agents and ARD registries can discover them without scraping HTML.
func BuildAICatalog(baseURL, instanceName string) AICatalog {
	return AICatalog{
		SpecVersion: "1.0",
		Host: AICatalogHost{
			DisplayName: instanceName,
			Identifier:  "did:web:" + hostFromBaseURL(baseURL),
		},
		Entries: []AICatalogEntry{
			{
				Identifier:  "urn:air:" + hostFromBaseURL(baseURL) + ":feed:events-rss",
				DisplayName: instanceName + " RSS Feed",
				Type:        "application/rss+xml",
				URL:         baseURL + "/feed.rss",
				Description: "RSS feed of all upcoming published community events on " + instanceName + ".",
				RepresentativeQueries: []string{
					"what events are happening in " + instanceName,
					"subscribe to a feed of upcoming community events",
					"get the latest published events",
				},
				Tags: []string{"events", "calendar", "rss"},
			},
			{
				Identifier:  "urn:air:" + hostFromBaseURL(baseURL) + ":feed:events-ical",
				DisplayName: instanceName + " iCal Feed",
				Type:        "text/calendar",
				URL:         baseURL + "/feed.ics",
				Description: "iCalendar feed of all upcoming published community events on " + instanceName + ", for subscribing in any calendar app.",
				RepresentativeQueries: []string{
					"add these community events to my calendar",
					"subscribe to an events calendar feed",
					"sync upcoming events to my calendar app",
				},
				Tags: []string{"events", "calendar", "ical"},
			},
			{
				Identifier:  "urn:air:" + hostFromBaseURL(baseURL) + ":api:search",
				DisplayName: instanceName + " Search API",
				Type:        "application/json",
				URL:         baseURL + "/api/search?q=",
				Description: "Unified full-text search across events, places, and tags. Append a URL-encoded query to the q parameter.",
				RepresentativeQueries: []string{
					"search for live music events",
					"find a venue by name",
					"look up events tagged with a specific category",
				},
				Capabilities: []string{"SearchEvents", "SearchPlaces", "SearchTags"},
				Tags:         []string{"events", "search"},
			},
			{
				Identifier:  "urn:air:" + hostFromBaseURL(baseURL) + ":api:events-expanded",
				DisplayName: instanceName + " Events Listing API",
				Type:        "application/json",
				URL:         baseURL + "/api/events/expanded",
				Description: "Paginated, recurrence-expanded event listings filterable by date range (start, end) and town.",
				RepresentativeQueries: []string{
					"list events happening this weekend",
					"find events in a specific town between two dates",
					"get all recurring event instances for the next month",
				},
				Capabilities: []string{"ListEvents", "FilterByDateRange", "FilterByTown"},
				Tags:         []string{"events", "calendar"},
			},
			{
				Identifier:  "urn:air:" + hostFromBaseURL(baseURL) + ":api:places-nearby",
				DisplayName: instanceName + " Places Nearby API",
				Type:        "application/json",
				URL:         baseURL + "/api/places/nearby",
				Description: "Find approved event venues near a given latitude/longitude within a radius in kilometres.",
				RepresentativeQueries: []string{
					"find event venues near a location",
					"what places host events within a few kilometres of a point",
				},
				Capabilities: []string{"FindNearbyPlaces"},
				Tags:         []string{"places", "venues", "geo"},
			},
		},
	}
}

// BuildAICatalogJSON marshals the manifest to indented JSON bytes.
func BuildAICatalogJSON(baseURL, instanceName string) ([]byte, error) {
	return json.MarshalIndent(BuildAICatalog(baseURL, instanceName), "", "  ")
}

// hostFromBaseURL strips the scheme from a baseURL (e.g. "https://example.com" -> "example.com")
// for use in did:web and urn:air identifiers.
func hostFromBaseURL(baseURL string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if len(baseURL) > len(prefix) && baseURL[:len(prefix)] == prefix {
			return baseURL[len(prefix):]
		}
	}
	return baseURL
}

// BuildLLMsTxt returns an llms.txt file (https://llmstxt.org/) pointing AI
// agents already browsing the site at its most useful entry points and feeds.
func BuildLLMsTxt(baseURL, instanceName, description string) string {
	if description == "" {
		description = "A community events calendar."
	}
	return fmt.Sprintf(`# %s

> %s

## Product

- [Homepage](%s/): Browse upcoming events, filterable by date, town, and category tag.
- [About](%s/about): What %s is and how it works.
- [Submit an Event](%s/submit): Add a community event to the calendar.

## Data & Feeds

- [Search API](%s/api/search): Full-text search across events, places, and tags.
- [Events Listing API](%s/api/events/expanded): Paginated, recurrence-expanded event listings filterable by date range and town.
- [Places Nearby API](%s/api/places/nearby): Find approved event venues near a latitude/longitude.
- [RSS Feed](%s/feed.rss): All upcoming published events.
- [iCal Feed](%s/feed.ics): Subscribe to all upcoming events in any calendar app.

## Optional

- [Editor's Picks](%s/picks): Curated highlights of upcoming events.
- [Agent Capability Catalog](%s/.well-known/ai-catalog.json): Machine-readable ARD manifest of the above APIs and feeds.
`, instanceName, description, baseURL, baseURL, instanceName, baseURL, baseURL, baseURL, baseURL, baseURL, baseURL, baseURL, baseURL)
}
