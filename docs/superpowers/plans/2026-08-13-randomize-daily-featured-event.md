# Randomize Daily Featured Event Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Randomize the order of events within each day's group on the home page timeline, so the "featured" slot isn't always the earliest-starting event, while keeping the order stable for the duration of a page load and reshuffling only on a hard refresh.

**Architecture:** Extract the day-grouping logic out of `EventTimeline.tsx` into a small pure module (`frontend/src/lib/shuffleDayEvents.ts`) that groups events by day and shuffles each day's array (Fisher-Yates), given a mutable per-day-ID-order cache. `EventTimeline.tsx` owns the cache as a `useRef<Map<string, string[]>>` that lives for the component's mount (i.e. the page load) and passes it into the pure function on every render. Because the cache is keyed by day and reused whenever a day's event-ID set is unchanged, re-renders triggered by pagination or the SSE refetch don't reshuffle already-rendered days; a hard refresh remounts the component and clears the cache, producing a fresh shuffle.

**Tech Stack:** Preact, TypeScript, Vite. No frontend test framework currently exists in this repo — this plan adds `vitest` (already compatible with the existing Vite config) as a devDependency solely to unit-test the new pure module.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-08-13-randomize-daily-featured-event-design.md`
- Frontend-only change. No backend (`internal/recurrence/api.go`) changes — its sort stays ascending by `start_datetime`.
- No weighting of the randomization — plain uniform shuffle.
- No persistence of the shuffle across page loads (no URL param, no localStorage) — a refresh must reshuffle.
- `EventCard` rendering and the "first element of the day array is featured" convention in `EventTimeline.tsx` stay unchanged.

---

### Task 1: Pure shuffle/grouping module with tests

**Files:**
- Create: `frontend/src/lib/shuffleDayEvents.ts`
- Test: `frontend/src/lib/shuffleDayEvents.test.ts`
- Modify: `frontend/package.json` (add `vitest` devDependency + `test` script)
- Create: `frontend/vitest.config.ts`

**Interfaces:**
- Produces:
  - `shuffle<T>(array: T[]): T[]` — returns a new array containing the same elements in Fisher-Yates-shuffled order; does not mutate the input.
  - `groupEventsByDayShuffled(events: Event[], cache: Map<string, string[]>): Map<string, Event[]>` — groups `events` by the date portion of `start_datetime` (`YYYY-MM-DD`, same derivation as today: `event.start_datetime?.split(' ')[0] ?? ''`), preserving each day's insertion order as encountered in `events`. For each day: if `cache` already has an entry for that `dateKey` whose ID set (order-independent) exactly matches the current day's event IDs, reuse that cached order; otherwise shuffle the day's events, store the resulting ID order in `cache` (overwriting any stale entry), and use that order. Returns a `Map<dateKey, Event[]>` in the shuffled/cached order. Mutates `cache` in place (adds/overwrites entries); does not remove stale entries for days no longer present in `events`.
  - Both are imported from `../lib/pocketbase` for the `Event` type (`import { Event } from './pocketbase'` inside `shuffleDayEvents.ts`).

- [ ] **Step 1: Install vitest**

```bash
cd frontend && npm install -D vitest
```

- [ ] **Step 2: Add vitest config**

Create `frontend/vitest.config.ts`:

```ts
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'node',
  },
})
```

- [ ] **Step 3: Add test script to package.json**

Modify `frontend/package.json` — add to `"scripts"`:

```json
"test": "vitest run"
```

(Keep existing `dev`, `build`, `preview` scripts as-is; just add this key.)

- [ ] **Step 4: Write the failing tests**

Create `frontend/src/lib/shuffleDayEvents.test.ts`:

```ts
import { describe, it, expect, vi, afterEach } from 'vitest'
import { shuffle, groupEventsByDayShuffled } from './shuffleDayEvents'
import { Event } from './pocketbase'

function makeEvent(id: string, startDatetime: string): Event {
  return {
    id,
    title: `Event ${id}`,
    description: '',
    start_datetime: startDatetime,
    status: 'published',
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('shuffle', () => {
  it('returns a permutation containing the same elements', () => {
    const input = [1, 2, 3, 4, 5]
    const result = shuffle(input)
    expect(result).toHaveLength(input.length)
    expect([...result].sort()).toEqual([...input].sort())
  })

  it('does not mutate the input array', () => {
    const input = [1, 2, 3, 4, 5]
    const copy = [...input]
    shuffle(input)
    expect(input).toEqual(copy)
  })

  it('applies the Fisher-Yates algorithm deterministically given a fixed random source', () => {
    vi.spyOn(Math, 'random').mockReturnValue(0)
    const result = shuffle([1, 2, 3, 4])
    // With Math.random() always 0, each step swaps the current top
    // element to index 0, producing this exact known permutation.
    expect(result).toEqual([2, 3, 4, 1])
  })
})

describe('groupEventsByDayShuffled', () => {
  it('groups events by the date portion of start_datetime', () => {
    const events = [
      makeEvent('a', '2026-08-13 09:00:00.000Z'),
      makeEvent('b', '2026-08-13 18:00:00.000Z'),
      makeEvent('c', '2026-08-14 10:00:00.000Z'),
    ]
    const cache = new Map<string, string[]>()
    const grouped = groupEventsByDayShuffled(events, cache)

    expect([...grouped.keys()]).toEqual(['2026-08-13', '2026-08-14'])
    expect(grouped.get('2026-08-13')!.map(e => e.id).sort()).toEqual(['a', 'b'])
    expect(grouped.get('2026-08-14')!.map(e => e.id)).toEqual(['c'])
  })

  it('reuses the cached order when a day\'s event ID set is unchanged', () => {
    const events = [
      makeEvent('a', '2026-08-13 09:00:00.000Z'),
      makeEvent('b', '2026-08-13 18:00:00.000Z'),
      makeEvent('c', '2026-08-13 12:00:00.000Z'),
    ]
    const cache = new Map<string, string[]>()

    const first = groupEventsByDayShuffled(events, cache)
    const firstOrder = first.get('2026-08-13')!.map(e => e.id)

    // Simulate a refetch: new array/object references, same IDs and dates.
    const refetched = events.map(e => ({ ...e }))
    const second = groupEventsByDayShuffled(refetched, cache)
    const secondOrder = second.get('2026-08-13')!.map(e => e.id)

    expect(secondOrder).toEqual(firstOrder)
  })

  it('reshuffles when a day\'s event ID set changes', () => {
    const events = [
      makeEvent('a', '2026-08-13 09:00:00.000Z'),
      makeEvent('b', '2026-08-13 18:00:00.000Z'),
    ]
    const cache = new Map<string, string[]>()
    cache.set('2026-08-13', ['a', 'b'])

    const withNewEvent = [...events, makeEvent('c', '2026-08-13 12:00:00.000Z')]
    const result = groupEventsByDayShuffled(withNewEvent, cache)

    expect(result.get('2026-08-13')!.map(e => e.id).sort()).toEqual(['a', 'b', 'c'])
    expect(cache.get('2026-08-13')!.sort()).toEqual(['a', 'b', 'c'])
  })
})
```

- [ ] **Step 5: Run tests to verify they fail**

Run: `cd frontend && npx vitest run src/lib/shuffleDayEvents.test.ts`
Expected: FAIL — `shuffleDayEvents.ts` does not exist yet (module not found).

- [ ] **Step 6: Implement the module**

Create `frontend/src/lib/shuffleDayEvents.ts`:

```ts
import { Event } from './pocketbase'

export function shuffle<T>(array: T[]): T[] {
  const result = [...array]
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[result[i], result[j]] = [result[j], result[i]]
  }
  return result
}

export function groupEventsByDayShuffled(
  events: Event[],
  cache: Map<string, string[]>
): Map<string, Event[]> {
  const byDay = new Map<string, Event[]>()
  for (const event of events) {
    const dateKey = event.start_datetime?.split(' ')[0] ?? ''
    if (!byDay.has(dateKey)) byDay.set(dateKey, [])
    byDay.get(dateKey)!.push(event)
  }

  const result = new Map<string, Event[]>()
  for (const [dateKey, dayEvents] of byDay) {
    const byId = new Map(dayEvents.map(e => [e.id, e]))
    const currentIds = [...byId.keys()].sort().join(',')
    const cached = cache.get(dateKey)
    const cachedIds = cached ? [...cached].sort().join(',') : null

    let order: string[]
    if (cached && cachedIds === currentIds) {
      order = cached
    } else {
      order = shuffle(dayEvents).map(e => e.id)
      cache.set(dateKey, order)
    }

    result.set(dateKey, order.map(id => byId.get(id)!))
  }

  return result
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/lib/shuffleDayEvents.test.ts`
Expected: PASS (7 tests)

- [ ] **Step 8: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/vitest.config.ts frontend/src/lib/shuffleDayEvents.ts frontend/src/lib/shuffleDayEvents.test.ts
git commit -m "feat: add per-day event shuffle with stable-per-load cache"
```

---

### Task 2: Wire the shuffle into EventTimeline

**Files:**
- Modify: `frontend/src/components/EventTimeline.tsx:1-28` (imports and the `grouped` useMemo)

**Interfaces:**
- Consumes: `groupEventsByDayShuffled(events: Event[], cache: Map<string, string[]>): Map<string, Event[]>` from `../lib/shuffleDayEvents` (Task 1).

- [ ] **Step 1: Replace the grouping logic**

In `frontend/src/components/EventTimeline.tsx`, change:

```ts
import { useMemo } from 'preact/hooks'
import { format, parseISO } from 'date-fns'
import { Event } from '../lib/pocketbase'
import { EventCard } from './EventCard'
import './EventTimeline.css'
```

to:

```ts
import { useMemo, useRef } from 'preact/hooks'
import { format, parseISO } from 'date-fns'
import { Event } from '../lib/pocketbase'
import { groupEventsByDayShuffled } from '../lib/shuffleDayEvents'
import { EventCard } from './EventCard'
import './EventTimeline.css'
```

and change:

```ts
export function EventTimeline({ events }: Props) {
  const grouped = useMemo(() => {
    const map = new Map<string, Event[]>()
    for (const event of events) {
      const dateKey = event.start_datetime?.split(' ')[0] ?? ''
      if (!map.has(dateKey)) map.set(dateKey, [])
      map.get(dateKey)!.push(event)
    }
    return map
  }, [events])
```

to:

```ts
export function EventTimeline({ events }: Props) {
  const dayOrderCache = useRef(new Map<string, string[]>())
  const grouped = useMemo(
    () => groupEventsByDayShuffled(events, dayOrderCache.current),
    [events]
  )
```

Leave everything else in the file (the `entries`, month-break logic, and the `dayEvents[0]` featured / `dayEvents.slice(1)` compact rendering) untouched — `groupEventsByDayShuffled` returns the same `Map<string, Event[]>` shape the old inline logic did, just with each day's array shuffled and cached instead of left in chronological order.

- [ ] **Step 2: Type-check**

Run: `cd frontend && npx tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Manual verification**

Run: `make dev` (from repo root) to start the backend + Vite dev server, then in a browser:
1. Open the home page. Note which event is featured on a day that has 2+ events.
2. Hard-refresh (Cmd/Ctrl+Shift+R or equivalent) several times. Confirm the featured event for that day changes across refreshes (over enough refreshes — it's random, so an occasional repeat is expected).
3. Without refreshing, scroll down to trigger pagination (load more events). Confirm days already on screen do not visibly reorder.
4. Leave the tab open and, in another tab/window, publish or edit an event so the SSE subscription refetches page 1. Confirm days whose event set didn't change don't reshuffle.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/EventTimeline.tsx
git commit -m "feat: randomize featured event within each day, stable per page load"
```
