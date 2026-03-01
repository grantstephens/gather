# Gather - Community Calendar Design

A Gancio-equivalent community calendar built with PocketBase and Preact.

## Overview

- **Backend**: PocketBase (Go) with custom hooks
- **Frontend**: Preact SPA embedded in Go binary
- **Database**: SQLite (PocketBase built-in)
- **Places**: OSM/Nominatim-first with manual fallback
- **Auth**: Anonymous + registered, configurable moderation
- **Federation**: Outbound ActivityPub (instance actor)
- **Feeds**: RSS + ICS with path-based filtering
- **Caching**: Short TTLs, stale-while-revalidate
- **Deployment**: Single binary

## Data Model

### events (base collection)

| Field | Type | Notes |
|-------|------|-------|
| title | text | required |
| description | editor | rich text |
| start_datetime | date | required |
| end_datetime | date | optional |
| place | relation → places | |
| online_locations | json | array of URLs |
| tags | relation → tags | multi |
| image | file | thumbnails: 100x100, 400x300, 800x600 |
| author | relation → users | null for anonymous |
| author_email | text | for anonymous submissions, hidden |
| status | select | draft/pending/published/cancelled |
| recurrence_rule | text | RRULE format |
| parent_event | relation → events | for recurrence instances |
| ap_id | text | ActivityPub ID |

### places (base collection)

| Field | Type | Notes |
|-------|------|-------|
| osm_id | number | OpenStreetMap ID |
| osm_type | select | node/way/relation |
| name | text | required |
| address | text | |
| latitude | number | required |
| longitude | number | required |
| city | text | |
| country_code | text | ISO 3166-1 alpha-2 |
| osm_data | json | cached OSM response |

Places are OSM-first: autocomplete queries Nominatim, and we cache/reuse existing places by osm_id. Manual fallback available when location not in OSM.

### tags (base collection)

| Field | Type | Notes |
|-------|------|-------|
| name | text | required, unique |
| color | text | hex color |

### users (auth collection)

| Field | Type | Notes |
|-------|------|-------|
| role | select | user/editor/admin |
| display_name | text | |

### settings (base collection, singleton)

| Field | Type | Notes |
|-------|------|-------|
| instance_name | text | |
| instance_description | text | |
| allow_anonymous | bool | |
| require_moderation | bool | |
| custom_css | text | |
| ap_enabled | bool | |

### ap_followers (base collection)

| Field | Type | Notes |
|-------|------|-------|
| actor_url | text | |
| inbox_url | text | |
| shared_inbox_url | text | |

### ap_delivery_queue (base collection)

| Field | Type | Notes |
|-------|------|-------|
| activity | json | AP activity to deliver |
| inbox_url | text | target inbox |
| attempts | number | retry count |
| last_error | text | |
| next_retry | date | |

## Project Structure

```
gather/
├── main.go                 # PocketBase init, hooks registration
├── go.mod
├── internal/
│   ├── hooks/
│   │   ├── events.go       # Event create/update/delete hooks
│   │   └── moderation.go   # Auto-moderation logic
│   ├── activitypub/
│   │   ├── actor.go        # Instance actor (keypair, profile)
│   │   ├── outbox.go       # Event → AP Note conversion
│   │   ├── inbox.go        # Handle Follow/Undo requests
│   │   ├── delivery.go     # HTTP signatures, POST to inboxes
│   │   └── webfinger.go    # /.well-known/webfinger handler
│   ├── ical/
│   │   └── export.go       # ICS feed generation
│   ├── rss/
│   │   └── export.go       # RSS/Atom feed generation
│   └── recurrence/
│       └── rrule.go        # Expand RRULE into event instances
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   ├── src/
│   │   ├── main.tsx
│   │   ├── app.tsx
│   │   ├── pages/
│   │   │   ├── Home.tsx
│   │   │   ├── Event.tsx
│   │   │   ├── Submit.tsx
│   │   │   ├── Tag.tsx
│   │   │   └── Place.tsx
│   │   ├── components/
│   │   │   ├── Calendar.tsx
│   │   │   ├── EventCard.tsx
│   │   │   ├── PlaceSearch.tsx
│   │   │   ├── TagPicker.tsx
│   │   │   └── RecurrenceEditor.tsx
│   │   └── lib/
│   │       └── pocketbase.ts
│   └── dist/
├── embed.go
└── migrations/
```

## Routes

### Feeds (cacheable)

| Route | Cache-Control |
|-------|---------------|
| `GET /feed.rss` | `public, max-age=300, stale-while-revalidate=60` |
| `GET /feed/tag/:tagname.rss` | same |
| `GET /feed/place/:id.rss` | same |
| `GET /feed.ics` | same |
| `GET /feed/tag/:tagname.ics` | same |
| `GET /event/:id.ics` | same |

### Embeds (cacheable)

| Route | Cache-Control |
|-------|---------------|
| `GET /embed/event/:id.html` | `public, max-age=60, stale-while-revalidate=30` |
| `GET /embed/upcoming.html` | same |

### ActivityPub

| Route | Cache-Control |
|-------|---------------|
| `GET /.well-known/webfinger` | `public, max-age=3600` |
| `GET /.well-known/nodeinfo` | `public, max-age=3600` |
| `GET /ap/actor` | `public, max-age=3600` |
| `GET /ap/outbox` | `public, max-age=300` |
| `POST /ap/inbox` | no-cache |

### API (PocketBase, not cached)

| Route | Notes |
|-------|-------|
| `GET /api/places/search` | Nominatim proxy with caching |
| `/api/collections/*` | PocketBase built-in |

### Frontend (Preact)

| Route | Page |
|-------|------|
| `/` | Home (calendar + list) |
| `/event/:id` | Event detail |
| `/submit` | Event submission |
| `/tag/:name` | Events by tag |
| `/place/:id` | Events at location |
| `/login` | Auth page |
| `/admin/*` | PocketBase admin |

### Static assets

| Route | Cache-Control |
|-------|---------------|
| `/assets/*` | `public, max-age=31536000, immutable` |

## ActivityPub

### Instance Actor

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Application",
  "id": "https://example.com/ap/actor",
  "preferredUsername": "events",
  "name": "Gather - Local Community Events",
  "inbox": "https://example.com/ap/inbox",
  "outbox": "https://example.com/ap/outbox",
  "publicKey": { "...RSA public key..." }
}
```

### Event → Note

Events are published as ActivityPub Notes with structured content:

```json
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "type": "Note",
  "id": "https://example.com/ap/events/abc123",
  "attributedTo": "https://example.com/ap/actor",
  "published": "2026-03-01T10:00:00Z",
  "content": "<p><strong>Event Title</strong></p><p>Date, location, description...</p>",
  "attachment": [{ "type": "Image", "url": "..." }],
  "tag": [{ "type": "Hashtag", "name": "#tag" }],
  "url": "https://example.com/event/abc123"
}
```

### Delivery

- Store followers in `ap_followers` collection
- On event publish: queue `Create{Note}` activities
- On event update: queue `Update{Note}` activities
- On event delete: queue `Delete{Note}` activities
- Worker processes queue with exponential backoff
- Batch by shared inbox when available

## Authentication & Authorization

### Roles

| Role | Capabilities |
|------|-------------|
| Anonymous | Submit events (moderated), view public |
| User | Submit events, edit own |
| Editor | Approve events, edit any, manage places/tags |
| Admin | All + users, settings, CSS |

### Submission Flow

Anonymous or moderation-enabled:
1. Submit form → status: pending
2. Email notification to editors/admins
3. Editor approves → status: published → AP delivery

Registered user (moderation disabled):
1. Submit form → status: published → AP delivery

### Anonymous Edit Token

Anonymous submissions receive a secret edit token via email, allowing editing via `?token=xxx` without login.

### API Rules

```
events.listRule: status = 'published'
events.viewRule: status = 'published' || @request.auth.id != ''
events.createRule: "" (anyone)
events.updateRule: @request.auth.role = 'admin' || @request.auth.role = 'editor' || (@request.auth.id = author && status != 'published')
events.deleteRule: @request.auth.role = 'admin' || @request.auth.role = 'editor'
```

## Frontend

### Stack

| Purpose | Library |
|---------|---------|
| Framework | Preact |
| Routing | preact-router |
| Dates | date-fns |
| Rich text | tiptap or textarea + markdown |
| Map | Leaflet + OSM tiles |
| Calendar | custom grid |
| HTTP | PocketBase JS SDK |

### Pages

- **Home**: Mini calendar + upcoming events list + tag cloud
- **Event**: Hero image, title, datetime, map, description, tags, share/download
- **Submit**: Title, description, datetime picker, recurrence, place search, tags, image upload
- **Tag/Place**: Filtered event list

### Build

1. `cd frontend && npm run build` → `frontend/dist`
2. `go build` embeds dist via `go:embed`
3. `./gather serve` runs everything

## Deferred Features

- Plugin system (can add via Go hooks later)
- Inbound federation / fediverse comments
- Custom static pages
