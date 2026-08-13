# Randomize event order within a day on the home timeline

## Problem

`EventTimeline.tsx` groups events by day and always renders the first event
in the group (earliest `start_datetime`, per the backend sort in
`internal/recurrence/api.go`) as the large "featured" card, with the rest as
small "compact" cards. This means the earliest event of the day always gets
outsized visual attention regardless of relevance.

## Goal

Randomize the order of events within each day's group, so a different event
gets the featured slot. The random order should:

- Stay stable for the duration of a single page load (no reshuffling while
  the user scrolls, paginates, or the SSE subscription refetches page 1).
- Change on a hard page refresh.

## Design

Frontend-only change, scoped to `frontend/src/components/EventTimeline.tsx`.

- When building the day groups, shuffle each day's events (Fisher-Yates)
  instead of leaving them in the backend-provided chronological order.
- Cache the shuffled order per `dateKey` in a `useRef<Map<string, string[]>>`
  (map of dateKey → ordered event IDs) that persists for the component's
  lifetime, i.e. the page load.
- On each recompute of the grouped map (triggered by `events` changing —
  pagination appending more days, or an SSE-triggered refetch of page 1):
  - If a cached order exists for a `dateKey` and its set of event IDs is
    unchanged, reuse the cached order to build that day's array.
  - Otherwise (new day, or the day's event ID set changed — e.g. an event
    was added or cancelled), shuffle freshly and store the new order in the
    cache.
- A hard refresh remounts `EventTimeline`, clearing the ref, so the shuffle
  is freshly randomized.
- No change to the backend sort, to which event is picked as
  `dayEvents[0]` → "featured" (still the first element of the array — that
  element is just no longer guaranteed to be the earliest), or to
  `EventCard` rendering.

## Out of scope

- Backend changes (`internal/recurrence/api.go` sort stays ascending by
  `start_datetime`).
- Weighting the randomization (e.g. by tag, popularity) — plain uniform
  shuffle.
- Persisting the shuffle across page loads (e.g. via URL param or
  localStorage) — refresh intentionally reshuffles.

## Testing

- Unit test for the shuffle/cache logic: given the same `events` array
  reference change (simulating a re-render or refetch) with an unchanged
  event set for a day, the resulting order is unchanged. Given a changed
  event set for a day, the order may change (and the new set is fully
  represented).
- Manual check: reload the home page a few times and confirm the featured
  card varies; scroll to load more pages and confirm already-rendered days
  don't visibly reorder.
