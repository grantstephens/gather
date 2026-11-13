# Gather API Specification

**Version:** 1.0
**Base URL:** `http://your-instance.com`
**API Version:** PocketBase 0.36.5

## Table of Contents

1. [Authentication](#authentication)
2. [Quick Start](#quick-start)
3. [Events API](#events-api)
4. [Places API](#places-api)
5. [Tags API](#tags-api)
6. [Feed Endpoints](#feed-endpoints)
7. [Error Handling](#error-handling)
8. [Rate Limiting](#rate-limiting)
9. [Best Practices](#best-practices)

---

## Authentication

Gather uses PocketBase's built-in authentication system with JWT tokens.

### Login

```http
POST /api/collections/users/auth-with-password
Content-Type: application/json

{
  "identity": "user@example.com",
  "password": "your_password"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "record": {
    "id": "user_id_123",
    "email": "user@example.com",
    "role": "user",
    "display_name": "John Doe"
  }
}
```

### Using the Token

Include the token in the `Authorization` header for authenticated requests:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### User Roles

- **user** (default): Can create events that are published immediately
- **editor**: Can moderate, edit, and delete any event
- **admin**: Full access including settings and user management

---

## Quick Start

### Creating Your First Event

**For authenticated users (recommended):**

```bash
# 1. Login
TOKEN=$(curl -s -X POST http://localhost:8090/api/collections/users/auth-with-password \
  -H "Content-Type: application/json" \
  -d '{"identity":"user@example.com","password":"password"}' \
  | jq -r '.token')

# 2. Create event
curl -X POST http://localhost:8090/api/collections/events/records \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Community Meetup" \
  -F "description=Join us for a community gathering!" \
  -F "start_datetime=2025-04-15T18:00:00Z" \
  -F "end_datetime=2025-04-15T20:00:00Z" \
  -F "status=published"
```

**For anonymous users:**

```bash
curl -X POST http://localhost:8090/api/collections/events/records \
  -F "title=Open Event" \
  -F "description=Everyone is welcome" \
  -F "start_datetime=2025-04-15T18:00:00Z" \
  -F "author_email=organizer@example.com" \
  -F "status=pending"
```

---

## Events API

### Event Object

```typescript
interface Event {
  id: string;                    // Auto-generated
  title: string;                 // Required, max 200 chars
  description?: string;          // HTML/Markdown supported
  start_datetime: string;        // Required, ISO 8601 format
  end_datetime?: string;         // ISO 8601 format
  place?: string;                // Place ID (relation)
  online_locations?: object;     // Virtual event details
  tags?: string[];               // Array of tag IDs
  image?: File;                  // Max 5MB, auto-converted to WebP
  author?: string;               // User ID (auto-set if authenticated)
  author_email?: string;         // Required for anonymous submissions
  status: 'draft' | 'pending' | 'published' | 'cancelled';
  recurrence_rule?: string;      // RRULE format (RFC 5545)
  ap_id?: string;                // ActivityPub ID (auto-generated)
  parent_event?: string;         // For recurring event instances
  created: string;               // Auto-generated timestamp
  updated: string;               // Auto-generated timestamp
}
```

### Create Event

```http
POST /api/collections/events/records
Content-Type: multipart/form-data
Authorization: Bearer {token}  # Optional for anonymous
```

**Form Fields:**

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `title` | string | Yes | Max 200 characters |
| `description` | string | No | Supports HTML/Markdown |
| `start_datetime` | string | Yes | ISO 8601 (e.g., `2025-04-15T18:00:00Z`) |
| `end_datetime` | string | No | ISO 8601 |
| `place` | string | No | Place ID from `/api/collections/places/records` |
| `tags` | string[] | No | Array of tag IDs |
| `image` | File | No | Max 5MB, JPEG/PNG/WebP/GIF |
| `author_email` | string | Conditional | Required if not authenticated |
| `status` | string | No | `published` (auth users) or `pending` (anonymous) |
| `recurrence_rule` | string | No | iCalendar RRULE format |

**Example Request:**

```bash
curl -X POST http://localhost:8090/api/collections/events/records \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Weekly Yoga Class" \
  -F "description=Beginner-friendly yoga session" \
  -F "start_datetime=2025-04-20T10:00:00Z" \
  -F "end_datetime=2025-04-20T11:00:00Z" \
  -F "place=place_123" \
  -F "tags=tag_456" \
  -F "tags=tag_789" \
  -F "status=published" \
  -F "recurrence_rule=FREQ=WEEKLY;BYDAY=SA" \
  -F "image=@yoga.jpg"
```

**Response (201 Created):**

```json
{
  "id": "event_abc123",
  "title": "Weekly Yoga Class",
  "description": "Beginner-friendly yoga session",
  "start_datetime": "2025-04-20T10:00:00Z",
  "end_datetime": "2025-04-20T11:00:00Z",
  "place": "place_123",
  "tags": ["tag_456", "tag_789"],
  "image": "event_abc123_yoga_12345.webp",
  "author": "user_xyz",
  "status": "published",
  "recurrence_rule": "FREQ=WEEKLY;BYDAY=SA",
  "created": "2025-03-17T12:00:00.000Z",
  "updated": "2025-03-17T12:00:00.000Z"
}
```

### List Events

```http
GET /api/collections/events/records?filter={filter}&expand={expand}
```

**Query Parameters:**

| Parameter | Type | Example | Notes |
|-----------|------|---------|-------|
| `filter` | string | `status='published'` | PocketBase filter syntax |
| `expand` | string | `place,tags,author` | Expand related records |
| `page` | number | `1` | Page number (default: 1) |
| `perPage` | number | `20` | Items per page (max: 200) |
| `sort` | string | `-start_datetime` | Sort order (prefix `-` for desc) |

**Example Requests:**

```bash
# Get all published events
curl "http://localhost:8090/api/collections/events/records?filter=status%3D%27published%27"

# Get upcoming events with details
curl "http://localhost:8090/api/collections/events/records?filter=status%3D%27published%27%26%26start_datetime%3E%3D%272025-03-17%27&expand=place,tags&sort=start_datetime"

# Get events at a specific place
curl "http://localhost:8090/api/collections/events/records?filter=place%3D%27place_123%27"

# Get events with a specific tag
curl "http://localhost:8090/api/collections/events/records?filter=tags.id%3D%27tag_456%27"
```

**Response (200 OK):**

```json
{
  "page": 1,
  "perPage": 20,
  "totalPages": 5,
  "totalItems": 87,
  "items": [
    {
      "id": "event_abc123",
      "title": "Weekly Yoga Class",
      "start_datetime": "2025-04-20T10:00:00Z",
      "status": "published",
      "expand": {
        "place": {
          "id": "place_123",
          "name": "Community Center",
          "address": "123 Main St",
          "latitude": 40.7128,
          "longitude": -74.0060
        },
        "tags": [
          {"id": "tag_456", "name": "fitness", "color": "#3B82F6"},
          {"id": "tag_789", "name": "wellness", "color": "#10B981"}
        ]
      }
    }
  ]
}
```

### Get Single Event

```http
GET /api/collections/events/records/{id}?expand={expand}
```

**Example:**

```bash
curl "http://localhost:8090/api/collections/events/records/event_abc123?expand=place,tags,author"
```

### Update Event

```http
PATCH /api/collections/events/records/{id}
Content-Type: multipart/form-data
Authorization: Bearer {token}  # Required
```

**Permissions:**
- Admins/editors: Can update any event
- Authors: Can only update their own draft/pending events (not published)

**Example:**

```bash
curl -X PATCH http://localhost:8090/api/collections/events/records/event_abc123 \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Updated Event Title" \
  -F "description=Updated description"
```

### Delete Event

```http
DELETE /api/collections/events/records/{id}
Authorization: Bearer {token}  # Required (admin/editor only)
```

**Example:**

```bash
curl -X DELETE http://localhost:8090/api/collections/events/records/event_abc123 \
  -H "Authorization: Bearer $TOKEN"
```

### Event Status Workflow

```
Anonymous Submission:
  pending → (admin approves) → published
         → (admin rejects) → cancelled

Authenticated Submission:
  published → (can be cancelled) → cancelled
           → (can be republished) → published
```

**Important Constraints:**

1. Events cannot be published if their associated place has `status='pending'`
2. Events cannot be published if any associated tag has `status='pending'`
3. Status transitions require proper permissions

---

## Places API

### Place Object

```typescript
interface Place {
  id: string;
  osm_id?: number;               // OpenStreetMap ID
  osm_type?: 'node' | 'way' | 'relation';
  name: string;                  // Required
  address?: string;
  latitude: number;              // Required
  longitude: number;             // Required
  city?: string;
  country_code?: string;         // ISO 3166-1 alpha-2
  osm_data?: object;             // Full OSM metadata
  status: 'pending' | 'approved';
  created: string;
  updated: string;
}
```

### Create Place

```http
POST /api/collections/places/records
Content-Type: application/json
Authorization: Bearer {token}  # Optional
```

**Request Body:**

```json
{
  "name": "Community Center",
  "address": "123 Main Street",
  "latitude": 40.7128,
  "longitude": -74.0060,
  "city": "New York",
  "country_code": "US",
  "osm_id": 123456789,
  "osm_type": "node"
}
```

**Response (201 Created):**

```json
{
  "id": "place_abc123",
  "name": "Community Center",
  "address": "123 Main Street",
  "latitude": 40.7128,
  "longitude": -74.0060,
  "city": "New York",
  "country_code": "US",
  "status": "approved",  // "approved" for admins, "pending" for users
  "created": "2025-03-17T12:00:00.000Z",
  "updated": "2025-03-17T12:00:00.000Z"
}
```

**Note:** Places created by non-admin users have `status='pending'` and require approval before events can reference them in published state.

### List Places

```http
GET /api/collections/places/records?filter=status='approved'
```

**Example:**

```bash
# Get all approved places
curl "http://localhost:8090/api/collections/places/records?filter=status%3D%27approved%27"

# Search places by name
curl "http://localhost:8090/api/collections/places/records?filter=name~%27community%27"

# Get places in a city
curl "http://localhost:8090/api/collections/places/records?filter=city%3D%27New%20York%27"
```

---

## Tags API

### Tag Object

```typescript
interface Tag {
  id: string;
  name: string;                  // Required, unique
  color?: string;                // Hex color code
  status: 'pending' | 'approved';
  created: string;
  updated: string;
}
```

### Create Tag

```http
POST /api/collections/tags/records
Content-Type: application/json
Authorization: Bearer {token}  # Optional
```

**Request Body:**

```json
{
  "name": "wellness",
  "color": "#10B981"
}
```

**Response (201 Created):**

```json
{
  "id": "tag_abc123",
  "name": "wellness",
  "color": "#10B981",
  "status": "approved",  // "approved" for admins, "pending" for users
  "created": "2025-03-17T12:00:00.000Z",
  "updated": "2025-03-17T12:00:00.000Z"
}
```

### List Tags

```http
GET /api/collections/tags/records?filter=status='approved'
```

**Example:**

```bash
# Get all approved tags
curl "http://localhost:8090/api/collections/tags/records?filter=status%3D%27approved%27"

# Search tags by name
curl "http://localhost:8090/api/collections/tags/records?filter=name~%27fit%27"
```

---

## Feed Endpoints

Gather provides RSS and iCalendar feeds for syndication.

### RSS Feeds

```http
GET /feed.rss                      # All published events
GET /feed/tag/{tagname}            # Events with specific tag
```

**Example:**

```bash
curl http://localhost:8090/feed.rss
curl http://localhost:8090/feed/tag/wellness
```

### iCalendar Feeds

```http
GET /feed.ics                      # All published events
GET /ics/tag/{tagname}             # Events with specific tag
GET /ics/event/{id}                # Single event
```

**Example:**

```bash
curl http://localhost:8090/feed.ics
curl http://localhost:8090/ics/tag/wellness
curl http://localhost:8090/ics/event/event_abc123
```

**Note:** All feeds are cached for 5 minutes with stale-while-revalidate.

---

## Error Handling

### Error Response Format

```json
{
  "code": 400,
  "message": "Failed to create record.",
  "data": {
    "title": {
      "code": "validation_required",
      "message": "Missing required value."
    }
  }
}
```

### Common Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | `validation_*` | Validation error (missing field, invalid format) |
| 401 | `unauthorized` | Missing or invalid authentication token |
| 403 | `forbidden` | Insufficient permissions |
| 404 | `not_found` | Resource not found |
| 500 | `internal_error` | Server error |

### Validation Errors

**Event cannot be published if dependencies are pending:**

```json
{
  "code": 400,
  "message": "cannot publish event: the associated place is still pending approval"
}
```

```json
{
  "code": 400,
  "message": "cannot publish event: one or more tags are still pending approval"
}
```

---

## Rate Limiting

Gather enforces rate limiting by default on sensitive endpoints:

- **Authentication** (`/api/collections/users/auth-with-password`) — rate limited to prevent brute-force attacks
- **Event submission** — rate limited for anonymous submissions

For additional protection at the network level, you can also configure rate limiting in your reverse proxy (nginx, Caddy).

---

## Best Practices

### 1. Use Authenticated Requests When Possible

Authenticated users can publish events immediately, while anonymous submissions require moderation.

```bash
# Login once, reuse token
TOKEN=$(curl -s -X POST http://localhost:8090/api/collections/users/auth-with-password \
  -H "Content-Type: application/json" \
  -d '{"identity":"api@example.com","password":"$STRONG_PASSWORD"}' \
  | jq -r '.token')

# Reuse $TOKEN for multiple requests
```

### 2. Create Places and Tags First

Before creating events, ensure places and tags exist and are approved:

```bash
# 1. Create place
PLACE_ID=$(curl -s -X POST http://localhost:8090/api/collections/places/records \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Community Center","latitude":40.7128,"longitude":-74.0060}' \
  | jq -r '.id')

# 2. Create tag
TAG_ID=$(curl -s -X POST http://localhost:8090/api/collections/tags/records \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"community"}' \
  | jq -r '.id')

# 3. Create event with references
curl -X POST http://localhost:8090/api/collections/events/records \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Meetup" \
  -F "start_datetime=2025-04-20T18:00:00Z" \
  -F "place=$PLACE_ID" \
  -F "tags=$TAG_ID" \
  -F "status=published"
```

### 3. Use Proper Date Formatting

Always use ISO 8601 format with timezone:

```javascript
// Good
const startTime = new Date('2025-04-20T18:00:00Z').toISOString();

// Bad
const startTime = '2025-04-20 18:00:00';  // Missing timezone
```

### 4. Expand Relations for Efficiency

Use the `expand` parameter to fetch related data in a single request:

```bash
# Efficient: One request with expanded data
curl "http://localhost:8090/api/collections/events/records/event_123?expand=place,tags,author"

# Inefficient: Multiple requests
curl "http://localhost:8090/api/collections/events/records/event_123"
curl "http://localhost:8090/api/collections/places/records/place_123"
curl "http://localhost:8090/api/collections/tags/records/tag_456"
```

### 5. Filter Published Events

Always filter for published events in public-facing integrations:

```bash
curl "http://localhost:8090/api/collections/events/records?filter=status%3D%27published%27"
```

### 6. Handle Image Uploads

Images are automatically converted to WebP and thumbnailed:

```bash
# Upload image with event
curl -X POST http://localhost:8090/api/collections/events/records \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Photo Event" \
  -F "start_datetime=2025-04-20T18:00:00Z" \
  -F "image=@event-photo.jpg" \
  -F "status=published"

# Access image variants
# Original: /api/files/events/{recordId}/{filename}
# Thumbnail: /api/files/events/{recordId}/{filename}?thumb=400x300
```

Available thumbnail sizes: `100x100`, `400x300`, `800x600`

### 7. Use Recurring Events

For repeating events, use RRULE format:

```bash
# Weekly event every Saturday
curl -X POST http://localhost:8090/api/collections/events/records \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Weekly Market" \
  -F "start_datetime=2025-04-19T09:00:00Z" \
  -F "end_datetime=2025-04-19T13:00:00Z" \
  -F "recurrence_rule=FREQ=WEEKLY;BYDAY=SA" \
  -F "status=published"

# Monthly event on the first Monday
curl -X POST http://localhost:8090/api/collections/events/records \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Monthly Meeting" \
  -F "start_datetime=2025-04-07T19:00:00Z" \
  -F "recurrence_rule=FREQ=MONTHLY;BYDAY=1MO" \
  -F "status=published"
```

### 8. Error Handling in Code

Always handle API errors gracefully:

```javascript
async function createEvent(eventData) {
  try {
    const formData = new FormData();
    Object.entries(eventData).forEach(([key, value]) => {
      formData.append(key, value);
    });

    const response = await fetch('http://localhost:8090/api/collections/events/records', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`
      },
      body: formData
    });

    if (!response.ok) {
      const error = await response.json();
      console.error('Failed to create event:', error.message);
      return null;
    }

    return await response.json();
  } catch (err) {
    console.error('Network error:', err);
    return null;
  }
}
```

---

## Integration Examples

### Node.js/JavaScript

```javascript
import PocketBase from 'pocketbase';

const pb = new PocketBase('http://localhost:8090');

// Authenticate
await pb.collection('users').authWithPassword('user@example.com', 'password');

// Create event
const event = await pb.collection('events').create({
  title: 'Tech Meetup',
  description: 'Discussing the latest trends',
  start_datetime: '2025-04-25T18:00:00Z',
  end_datetime: '2025-04-25T20:00:00Z',
  status: 'published'
});

// List upcoming events
const events = await pb.collection('events').getList(1, 50, {
  filter: 'status="published" && start_datetime>="2025-03-17"',
  expand: 'place,tags',
  sort: 'start_datetime'
});
```

### Python

```python
import requests
from datetime import datetime

BASE_URL = 'http://localhost:8090'

# Authenticate
auth_response = requests.post(f'{BASE_URL}/api/collections/users/auth-with-password', json={
    'identity': 'user@example.com',
    'password': 'password'
})
token = auth_response.json()['token']

headers = {'Authorization': f'Bearer {token}'}

# Create event
event_data = {
    'title': 'Python Workshop',
    'description': 'Learn Python basics',
    'start_datetime': '2025-04-30T14:00:00Z',
    'status': 'published'
}

response = requests.post(
    f'{BASE_URL}/api/collections/events/records',
    headers=headers,
    data=event_data
)

event = response.json()
print(f"Created event: {event['id']}")
```

### PHP

```php
<?php
$baseUrl = 'http://localhost:8090';

// Authenticate
$authResponse = file_get_contents($baseUrl . '/api/collections/users/auth-with-password', false, stream_context_create([
    'http' => [
        'method' => 'POST',
        'header' => 'Content-Type: application/json',
        'content' => json_encode([
            'identity' => 'user@example.com',
            'password' => 'password'
        ])
    ]
]));

$authData = json_decode($authResponse, true);
$token = $authData['token'];

// Create event
$eventData = [
    'title' => 'PHP Conference',
    'description' => 'Annual PHP developers meetup',
    'start_datetime' => '2025-05-10T09:00:00Z',
    'status' => 'published'
];

$response = file_get_contents($baseUrl . '/api/collections/events/records', false, stream_context_create([
    'http' => [
        'method' => 'POST',
        'header' => "Authorization: Bearer $token\r\nContent-Type: application/json",
        'content' => json_encode($eventData)
    ]
]));

$event = json_decode($response, true);
echo "Created event: " . $event['id'];
?>
```

---

## ActivityPub Federation

Gather publishes events over ActivityPub to the Fediverse (Mastodon and similar) and to Mobilizon.

### Actors and endpoints

| Actor | Type | Handle (WebFinger) | URLs |
|---|---|---|---|
| Instance | `Service` | `events@host` | `GET /ap/actor`, `GET /ap/outbox`, `GET /ap/actor/followers`, `POST /ap/inbox` |
| Relay | `Application` | `relay@host` | `GET /ap/relay`, `POST /ap/relay/inbox` |
| Venue | `Group` | `{place-slug}@host` | `GET /ap/actor/place/{placeId}` and `/followers`, `/outbox`, `POST /inbox`, `GET /events/{eventId}` under it |
| Town | `Group` | `town-{city-slug}@host` | `GET /ap/actor/town/{city-slug}` and the same sub-paths |

Other URLs: `GET /.well-known/webfinger?resource=acct:{handle}@host`, and `GET /ap/events/{eventId}`, the canonical object for an event.

Inbox requests must carry a valid HTTP Signature (draft-cavage, `rsa-sha256`); the signing key is fetched from the sender's actor, PKCS#1 and PKIX public keys are both accepted, and the `host` header is checked against the request's `Host`.

`ap_enabled` in settings currently controls only the weekly picks announcement. Routes and event delivery do not check it.

### Who gets what

Three follower lists, each with its own delivery shape:

| Followers of | Stored in | Receive |
|---|---|---|
| Instance actor | `ap_followers` | A `Note` (`Create`, `Update`, `Delete`), as before |
| Relay | `ap_relay_followers` | The `Event` shape: `Create` then `Announce` (10 s later), `Update`, `Delete` |
| A venue or town | `ap_subject_followers` (`subject` is `place:{id}` or `town:{city-slug}`) | The `Event` shape, as that group's own copy: `Create`, `Update`, `Delete` |

A remote inbox that follows both a venue and its town gets one copy, the venue's. Shared inboxes are used when the follower has one, so an inbox is never sent the same activity twice.

**Why a relay.** Mobilizon lists a remote event in Explore only if a `Share` exists for it from an actor the Mobilizon instance follows. The relay's `Announce` creates that `Share`. A plain `Create` is stored but not listed.

**Why group-owned copies.** Mobilizon attaches an event to a group only when the Event has an organizer `actor` that differs from `attributedTo`. A venue or town therefore serves its own copy of each event at `/ap/actor/{kind}/{ref}/events/{eventId}` with `attributedTo` = the group and `actor` = the instance actor; the instance actor sends the `Create`. These ids are separate from `/ap/events/{eventId}`.

**Lazy actors.** Venue and town actors are created on first request. The first `GET` of the actor generates an RSA keypair (stored in `ap_actors`); names that do not resolve never create a key. A venue resolves only when it is `approved`; a town resolves when at least one approved venue has that city. Each actor signs its own deliveries (`keyId` is `{actor}#main-key`). The relay signs with the instance key.

### Status changes

The delivery hooks run on the `events` collection. Notes go to instance followers; Events go to relay, venue and town followers.

| Change | Instance followers (Note) | Relay, venue, town followers (Event) |
|---|---|---|
| Created as `published`, or `draft/pending` → `published` | `Create` | `Create` (+ `Announce` for the relay) |
| `published` → `published` (edit) | `Update` | `Update` |
| `published` → `cancelled` | `Delete` | `Update` with `status: CANCELLED` |
| `published` → `draft` / `pending` | `Delete` | `Delete` |
| `cancelled` → `published` | `Create` | `Update` |
| `cancelled` → `draft` / `pending`, or deleting a cancelled event | nothing | `Delete` |
| Deleting a `published` event | `Delete` | `Delete` |

A cancelled event is sent as an update, not a delete, because `/ap/events/{id}` still serves cancelled events and Mobilizon re-fetches the object on `Delete`. Nothing is sent when an event simply becomes past, and events published before a follow are not backfilled.

### Event object

`GET /ap/events/{eventId}` returns the Event for `published` and `cancelled` events. For anything else (draft, pending, missing) it returns `410 Gone` with a `Tombstone` that reveals nothing about the event. Group copies behave the same way, and also return `410` when the event is not at that venue or in that town.

| Field | Source |
|---|---|
| `name` | Event title (plain text) |
| `content` | Description with the scraper `Cost:` and `Source:` lines removed, the recurrence label, and a link to the Gather event page |
| `startTime`, `endTime` | RFC 3339 in the `TZ` zone, with offset. `endTime` omitted when unset |
| `timezone` | The `TZ` zone name (default `Europe/London` when unset or invalid) |
| `status` | `CONFIRMED`, or `CANCELLED` for cancelled events |
| `joinMode`, `externalParticipationUrl` | `external`, pointing at the Gather event page |
| `location` | The event's place, only if `approved`: name, address, and latitude/longitude as JSON floats (Mobilizon ignores integers) |
| `tag` | Approved tags only, as `#name` hashtags with spaces removed; tags over 40 characters are skipped |
| `attachment` | The event image, always as `.webp`, with `name: "Banner"` (Mobilizon fails without a name) |

The JSON-LD context uses Mobilizon's namespace `https://mobilizon.org/ns#`. A real Mobilizon-produced Event is kept in `internal/activitypub/testdata/mobilizon_event.json` and a test compares our output's keys and types against it.

### Delivery

Activities are queued in `ap_delivery_queue` and sent by a worker that runs every 5 minutes (`POST /api/ap/retry-queue` runs it immediately, for admins). Failures retry with exponential backoff, up to 5 attempts, then are dropped.

### Collections

`ap_followers`, `ap_relay_followers`, `ap_subject_followers`, `ap_actors` (keypairs for venue and town actors), `ap_delivery_queue`.

---

## Support

For issues, questions, or feature requests:

- **GitHub Issues:** https://github.com/grantstephens/gather/issues
- **Documentation:** See README.md and DEPLOY.md in the repository
- **Admin Dashboard:** Access at `http://your-instance.com/_/` for database inspection

---

## Changelog

### Unreleased
- Mobilizon federation: `Event` objects at `/ap/events/{id}`, relay actor `relay@host`, lazily-created venue and town `Group` actors, `TZ` used for event times
- Fix: inbox signature verification now accepts PKCS#1 public keys and checks the `host` header against the request `Host`

### Version 1.0 (2025-03-17)
- Initial API specification
- Events, Places, Tags CRUD operations
- Authentication with PocketBase
- RSS and iCalendar feed endpoints
- ActivityPub federation support
- Image upload with automatic WebP conversion
- Moderation workflow documentation
