# Town-Based Location Filters

**Date:** 2026-03-26
**Status:** Approved

## Overview

Add town-based filtering to the home page, allowing users to filter events by the town/city of their associated place. Follows the same inline-filter pattern as the existing date picker — selecting a town filters events in place with a "Show all" button to clear. Composes with the date filter.

## Requirements

- Town filter in the home page sidebar (desktop) and mobile filter bar (mobile)
- Only show towns that have upcoming published events, with event counts
- Clicking a town filters events inline (no navigation away from home)
- Town + date filters are composable (e.g. "Saturday, March 28 in Dublin")
- "Show all" clears the town filter (independent of date filter, same as existing date "Show all" only clears date)
- Mobile uses the same SVG map-pin icon from EventCard, not emoji

## Backend

### New endpoint: `GET /api/towns/counts`

Added to `main.go`, following the same pattern as `/api/tags/counts`.

**Query logic:**
- Join events to places where:
  - `events.status = 'published'`
  - `events.start_datetime >= today`
  - `places.status = 'approved'`
  - `places.city != ''` (non-empty city)
- Group by `places.city`
- Return `[{name: string, count: number}]` sorted by count descending

**Response example:**
```json
[
  {"name": "Dublin", "count": 15},
  {"name": "Cork", "count": 8},
  {"name": "Galway", "count": 6}
]
```

## Frontend

### Home.tsx Changes

**New state:**
- `selectedTown: string | null` — currently selected town filter
- `towns: {name: string, count: number}[]` — town list from API
- `mobilePanel` type extended to `'calendar' | 'tags' | 'towns' | null`

**New effect:**
- Fetch `/api/towns/counts` on mount (same pattern as tag counts fetch)

**Filter logic in `fetchPage`:**
- When `selectedTown` is set, append `place.city = '{selectedTown}'` to the PocketBase filter
- `selectedTown` added as dependency to the reset/reload effect alongside `selectedDate`

**Header text logic:**
- No filters: "Upcoming Events"
- Date only: "Saturday, March 28"
- Town only: "Events in Dublin"
- Both: "Saturday, March 28 in Dublin"

**Clear filter buttons:**
- Each active filter shows its own "Show all" / clear button independently
- Clearing town keeps date filter active and vice versa

**Sidebar:**
- New `sidebar-section` below tags with title "Browse by town"
- Town pills rendered as neutral-colored chips (no tag colors) with event counts
- Click toggles `selectedTown` (same toggle pattern as date)
- Selected town gets active/highlighted styling

**Mobile filter bar:**
- New "Towns" button with SVG map-pin icon (same SVG as `EventCard.tsx .icon-pin`)
- Opens a `mobile-panel` containing the town pills list

### Home.css Changes

- `.town-cloud` — flex wrap layout for town pills (mirrors `.tag-cloud`)
- `.town` pill styles — neutral background, border, hover/active states
- `.town.active` — highlighted state for selected town
- `.mobile-panel-towns` — mobile panel variant for towns

## Files Modified

- `main.go` — new `/api/towns/counts` route handler
- `frontend/src/pages/Home.tsx` — town state, fetch, filter logic, sidebar section, mobile panel
- `frontend/src/pages/Home.css` — town pill and mobile panel styles

## No New Files

All changes are additions to existing files. No new components or pages needed.
