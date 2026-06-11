# Editorial Picks Feature Design

**Date:** 2026-06-11

## Overview

Admins and editors can publish curated "picks" posts that spotlight upcoming events — think "Weekend Picks: 14–15 June". Each post has a title, a markdown blurb, and a set of explicitly chosen events. Posts auto-archive once all their featured events have ended, with a manual hidden override. A teaser (title + blurb only) appears on the home page; full posts with event cards live at `/picks` and `/picks/<slug>`.

## Decisions

- **Who can publish:** Admins and editors. No draft/review workflow — save = live.
- **Auto-archiving:** Computed, not stored. A post is "current" if `hidden = false` and at least one featured event has `end_datetime >= now()`. No cron job or status field needed.
- **Manual override:** `hidden` BoolField on the post. Editors can force-archive any post regardless of event dates.
- **Home page:** Shows the single current post — title + blurb only, no event cards.
- **Individual post pages:** `/picks/<slug>` for SEO; OG tags populated from post title, blurb, and first featured event image.
- **Event cards:** Only on `/picks` and `/picks/<slug>`. Each card shows image thumbnail, title, date/time, place name, and links to the event page.

## Data Model

New `picks` PocketBase collection. Migration number: next after `1709300013_settings_umami_host_url.go` — verify at implementation time.

**Fields:**

| Field | Type | Notes |
|---|---|---|
| `title` | TextField | Required. e.g. "Weekend Picks: 14–15 June" |
| `slug` | TextField | Required, unique index. Auto-generated from title (lowercase-hyphenated), user-editable. |
| `blurb` | EditorField | Markdown intro text. |
| `events` | RelationField (multi) | Links to the events collection. |
| `hidden` | BoolField | Default false. Manual override to force-archive. |

**Access rules:**
- ListRule / ViewRule: `""` (public)
- CreateRule / UpdateRule / DeleteRule: `@request.auth.role = "admin" || @request.auth.role = "editor"`

**Migration:** include both an up function (creates collection) and a down/rollback function (deletes collection), consistent with existing migration patterns.

## Backend

No custom routes needed. The PocketBase REST API handles all CRUD. Frontend queries use PocketBase's relation filter syntax and `expand=events` to fetch full event objects inline.

**Example query for current post:**
```
GET /api/collections/picks/records
  ?filter=hidden=false&&events.end_datetime>="<today>"
  &sort=-created
  &perPage=1
  &expand=events
```

**Example query for archive:**
```
GET /api/collections/picks/records
  ?sort=-created
  &expand=events
```

**Creating a post via API:**
```http
POST /api/collections/picks/records
Authorization: Bearer <editor-token>
Content-Type: application/json

{
  "title": "Weekend Picks: 14–15 June",
  "slug": "weekend-picks-14-15-june",
  "blurb": "Here are our top picks for the weekend...",
  "events": ["abc123", "def456"],
  "hidden": false
}
```

## Frontend

### TypeScript interface

Add to `frontend/src/lib/pocketbase.ts`:

```ts
export interface PicksRecord extends BaseModel {
  title: string
  slug: string
  blurb: string
  events?: string[]
  hidden: boolean
  expand?: {
    events?: Event[]
  }
}
```

### New pages

**`frontend/src/pages/Picks.tsx`** — archive page at `/picks`
- Fetches all picks posts sorted by `-created`, expanding events.
- Renders each post: title (linked to `/picks/<slug>`), blurb, then a row of `PicksEventCard` components.
- The current post (first result where any event end_datetime >= today) gets a subtle "This Weekend" badge.

**`frontend/src/pages/PicksPost.tsx`** — individual post at `/picks/:slug`
- Fetches single post by slug (`getFirstListItem`).
- Renders title, full blurb, and all event cards.
- Sets `<title>` and OG meta tags: `og:title` = post title, `og:description` = blurb (plain text, strip markdown), `og:image` = first featured event image URL (if present).
- Shows "Post not found" on missing slug.

### New component

**`frontend/src/components/PicksEventCard.tsx`**
- Props: `event: Event`
- Renders: thumbnail image (via `getImageUrl`), title, formatted start date/time, place name.
- Entire card links to `eventPath(event)`.
- Consistent visual style with existing event cards in the timeline.

### Home page teaser

In `frontend/src/pages/Home.tsx`:
- On mount, fetch the single current picks post (filter as above, perPage=1, no expand needed — just title, blurb, slug).
- If result exists, render a teaser section above the event timeline: post title, blurb, and a "See all picks →" link to `/picks`.
- If no current post, section is hidden (render nothing).

### Routing

In `frontend/src/app.tsx`:
- Add `<Picks path="/picks" />`
- Add `<PicksPost path="/picks/:slug" />`
- Both routes added before the catch-all `<Page path="/:slug" />` so they take priority.

### Admin UI

New "Picks" tab in `frontend/src/pages/Admin.tsx`. Visible when `canModerate()` (editors and admins), unlike Pages which requires `isAdmin()`.

**List view:**
- Table: title, event count, hidden status (toggle), Edit / Delete actions.
- "New Post" button.

**Create/edit form:**
- Title (text input)
- Slug (auto-generated from title, user-editable)
- Blurb (MarkdownEditor component)
- Events (multi-select search: type to search published events by title, same UX pattern as PlaceSearch)
- Hidden checkbox
- Save / Cancel

## SEO

Individual post pages (`/picks/:slug`) use the existing `internal/seo` package pattern:
- `<title>{post.title} | {instance_name}</title>`
- `<meta name="description" content="{blurb with HTML tags stripped, truncated to 160 chars}" />`
- `og:title`, `og:description`, `og:image` (first featured event image URL, or omitted if none)

The `/picks` archive page gets a generic title: `Picks | {instance_name}`.

## Error Handling

- Home page picks fetch failure: silently ignored, no teaser rendered.
- Unknown slug on `/picks/:slug`: "Post not found" message (same pattern as `Page.tsx`).
- Events relation fetch failure on `/picks`: render posts without event cards (graceful degradation).

## Out of Scope

- Scheduling posts for future publish
- Editor-role post ownership (any editor can edit any post)
- RSS/ActivityPub federation of picks posts
- Post ordering within the archive beyond creation date
