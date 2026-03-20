package seo

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// EventJSONLD represents a Schema.org Event in JSON-LD format
type EventJSONLD struct {
	Context             string                 `json:"@context"`
	Type                string                 `json:"@type"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description,omitempty"`
	StartDate           string                 `json:"startDate"`
	EndDate             string                 `json:"endDate,omitempty"`
	EventStatus         string                 `json:"eventStatus"`
	EventAttendanceMode string                 `json:"eventAttendanceMode"`
	Location            *EventLocation         `json:"location,omitempty"`
	Image               string                 `json:"image,omitempty"`
	Organizer           *EventOrganizer        `json:"organizer,omitempty"`
	URL                 string                 `json:"url"`
}

// EventLocation represents a Schema.org Place for event location
type EventLocation struct {
	Type    string        `json:"@type"`
	Name    string        `json:"name"`
	Address *EventAddress `json:"address,omitempty"`
	Geo     *EventGeo     `json:"geo,omitempty"`
}

// EventAddress represents a Schema.org PostalAddress
type EventAddress struct {
	Type          string `json:"@type"`
	StreetAddress string `json:"streetAddress,omitempty"`
	AddressLocality string `json:"addressLocality,omitempty"`
	AddressCountry string `json:"addressCountry,omitempty"`
}

// EventGeo represents Schema.org GeoCoordinates
type EventGeo struct {
	Type      string  `json:"@type"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// EventOrganizer represents a Schema.org Organization
type EventOrganizer struct {
	Type string `json:"@type"`
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// GenerateEventJSONLD creates Schema.org Event JSON-LD from a PocketBase event record
func GenerateEventJSONLD(app core.App, event *core.Record, baseURL string) ([]byte, error) {
	jsonLD := &EventJSONLD{
		Context:             "https://schema.org",
		Type:                "Event",
		Name:                event.GetString("title"),
		Description:         stripMarkdown(event.GetString("description")),
		StartDate:           formatDateTime(event.GetDateTime("start_datetime")),
		EventStatus:         mapEventStatus(event.GetString("status")),
		EventAttendanceMode: determineAttendanceMode(event),
		URL:                 fmt.Sprintf("%s/event/%s", baseURL, event.Id),
	}

	// Add end date if it exists
	if endDateTime := event.GetDateTime("end_datetime"); !endDateTime.IsZero() {
		jsonLD.EndDate = formatDateTime(endDateTime)
	}

	// Add location if place exists
	if placeID := event.GetString("place"); placeID != "" {
		place, err := app.FindRecordById("places", placeID)
		if err == nil {
			jsonLD.Location = &EventLocation{
				Type: "Place",
				Name: place.GetString("name"),
				Geo: &EventGeo{
					Type:      "GeoCoordinates",
					Latitude:  place.GetFloat("latitude"),
					Longitude: place.GetFloat("longitude"),
				},
			}

			// Add address if available
			if address := place.GetString("address"); address != "" {
				jsonLD.Location.Address = &EventAddress{
					Type:          "PostalAddress",
					StreetAddress: address,
					AddressLocality: place.GetString("city"),
					AddressCountry: place.GetString("country_code"),
				}
			}
		}
	}

	// Add image if exists
	if image := event.GetString("image"); image != "" {
		jsonLD.Image = fmt.Sprintf("%s/api/files/events/%s/%s", baseURL, event.Id, image)
	}

	// Add organizer from settings
	settings, err := app.FindFirstRecordByFilter("settings", "")
	if err == nil {
		if instanceName := settings.GetString("instance_name"); instanceName != "" {
			jsonLD.Organizer = &EventOrganizer{
				Type: "Organization",
				Name: instanceName,
				URL:  baseURL,
			}
		}
	}

	return json.MarshalIndent(jsonLD, "", "  ")
}

// formatDateTime converts types.DateTime to ISO 8601 (RFC3339) format
func formatDateTime(t types.DateTime) string {
	if t.IsZero() {
		return ""
	}
	return t.Time().Format(time.RFC3339)
}

// mapEventStatus converts event status to Schema.org EventStatusType
func mapEventStatus(status string) string {
	switch status {
	case "cancelled":
		return "https://schema.org/EventCancelled"
	case "published":
		return "https://schema.org/EventScheduled"
	default:
		return "https://schema.org/EventScheduled"
	}
}

// determineAttendanceMode checks if event is online or offline
func determineAttendanceMode(event *core.Record) string {
	// If event has a physical place, it's offline
	if placeID := event.GetString("place"); placeID != "" {
		return "https://schema.org/OfflineEventAttendanceMode"
	}
	// If event has online_locations, it's online
	if onlineLocations := event.GetString("online_locations"); onlineLocations != "" && onlineLocations != "[]" && onlineLocations != "null" {
		return "https://schema.org/OnlineEventAttendanceMode"
	}
	// Default to offline
	return "https://schema.org/OfflineEventAttendanceMode"
}

// stripMarkdown removes markdown and HTML formatting from text
func stripMarkdown(text string) string {
	// Remove HTML tags
	text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, " ")
	// Remove markdown links [text](url)
	text = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`).ReplaceAllString(text, "$1")
	// Remove bold/italic **text** or *text*
	text = regexp.MustCompile(`\*\*([^\*]+)\*\*`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`\*([^\*]+)\*`).ReplaceAllString(text, "$1")
	// Remove headings
	text = regexp.MustCompile(`(?m)^#+\s+`).ReplaceAllString(text, "")
	// Remove list markers
	text = regexp.MustCompile(`(?m)^[\*\-\+]\s+`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`(?m)^\d+\.\s+`).ReplaceAllString(text, "")
	// Collapse multiple spaces/newlines
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n\n+`).ReplaceAllString(text, "\n")
	// Trim whitespace
	return strings.TrimSpace(text)
}
