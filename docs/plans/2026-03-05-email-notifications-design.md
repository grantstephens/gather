# Email Notifications for Event Approvals - Design

## Overview

Add email notifications to the Gather community calendar for event moderation workflow:
- Alert admins/editors when new events need review
- Notify submitters when events are approved or rejected

**Approach:** Hook-based notifications using PocketBase's mail client, sent synchronously within the request lifecycle.

## Requirements

**Notification Types:**
1. **Moderator Alert** - New event with status='pending' submitted
2. **Approval Notification** - Event status changes from 'pending' to 'published'
3. **Rejection Notification** - Event status changes from 'pending' to 'cancelled'

**Scope:**
- Only event notifications (not places/tags)
- No email validation for anonymous submissions
- No anonymous edit tokens (future enhancement)
- Include action links in all emails

## Architecture

### Hook Integration Points

**New File:** `internal/hooks/notifications.go`
- Helper functions for sending each notification type
- Email template generation
- Recipient resolution logic

**Modified File:** `internal/hooks/events.go`
- Add `OnRecordAfterCreateRequest("events")` hook for moderator alerts
- Extend `OnRecordAfterUpdateRequest("events")` hook for approval/rejection notifications

**No Changes Needed:** `main.go`
- Event hooks already registered via `hooks.RegisterEventHooks()`

### PocketBase Mail Client

Use `app.NewMailClient().Send()` which:
- Respects SMTP settings from PocketBase admin dashboard (Settings → Mail)
- Handles connection pooling and authentication
- Returns error if SMTP not configured or send fails

## Hook Logic

### OnRecordAfterCreateRequest("events")

```
1. Check if record.status == "pending"
2. If yes:
   a. Query users collection: role='admin' OR role='editor'
   b. For each moderator:
      - Send "New Event Needs Review" email
      - Include event details and link to admin dashboard
3. Continue to next hook
```

**Why AfterCreateRequest:** Event is already saved to database, has valid ID for links.

### OnRecordAfterUpdateRequest("events")

```
1. Get oldStatus = record.Original().GetString("status")
2. Get newStatus = record.GetString("status")

3. If oldStatus="pending" AND newStatus="published":
   a. Resolve recipient email (author_email or author.email)
   b. Send "Event Approved" email
   c. Include event details and link to published event

4. If oldStatus="pending" AND newStatus="cancelled":
   a. Resolve recipient email (same logic)
   b. Send "Event Not Approved" email
   c. Include event title and gentle rejection message

5. Continue to existing ActivityPub hooks
```

**Why AfterUpdateRequest:** Event update is committed, published events are live.

## Data Flow

### Moderator Alert Flow

```
User submits event (status='pending')
  ↓
OnRecordAfterCreateRequest hook fires
  ↓
Query users collection for moderators
  ↓
For each moderator:
  - Build email with event details
  - Call app.NewMailClient().Send()
  - Log any errors, don't fail request
  ↓
Hook chain continues
```

### Approval/Rejection Flow

```
Moderator changes event status
  ↓
OnRecordAfterUpdateRequest hook fires
  ↓
Compare old vs new status
  ↓
If pending→published or pending→cancelled:
  - Resolve submitter email
  - Build appropriate email
  - Call app.NewMailClient().Send()
  - Log any errors, don't fail request
  ↓
Hook chain continues (ActivityPub delivery if published)
```

## Email Recipients

### Moderator Alerts

Recipients: All users with `role='admin'` OR `role='editor'`

Query:
```go
moderators, err := app.FindRecordsByFilter("users", "role='admin' || role='editor'", "", 0, 0)
```

For each moderator, extract: `moderator.GetString("email")`

### Submitter Notifications

**Anonymous Events:**
```go
email := record.GetString("author_email")
```

**Registered Events:**
```go
authorId := record.GetString("author")
author, err := app.FindRecordById("users", authorId)
email := author.GetString("email")
displayName := author.GetString("display_name") // optional
```

## Email Templates

### Template 1: Moderator Alert

**Subject:** `New Event Pending Review: {event.title}`

**Body (HTML):**
```html
<p>A new event has been submitted and needs review.</p>

<p><strong>Event:</strong> {event.title}<br>
<strong>Submitted by:</strong> {submitter_info}<br>
<strong>Start:</strong> {formatted_start_datetime}<br>
<strong>Location:</strong> {place.name or "Online/TBD"}</p>

<p><a href="{review_link}">Review this event</a></p>

<hr>
<p><small>This is an automated notification from Gather.</small></p>
```

**Plain Text:**
```
A new event has been submitted and needs review.

Event: {event.title}
Submitted by: {submitter_info}
Start: {formatted_start_datetime}
Location: {place.name or "Online/TBD"}

Review this event:
{review_link}

---
This is an automated notification from Gather.
```

**Links:**
- `review_link`: `{baseURL}/_/#/collections?collectionId=events&filter=id='{event.id}'`

**Submitter Info Display:**
- Anonymous: `{author_email}`
- Registered: `{display_name} ({email})` or just `{email}` if no display_name

### Template 2: Approval Notification

**Subject:** `Your Event Has Been Published: {event.title}`

**Body (HTML):**
```html
<p>Good news! Your event has been approved and is now live.</p>

<p><strong>Event:</strong> {event.title}<br>
<strong>Start:</strong> {formatted_start_datetime}</p>

<p><a href="{event_link}">View your published event</a></p>

<hr>
<p><small>This is an automated notification from Gather.</small></p>
```

**Plain Text:**
```
Good news! Your event has been approved and is now live.

Event: {event.title}
Start: {formatted_start_datetime}

View your published event:
{event_link}

---
This is an automated notification from Gather.
```

**Links:**
- `event_link`: `{baseURL}/event/{event.id}`

### Template 3: Rejection Notification

**Subject:** `Event Submission Update: {event.title}`

**Body (HTML):**
```html
<p>Thank you for submitting an event. Unfortunately, we're unable to publish "{event.title}" at this time.</p>

<p>If you have questions or would like to resubmit with changes, please contact the site administrators.</p>

<hr>
<p><small>This is an automated notification from Gather.</small></p>
```

**Plain Text:**
```
Thank you for submitting an event. Unfortunately, we're unable to publish "{event.title}" at this time.

If you have questions or would like to resubmit with changes, please contact the site administrators.

---
This is an automated notification from Gather.
```

**No Links:** Rejection emails don't include action links.

## Error Handling

### Email Send Failures

**Behavior:**
- Log error with context (event ID, recipient, error message)
- Continue hook processing (don't block request)
- Event operation succeeds regardless of email failure

**Example Log:**
```
[WARN] Failed to send notification: event=abc123, recipient=user@example.com, error=smtp: connection refused
```

### Missing Email Addresses

**Scenarios:**
1. Anonymous event with empty `author_email`: Skip notification, log warning
2. Registered event with no `author` relation: Skip notification, log warning
3. Author user deleted (orphaned event): Skip notification, log warning
4. No moderators found: Skip moderator alert, log warning

**Behavior:** Log clear warning, skip sending, continue processing

### SMTP Not Configured

**Detection:** `app.NewMailClient().Send()` returns error

**Behavior:**
- Log error: `"Email notifications require SMTP configuration in Admin Settings"`
- Continue hook processing
- First email send will reveal configuration issue

### Edge Cases

**Duplicate Notifications:**
- Status changes like pending→published→pending→published trigger multiple notifications
- Accept this behavior - each transition is legitimate
- Moderators only get alerts on initial create (not re-submissions)

**Direct Status Changes:**
- Events created with status='published': No moderator alert (expected)
- Events created with status='draft': No moderator alert (expected)
- Status changes bypassing 'pending' (draft→published): No approval notification (acceptable)

**Orphaned Authors:**
- Registered user submits event, then account is deleted
- Author relation exists but points to deleted record
- Handle gracefully: `app.FindRecordById()` returns error, skip notification

## URL Construction

**Base URL:** Use `baseURL` parameter already passed to `RegisterEventHooks(app, baseURL)`

**Moderator Review Link:**
```
{baseURL}/_/#/collections?collectionId=events&filter=id='{event.id}'
```

**Published Event Link:**
```
{baseURL}/event/{event.id}
```

## Testing Considerations

**Manual Testing:**
1. Configure SMTP in PocketBase admin dashboard
2. Create admin/editor users with valid email addresses
3. Submit anonymous event (status='pending')
   - Verify moderators receive alert email
4. Approve event (change status to 'published')
   - Verify submitter receives approval email
5. Submit another event and reject it (change status to 'cancelled')
   - Verify submitter receives rejection email

**Error Testing:**
1. Submit event with invalid author_email
   - Verify logs show warning, request succeeds
2. Disable SMTP configuration
   - Verify logs show configuration error, request succeeds
3. Submit event with no moderators in system
   - Verify logs show warning, request succeeds

## Future Enhancements

**Out of Scope for Initial Implementation:**
- Place/tag approval notifications
- Anonymous edit token emails
- Email validation/confirmation for anonymous submissions
- Queue-based delivery with retry logic
- Email preferences (opt-out, digest mode)
- Customizable email templates via admin UI
- BCC to site admin on all notifications

These can be added incrementally based on user feedback.
