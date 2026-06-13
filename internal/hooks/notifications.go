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
	unknownSubmitter = "Unknown"
	defaultLocation  = "Online/TBD"
	emailTimeFormat  = "Monday, January 2, 2006 at 3:04 PM MST"
	appName          = "Gather"
)

// emailHTML wraps body content in a branded HTML shell for outgoing emails.
// All styles are inlined since email clients strip <head> CSS.
func emailHTML(body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f1f5f9;font-family:Inter,system-ui,-apple-system,sans-serif;color:#0f172a;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f1f5f9;padding:32px 16px;">
    <tr><td align="center">
      <table width="100%%" cellpadding="0" cellspacing="0" style="max-width:600px;">

        <!-- Header -->
        <tr>
          <td style="background:#0d9488;border-radius:8px 8px 0 0;padding:24px 32px;">
            <span style="font-size:1.25rem;font-weight:700;color:#ffffff;letter-spacing:-0.01em;">%s</span>
          </td>
        </tr>

        <!-- Body -->
        <tr>
          <td style="background:#ffffff;padding:32px;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;">
            %s
          </td>
        </tr>

        <!-- Footer -->
        <tr>
          <td style="background:#f8fafc;border:1px solid #e2e8f0;border-top:none;border-radius:0 0 8px 8px;padding:16px 32px;">
            <p style="margin:0;font-size:0.75rem;color:#475569;">This is an automated notification from %s.</p>
          </td>
        </tr>

      </table>
    </td></tr>
  </table>
</body>
</html>`, appName, body, appName)
}

func formatEventTime(datetimeStr string) string {
	t, err := time.Parse(time.RFC3339, datetimeStr)
	if err != nil {
		return datetimeStr
	}
	return t.Format(emailTimeFormat)
}

func getSubmitterInfo(app core.App, event core.Record) string {
	authorEmail := event.GetString("author_email")
	if authorEmail != "" {
		return authorEmail
	}

	authorID := event.GetString("author")
	if authorID == "" {
		return unknownSubmitter
	}

	author, err := app.FindRecordById("users", authorID)
	if err != nil {
		return unknownSubmitter
	}

	displayName := author.GetString("display_name")
	email := author.GetString("email")

	if displayName != "" {
		return fmt.Sprintf("%s (%s)", displayName, email)
	}
	return email
}

func getLocationString(app core.App, event core.Record) string {
	placeID := event.GetString("place")
	if placeID == "" {
		return defaultLocation
	}

	place, err := app.FindRecordById("places", placeID)
	if err != nil {
		return defaultLocation
	}

	return place.GetString("name")
}

func getSubmitterEmail(app core.App, event core.Record) string {
	authorEmail := event.GetString("author_email")
	if authorEmail != "" {
		return authorEmail
	}

	authorID := event.GetString("author")
	if authorID == "" {
		return ""
	}

	author, err := app.FindRecordById("users", authorID)
	if err != nil {
		log.Printf("[WARN] Failed to find author %s for event %s: %v", authorID, event.Id, err)
		return ""
	}

	return author.GetString("email")
}

func sendApprovalNotification(app core.App, event core.Record, baseURL string) {
	email := getSubmitterEmail(app, event)
	if email == "" {
		log.Printf("[WARN] No email found for event %s approval notification", event.Id)
		return
	}

	title := event.GetString("title")
	startTime := formatEventTime(event.GetString("start_datetime"))
	eventLink := fmt.Sprintf("%s/event/%s", baseURL, event.Id)

	subject := fmt.Sprintf("Your Event Has Been Published: %s", title)

	bodyHTML := fmt.Sprintf(`
<p style="margin:0 0 16px;font-size:1rem;line-height:1.6;">Good news! Your event has been approved and is now live.</p>
<table cellpadding="0" cellspacing="0" style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:6px;padding:16px 20px;margin:0 0 24px;width:100%%;">
  <tr><td style="font-size:0.875rem;color:#475569;padding-bottom:6px;">Event</td></tr>
  <tr><td style="font-size:1rem;font-weight:600;color:#0f172a;padding-bottom:12px;">%s</td></tr>
  <tr><td style="font-size:0.875rem;color:#475569;padding-bottom:6px;">Start</td></tr>
  <tr><td style="font-size:0.875rem;color:#0f172a;">%s</td></tr>
</table>
<a href="%s" style="display:inline-block;background:#0d9488;color:#ffffff;text-decoration:none;font-weight:600;font-size:0.875rem;padding:10px 20px;border-radius:6px;">View your published event</a>`,
		html.EscapeString(title),
		html.EscapeString(startTime),
		eventLink)

	textBody := fmt.Sprintf(`Good news! Your event has been approved and is now live.

Event: %s
Start: %s

View your published event:
%s

---
This is an automated notification from %s.`,
		title, startTime, eventLink, appName)

	sendMail(app, email, subject, emailHTML(bodyHTML), textBody, event.Id)
}

func sendRejectionNotification(app core.App, event core.Record, baseURL string) {
	email := getSubmitterEmail(app, event)
	if email == "" {
		log.Printf("[WARN] No email found for event %s rejection notification", event.Id)
		return
	}

	title := event.GetString("title")
	subject := fmt.Sprintf("Event Submission Update: %s", title)

	bodyHTML := fmt.Sprintf(`
<p style="margin:0 0 16px;font-size:1rem;line-height:1.6;">Thank you for submitting an event. Unfortunately, we&#39;re unable to publish <strong>%s</strong> at this time.</p>
<p style="margin:0;font-size:0.875rem;color:#475569;line-height:1.6;">If you have questions or would like to resubmit with changes, please contact the site administrators.</p>`,
		html.EscapeString(title))

	textBody := fmt.Sprintf(`Thank you for submitting an event. Unfortunately, we're unable to publish "%s" at this time.

If you have questions or would like to resubmit with changes, please contact the site administrators.

---
This is an automated notification from %s.`,
		title, appName)

	sendMail(app, email, subject, emailHTML(bodyHTML), textBody, event.Id)
}

func sendModeratorAlert(app core.App, event core.Record, baseURL string) {
	if event.GetString("status") != "pending" {
		return
	}

	moderators, err := app.FindRecordsByFilter("users", "role='admin' || role='editor'", "", 0, 0)
	if err != nil {
		log.Printf("[WARN] Failed to find moderators for event %s: %v", event.Id, err)
		return
	}

	if len(moderators) == 0 {
		log.Printf("[WARN] No moderators found to notify for event %s", event.Id)
		return
	}

	title := event.GetString("title")
	submitterInfo := getSubmitterInfo(app, event)
	startTime := formatEventTime(event.GetString("start_datetime"))
	location := getLocationString(app, event)
	reviewLink := fmt.Sprintf("%s/_/#/collections?collectionId=events&filter=id='%s'", baseURL, event.Id)

	safeTitle := strings.ReplaceAll(strings.ReplaceAll(title, "\n", " "), "\r", " ")
	subject := fmt.Sprintf("New Event Pending Review: %s", safeTitle)

	bodyHTML := fmt.Sprintf(`
<p style="margin:0 0 16px;font-size:1rem;line-height:1.6;">A new event has been submitted and needs review.</p>
<table cellpadding="0" cellspacing="0" style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:6px;padding:16px 20px;margin:0 0 24px;width:100%%;">
  <tr><td style="font-size:0.875rem;color:#475569;padding-bottom:4px;">Event</td></tr>
  <tr><td style="font-size:1rem;font-weight:600;color:#0f172a;padding-bottom:12px;">%s</td></tr>
  <tr><td style="font-size:0.875rem;color:#475569;padding-bottom:4px;">Submitted by</td></tr>
  <tr><td style="font-size:0.875rem;color:#0f172a;padding-bottom:12px;">%s</td></tr>
  <tr><td style="font-size:0.875rem;color:#475569;padding-bottom:4px;">Start</td></tr>
  <tr><td style="font-size:0.875rem;color:#0f172a;padding-bottom:12px;">%s</td></tr>
  <tr><td style="font-size:0.875rem;color:#475569;padding-bottom:4px;">Location</td></tr>
  <tr><td style="font-size:0.875rem;color:#0f172a;">%s</td></tr>
</table>
<a href="%s" style="display:inline-block;background:#0d9488;color:#ffffff;text-decoration:none;font-weight:600;font-size:0.875rem;padding:10px 20px;border-radius:6px;">Review this event</a>`,
		html.EscapeString(title),
		html.EscapeString(submitterInfo),
		html.EscapeString(startTime),
		html.EscapeString(location),
		reviewLink)

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
			HTML:    emailHTML(bodyHTML),
			Text:    textBody,
		}

		if err := mailClient.Send(message); err != nil {
			log.Printf("[WARN] Failed to send moderator alert to %s for event %s: %v", email, event.Id, err)
		}
	}
}

func sendMail(app core.App, to, subject, htmlBody, textBody, eventId string) {
	message := &mailer.Message{
		From: mail.Address{
			Name:    app.Settings().Meta.SenderName,
			Address: app.Settings().Meta.SenderAddress,
		},
		To:      []mail.Address{{Address: to}},
		Subject: subject,
		HTML:    htmlBody,
		Text:    textBody,
	}

	if err := app.NewMailClient().Send(message); err != nil {
		log.Printf("[WARN] Failed to send email to %s for event %s: %v", to, eventId, err)
	}
}
