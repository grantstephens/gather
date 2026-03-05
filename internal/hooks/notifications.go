package hooks

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const (
	// Default values for missing or invalid data
	unknownSubmitter = "Unknown"
	defaultLocation  = "Online/TBD"

	// Email display format for event times
	emailTimeFormat = "Monday, January 2, 2006 at 3:04 PM MST"
)

// formatEventTime formats a datetime string for email display.
// Returns the original string if parsing fails (graceful degradation).
func formatEventTime(datetimeStr string) string {
	t, err := time.Parse(time.RFC3339, datetimeStr)
	if err != nil {
		// Return original string rather than failing email send
		return datetimeStr
	}
	return t.Format(emailTimeFormat)
}

// getSubmitterInfo returns a formatted string describing the event submitter.
// Returns "Unknown" if submitter information cannot be determined (graceful degradation).
func getSubmitterInfo(app core.App, event core.Record) string {
	// Check for anonymous submission first
	authorEmail := event.GetString("author_email")
	if authorEmail != "" {
		return authorEmail
	}

	// Check for registered user
	authorID := event.GetString("author")
	if authorID == "" {
		return unknownSubmitter
	}

	author, err := app.FindRecordById("users", authorID)
	if err != nil {
		// Database lookup failed, return default
		return unknownSubmitter
	}

	displayName := author.GetString("display_name")
	email := author.GetString("email")

	if displayName != "" {
		return fmt.Sprintf("%s (%s)", displayName, email)
	}
	return email
}

// getLocationString returns a formatted location string for an event.
// Returns "Online/TBD" if place information is missing or cannot be retrieved (graceful degradation).
func getLocationString(app core.App, event core.Record) string {
	placeID := event.GetString("place")
	if placeID == "" {
		return defaultLocation
	}

	place, err := app.FindRecordById("places", placeID)
	if err != nil {
		// Database lookup failed, return default
		return defaultLocation
	}

	return place.GetString("name")
}
