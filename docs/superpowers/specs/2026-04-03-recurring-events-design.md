# Recurring Events Design

## Overview

Add recurring event support to Gather with a human-friendly UI that stores RFC 5545 RRULE strings under the hood. Events are expanded virtually at query time by the backend — no materialized instances unless a specific occurrence needs to be cancelled. The frontend treats virtual instances identically to regular events.

## Decisions

- **UI approach:** Common patterns via a friendly picker (daily, weekly, fortnightly, monthly, yearly). No raw RRULE editing. Stored as RRULE for iCal compatibility.
- **Storage:** Hybrid — virtual expansion at query time, with materialization only for exceptions (cancelled occurrences stored as `recurrence_exceptions` on the base event).
- **Editing model:** Edit the whole series only. No per-instance modifications. Cancel a single occurrence by adding it to the exceptions list.
- **End conditions:** Optional. Can repeat forever (capped at 1-year expansion horizon), until a date, or for N occurrences.
- **Expansion location:** Backend. All consumers (web UI, iCal, RSS, AP) get consistent results.

## Data Model

### Existing fields (no changes needed)

- `recurrence_rule` (TextField) — RFC 5545 RRULE string (e.g., `FREQ=WEEKLY;BYDAY=SA`)
- `parent_event` (RelationField) — unused for now, kept for potential future use

### New field

- `recurrence_exceptions` (TextField) — comma-separated ISO dates of cancelled occurrences (e.g., `2026-04-15,2026-05-06`)

## Backend Expansion Engine (`internal/recurrence/`)

Uses a Go RRULE library (e.g., `teambition/rrule-go`) for parsing and expansion.

### Core functions

- `Expand(rule string, dtstart time.Time, rangeStart time.Time, rangeEnd time.Time) []time.Time` — returns all occurrence start times within a date range. Respects `UNTIL` and `COUNT` terminators. For events with no end condition, capped at 1 year from now.
- `BuildRule(freq string, interval int, byDay string, bySetPos int, until *time.Time) string` — generates RRULE strings from structured UI input.

### RRULE mappings from UI options

| UI Option | RRULE |
|-----------|-------|
| Daily | `FREQ=DAILY` |
| Weekly | `FREQ=WEEKLY` |
| Fortnightly | `FREQ=WEEKLY;INTERVAL=2` |
| Monthly (same date) | `FREQ=MONTHLY` |
| Monthly (same weekday, e.g., 3rd Saturday) | `FREQ=MONTHLY;BYDAY=SA;BYSETPOS=3` |
| Yearly | `FREQ=YEARLY` |

### Virtual instance construction

Given a base event record and an occurrence time, produce a virtual event with:
- Shifted `start_datetime` and `end_datetime` (preserving original duration)
- Synthetic ID: `{baseEventId}__{YYYYMMDD}` for frontend linking

## Query Integration

### Event listing (home page, tag pages)

1. Fetch published non-recurring events in the requested date range
2. Fetch published events where `recurrence_rule != ''` and `start_datetime` is before range end
3. Expand each recurring event's RRULE within the range
4. Filter out dates in `recurrence_exceptions`
5. Merge virtual instances with regular events, sort by `start_datetime`
6. Paginate the combined result

### Single event by synthetic ID

When requesting `abc123__20260415`:
1. Parse base event ID (`abc123`) and occurrence date (`2026-04-15`)
2. Fetch base event, verify the date is valid per the RRULE and not in exceptions
3. Return virtual instance with shifted dates

### Pagination

- Backend handles expansion and pagination together
- For "upcoming events" (no fixed end date), expand a rolling 3-month window, extend if page isn't full

## Frontend UI

### Recurrence picker (Submit/Edit forms)

Collapsible "Repeat" section, visible after setting start date:

- **Frequency dropdown:** Does not repeat / Daily / Weekly / Fortnightly / Monthly / Yearly
- **Monthly sub-option:** Toggle between "Same date (e.g., 15th)" and "Same weekday (e.g., 3rd Saturday)" — auto-calculated from start date
- **End condition:** Optional radio buttons: "Never" (default) / "Until [date picker]" / "After [number] occurrences"

Picker builds structured object, converted to RRULE string before submission.

### Event display

- Event detail page shows human-readable recurrence label (e.g., "Repeats weekly", "Repeats monthly on the 3rd Saturday")
- "Next occurrences" list showing upcoming dates (next 5) with links to virtual instances
- Home page: recurring instances look identical to regular events

### Editing and cancelling

- Edit always modifies the whole series. Virtual instance edit links go to the base event's edit page.
- "Cancel this occurrence" button on virtual instance pages — adds the date to `recurrence_exceptions`

## Feed Integration

### iCal (`/feed.ics`)

- Recurring events output a single VEVENT with `RRULE` property and `EXDATE` entries from `recurrence_exceptions`
- Calendar clients handle their own expansion — native RRULE support

### RSS (`/feed.rss`)

- Show only the next upcoming occurrence of each recurring event as a regular item
- Include "Repeats weekly" (or equivalent) in the description

### ActivityPub

- Federate the next upcoming occurrence as a Note
- When that occurrence passes, the next one takes its place

## Edge Cases & Constraints

- **Timezone:** All datetimes stored/expanded in UTC. Frontend handles display timezone.
- **Past occurrences:** Generated when viewing past events, but home page default is future-only.
- **Editing start date:** Shifts all future occurrences. Stale exceptions silently ignored.
- **Deleting recurring event:** Deletes base event, all virtual instances vanish.
- **Moderation:** Approving a recurring event approves the whole series.
- **Max expansion horizon:** 1 year for never-ending series.
- **Admin panel:** Recurring events show once (base event) with a "recurring" badge, not expanded.
