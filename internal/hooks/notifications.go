package hooks

import (
	"fmt"
	"html"
	"log"
	"net/mail"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

const (
	// Default values for missing or invalid data
	unknownSubmitter = "Unknown"
	defaultLocation  = "Online/TBD"

	// Email display format for event times
	emailTimeFormat = "Monday, January 2, 2006 at 3:04 PM MST"

	// Application name
	appName = "Gather"
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

// sendModeratorAlert sends email to all moderators when a new event needs review
func sendModeratorAlert(app core.App, event core.Record, baseURL string) {
	// Only send for pending events
	if event.GetString("status") != "pending" {
		return
	}

	// Find all moderators (admin or editor role)
	moderators, err := app.FindRecordsByFilter("users", "role='admin' || role='editor'", "", 0, 0)
	if err != nil {
		log.Printf("[WARN] Failed to find moderators for event %s: %v", event.Id, err)
		return
	}

	if len(moderators) == 0 {
		log.Printf("[WARN] No moderators found to notify for event %s", event.Id)
		return
	}

	// Get event details
	title := event.GetString("title")
	submitterInfo := getSubmitterInfo(app, event)
	startTime := formatEventTime(event.GetString("start_datetime"))
	location := getLocationString(app, event)
	reviewLink := fmt.Sprintf("%s/_/#/collections?collectionId=events&filter=id='%s'", baseURL, event.Id)

	// Build email content
	// Sanitize title for email subject (prevent header injection)
	safeTitle := strings.ReplaceAll(strings.ReplaceAll(title, "\n", " "), "\r", " ")
	subject := fmt.Sprintf("New Event Pending Review: %s", safeTitle)

	htmlBody := fmt.Sprintf(`<p>A new event has been submitted and needs review.</p>

<p><strong>Event:</strong> %s<br>
<strong>Submitted by:</strong> %s<br>
<strong>Start:</strong> %s<br>
<strong>Location:</strong> %s</p>

<p><a href="%s">Review this event</a></p>

<hr>
<p><small>This is an automated notification from %s.</small></p>`,
		html.EscapeString(title),
		html.EscapeString(submitterInfo),
		html.EscapeString(startTime),
		html.EscapeString(location),
		reviewLink,
		appName)

	textBody := fmt.Sprintf(`A new event has been submitted and needs review.

Event: %s
Submitted by: %s
Start: %s
Location: %s

Review this event:
%s

---
This is an automated notification from %s.`,
		title, submitterInfo, startTime, location, reviewLink, appName)

	// Send to each moderator
	mailClient := app.NewMailClient()
	for _, moderator := range moderators {
		email := moderator.GetString("email")
		if email == "" {
			log.Printf("[WARN] Moderator %s has no email address, skipping notification for event %s", moderator.Id, event.Id)
			continue
		}

		message := &mailer.Message{
			From: mail.Address{
				Name:    app.Settings().Meta.SenderName,
				Address: app.Settings().Meta.SenderAddress,
			},
			To:      []mail.Address{{Address: email}},
			Subject: subject,
			HTML:    htmlBody,
			Text:    textBody,
		}

		if err := mailClient.Send(message); err != nil {
			log.Printf("[WARN] Failed to send moderator alert to %s for event %s: %v", email, event.Id, err)
		}
	}
}
