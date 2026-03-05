# Email Notifications Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add email notifications for event moderation workflow (moderator alerts + submitter approval/rejection notifications)

**Architecture:** Hook-based notifications using PocketBase's mail client. Create `internal/hooks/notifications.go` with email helper functions, modify `internal/hooks/events.go` to call notifications on event create/update.

**Tech Stack:** Go, PocketBase SDK, PocketBase mail client

---

## Task 1: Create Email Template Helper Functions

**Files:**
- Create: `internal/hooks/notifications.go`

**Step 1: Create notifications.go with package declaration**

Create `internal/hooks/notifications.go`:

```go
package hooks

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)
```

**Step 2: Add helper function to format datetime**

Add to `internal/hooks/notifications.go`:

```go
// formatEventTime formats a datetime string for email display
func formatEventTime(datetimeStr string) string {
	t, err := time.Parse(time.RFC3339, datetimeStr)
	if err != nil {
		return datetimeStr
	}
	return t.Format("Monday, January 2, 2006 at 3:04 PM MST")
}
```

**Step 3: Add function to get submitter info string**

Add to `internal/hooks/notifications.go`:

```go
// getSubmitterInfo returns a formatted string describing the event submitter
func getSubmitterInfo(app core.App, event core.Record) string {
	// Check for anonymous submission
	authorEmail := event.GetString("author_email")
	if authorEmail != "" {
		return authorEmail
	}

	// Check for registered user
	authorId := event.GetString("author")
	if authorId == "" {
		return "Unknown"
	}

	author, err := app.FindRecordById("users", authorId)
	if err != nil {
		return "Unknown"
	}

	displayName := author.GetString("display_name")
	email := author.GetString("email")

	if displayName != "" {
		return fmt.Sprintf("%s (%s)", displayName, email)
	}
	return email
}
```

**Step 4: Add function to get location string**

Add to `internal/hooks/notifications.go`:

```go
// getLocationString returns a formatted location string for an event
func getLocationString(app core.App, event core.Record) string {
	placeId := event.GetString("place")
	if placeId == "" {
		return "Online/TBD"
	}

	place, err := app.FindRecordById("places", placeId)
	if err != nil {
		return "Online/TBD"
	}

	return place.GetString("name")
}
```

**Step 5: Commit**

```bash
git add internal/hooks/notifications.go
git commit -m "feat: add email notification helper functions"
```

---

## Task 2: Implement Moderator Alert Email Function

**Files:**
- Modify: `internal/hooks/notifications.go`

**Step 1: Add sendModeratorAlert function**

Add to `internal/hooks/notifications.go`:

```go
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
	subject := fmt.Sprintf("New Event Pending Review: %s", title)

	htmlBody := fmt.Sprintf(`<p>A new event has been submitted and needs review.</p>

<p><strong>Event:</strong> %s<br>
<strong>Submitted by:</strong> %s<br>
<strong>Start:</strong> %s<br>
<strong>Location:</strong> %s</p>

<p><a href="%s">Review this event</a></p>

<hr>
<p><small>This is an automated notification from Gather.</small></p>`,
		title, submitterInfo, startTime, location, reviewLink)

	textBody := fmt.Sprintf(`A new event has been submitted and needs review.

Event: %s
Submitted by: %s
Start: %s
Location: %s

Review this event:
%s

---
This is an automated notification from Gather.`,
		title, submitterInfo, startTime, location, reviewLink)

	// Send to each moderator
	mailClient := app.NewMailClient()
	for _, moderator := range moderators {
		email := moderator.GetString("email")
		if email == "" {
			continue
		}

		message := &mailer.Message{
			From: mailClient.From(),
			To:   []string{email},
			Subject: subject,
			HTML: htmlBody,
			Text: textBody,
		}

		if err := mailClient.Send(message); err != nil {
			log.Printf("[WARN] Failed to send moderator alert to %s for event %s: %v", email, event.Id, err)
		}
	}
}
```

**Step 2: Commit**

```bash
git add internal/hooks/notifications.go
git commit -m "feat: add moderator alert email function"
```

---

## Task 3: Implement Approval Notification Email Function

**Files:**
- Modify: `internal/hooks/notifications.go`

**Step 1: Add helper to get submitter email**

Add to `internal/hooks/notifications.go`:

```go
// getSubmitterEmail returns the email address to notify for an event
func getSubmitterEmail(app core.App, event core.Record) string {
	// Check for anonymous submission
	authorEmail := event.GetString("author_email")
	if authorEmail != "" {
		return authorEmail
	}

	// Check for registered user
	authorId := event.GetString("author")
	if authorId == "" {
		return ""
	}

	author, err := app.FindRecordById("users", authorId)
	if err != nil {
		log.Printf("[WARN] Failed to find author %s for event %s: %v", authorId, event.Id, err)
		return ""
	}

	return author.GetString("email")
}
```

**Step 2: Add sendApprovalNotification function**

Add to `internal/hooks/notifications.go`:

```go
// sendApprovalNotification sends email to submitter when event is approved
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

	htmlBody := fmt.Sprintf(`<p>Good news! Your event has been approved and is now live.</p>

<p><strong>Event:</strong> %s<br>
<strong>Start:</strong> %s</p>

<p><a href="%s">View your published event</a></p>

<hr>
<p><small>This is an automated notification from Gather.</small></p>`,
		title, startTime, eventLink)

	textBody := fmt.Sprintf(`Good news! Your event has been approved and is now live.

Event: %s
Start: %s

View your published event:
%s

---
This is an automated notification from Gather.`,
		title, startTime, eventLink)

	mailClient := app.NewMailClient()
	message := &mailer.Message{
		From: mailClient.From(),
		To:   []string{email},
		Subject: subject,
		HTML: htmlBody,
		Text: textBody,
	}

	if err := mailClient.Send(message); err != nil {
		log.Printf("[WARN] Failed to send approval notification to %s for event %s: %v", email, event.Id, err)
	}
}
```

**Step 3: Commit**

```bash
git add internal/hooks/notifications.go
git commit -m "feat: add approval notification email function"
```

---

## Task 4: Implement Rejection Notification Email Function

**Files:**
- Modify: `internal/hooks/notifications.go`

**Step 1: Add sendRejectionNotification function**

Add to `internal/hooks/notifications.go`:

```go
// sendRejectionNotification sends email to submitter when event is rejected
func sendRejectionNotification(app core.App, event core.Record, baseURL string) {
	email := getSubmitterEmail(app, event)
	if email == "" {
		log.Printf("[WARN] No email found for event %s rejection notification", event.Id)
		return
	}

	title := event.GetString("title")

	subject := fmt.Sprintf("Event Submission Update: %s", title)

	htmlBody := fmt.Sprintf(`<p>Thank you for submitting an event. Unfortunately, we're unable to publish "%s" at this time.</p>

<p>If you have questions or would like to resubmit with changes, please contact the site administrators.</p>

<hr>
<p><small>This is an automated notification from Gather.</small></p>`,
		title)

	textBody := fmt.Sprintf(`Thank you for submitting an event. Unfortunately, we're unable to publish "%s" at this time.

If you have questions or would like to resubmit with changes, please contact the site administrators.

---
This is an automated notification from Gather.`,
		title)

	mailClient := app.NewMailClient()
	message := &mailer.Message{
		From: mailClient.From(),
		To:   []string{email},
		Subject: subject,
		HTML: htmlBody,
		Text: textBody,
	}

	if err := mailClient.Send(message); err != nil {
		log.Printf("[WARN] Failed to send rejection notification to %s for event %s: %v", email, event.Id, err)
	}
}
```

**Step 2: Commit**

```bash
git add internal/hooks/notifications.go
git commit -m "feat: add rejection notification email function"
```

---

## Task 5: Integrate Moderator Alerts into Event Hooks

**Files:**
- Modify: `internal/hooks/events.go`

**Step 1: Add OnRecordAfterCreateRequest hook**

Add to `internal/hooks/events.go` after the existing imports, before the delete hook:

```go
	app.OnRecordAfterCreateRequest("events").BindFunc(func(e *core.RecordRequestEvent) error {
		// Send moderator alert for pending events
		sendModeratorAlert(app, e.Record, baseURL)
		return e.Next()
	})
```

**Step 2: Commit**

```bash
git add internal/hooks/events.go
git commit -m "feat: send moderator alerts on pending event creation"
```

---

## Task 6: Integrate Approval/Rejection Notifications into Event Hooks

**Files:**
- Modify: `internal/hooks/events.go`

**Step 1: Add OnRecordAfterUpdateRequest hook**

Add to `internal/hooks/events.go` after the create hook, before the existing update hook:

```go
	app.OnRecordAfterUpdateRequest("events").BindFunc(func(e *core.RecordRequestEvent) error {
		oldStatus := e.Record.Original().GetString("status")
		newStatus := e.Record.GetString("status")

		// Send approval notification
		if oldStatus == "pending" && newStatus == "published" {
			sendApprovalNotification(app, e.Record, baseURL)
		}

		// Send rejection notification
		if oldStatus == "pending" && newStatus == "cancelled" {
			sendRejectionNotification(app, e.Record, baseURL)
		}

		return e.Next()
	})
```

**Step 2: Commit**

```bash
git add internal/hooks/events.go
git commit -m "feat: send submitter notifications on event status changes"
```

---

## Task 7: Build and Test

**Files:**
- N/A (testing only)

**Step 1: Build the application**

```bash
make build-backend
```

Expected: Build succeeds with no errors

**Step 2: Start the development server**

```bash
make dev
```

Expected: Server starts successfully on http://127.0.0.1:8090

**Step 3: Configure SMTP settings**

1. Navigate to http://127.0.0.1:8090/_/
2. Log in with admin@example.com / adminpassword123
3. Go to Settings → Mail settings
4. Configure SMTP (use a test SMTP service like Mailtrap or real SMTP)
5. Save settings

Expected: SMTP settings saved successfully

**Step 4: Test moderator alert (anonymous submission)**

1. Log out from admin dashboard
2. Navigate to http://127.0.0.1:8090/submit
3. Fill out event form:
   - Title: "Test Pending Event"
   - Description: "Testing moderator alerts"
   - Start date/time: Tomorrow at 10:00 AM
   - Email: your-test-email@example.com
4. Submit the event

Expected:
- Event created with status='pending'
- Admin/editor users receive "New Event Pending Review" email
- Check inbox for moderator alert email

**Step 5: Test approval notification**

1. Log in to admin dashboard (http://127.0.0.1:8090/_/)
2. Navigate to Collections → events
3. Find the pending event created in Step 4
4. Edit the event and change status to "published"
5. Save

Expected:
- Event status changed to 'published'
- Submitter (your-test-email@example.com) receives "Your Event Has Been Published" email
- Check inbox for approval email

**Step 6: Test rejection notification**

1. Create another pending event (repeat Step 4 with different title)
2. In admin dashboard, find the new pending event
3. Edit the event and change status to "cancelled"
4. Save

Expected:
- Event status changed to 'cancelled'
- Submitter receives "Event Submission Update" email
- Check inbox for rejection email

**Step 7: Test with registered user**

1. Log in as user (user@example.com / userpassword123)
2. Navigate to http://127.0.0.1:8090/submit
3. Create event (no email field shown for logged-in users)
4. Submit

Expected:
- Event created with status='published' (no moderation for logged-in users)
- No moderator alert sent (status is not 'pending')
- No approval notification sent (didn't go through pending→published transition)

**Step 8: Verify error handling**

In terminal where `make dev` is running, check logs for:
- No errors during email sends
- If SMTP not configured, should see warning logs but app continues
- If no moderators exist, should see warning log but app continues

Expected: Clean logs or expected warnings only

**Step 9: Stop the server**

```bash
# Press Ctrl+C in the terminal running make dev
```

Expected: Server stops cleanly

---

## Task 8: Final Review and Cleanup

**Files:**
- N/A

**Step 1: Review all code changes**

```bash
git diff HEAD~7
```

Expected: Review shows:
- `internal/hooks/notifications.go` created with all email functions
- `internal/hooks/events.go` modified with new hooks
- No unintended changes

**Step 2: Check code builds cleanly**

```bash
go build -o gather
```

Expected: No build errors or warnings

**Step 3: Clean up**

```bash
rm gather
```

**Step 4: Final commit message**

All changes already committed in previous tasks. Verify with:

```bash
git log --oneline -7
```

Expected: Should see 7 commits for this feature

---

## Manual Testing Checklist

After implementation, verify these scenarios:

- [ ] Anonymous event submission (pending) → moderators receive alert
- [ ] Pending event approved → submitter receives approval email
- [ ] Pending event rejected → submitter receives rejection email
- [ ] Logged-in user creates event (published) → no moderator alert
- [ ] Event with missing author_email → logs warning, continues
- [ ] Event with deleted author → logs warning, continues
- [ ] No moderators in system → logs warning, continues
- [ ] SMTP not configured → logs error on first send, continues
- [ ] All emails contain correct links and formatting
- [ ] Plain text and HTML versions both render correctly

## Notes

- Email notifications are synchronous and best-effort (failures logged but don't block)
- Requires SMTP configuration in PocketBase admin settings
- Moderator alerts only fire on event creation (not updates)
- Approval/rejection notifications only fire on pending→published or pending→cancelled transitions
- No emails sent for draft→published or other direct transitions
