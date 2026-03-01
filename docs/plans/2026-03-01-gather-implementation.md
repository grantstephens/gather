# Gather Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Gancio-equivalent community calendar with PocketBase backend and Preact frontend, supporting anonymous/registered event submission, moderation, ActivityPub federation, and RSS/ICS feeds.

**Architecture:** PocketBase serves as both API and embedded SQLite database. Custom Go code handles ActivityPub, feeds, and business logic via hooks. Preact SPA is embedded in the Go binary and served as static files.

**Tech Stack:** Go 1.23+, PocketBase, Preact, Vite, TypeScript, Leaflet, date-fns, PocketBase JS SDK

---

## Phase 1: Project Foundation

### Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `main.go`

**Step 1: Initialize Go module**

Run:
```bash
go mod init gather
```

Expected: Creates `go.mod` with `module gather`

**Step 2: Create minimal main.go**

Create `main.go`:
```go
package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
)

func main() {
	app := pocketbase.New()

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
```

**Step 3: Download dependencies**

Run:
```bash
go mod tidy
```

Expected: Downloads pocketbase, creates `go.sum`

**Step 4: Verify it runs**

Run:
```bash
go run main.go serve
```

Expected: PocketBase starts on http://127.0.0.1:8090, admin UI accessible at `/_/`

**Step 5: Commit**

```bash
git add go.mod go.sum main.go
git commit -m "feat: initialize PocketBase Go project"
```

---

### Task 2: Create Collection Migrations

**Files:**
- Create: `migrations/1709300000_collections.go`

**Step 1: Create migrations directory**

Run:
```bash
mkdir -p migrations
```

**Step 2: Create collections migration**

Create `migrations/1709300000_collections.go`:
```go
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Tags collection
		tags := core.NewBaseCollection("tags")
		tags.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})
		tags.Fields.Add(&core.TextField{
			Name: "color",
		})
		tags.Indexes = []string{
			"CREATE UNIQUE INDEX idx_tags_name ON tags (name)",
		}
		tags.ListRule = nil // public
		tags.ViewRule = nil
		if err := app.Save(tags); err != nil {
			return err
		}

		// Places collection
		places := core.NewBaseCollection("places")
		places.Fields.Add(&core.NumberField{
			Name: "osm_id",
		})
		places.Fields.Add(&core.SelectField{
			Name:   "osm_type",
			Values: []string{"node", "way", "relation"},
		})
		places.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})
		places.Fields.Add(&core.TextField{
			Name: "address",
		})
		places.Fields.Add(&core.NumberField{
			Name:     "latitude",
			Required: true,
		})
		places.Fields.Add(&core.NumberField{
			Name:     "longitude",
			Required: true,
		})
		places.Fields.Add(&core.TextField{
			Name: "city",
		})
		places.Fields.Add(&core.TextField{
			Name: "country_code",
			Max:  2,
		})
		places.Fields.Add(&core.JSONField{
			Name: "osm_data",
		})
		places.ListRule = nil
		places.ViewRule = nil
		if err := app.Save(places); err != nil {
			return err
		}

		// Events collection
		events := core.NewBaseCollection("events")
		events.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
			Max:      200,
		})
		events.Fields.Add(&core.EditorField{
			Name: "description",
		})
		events.Fields.Add(&core.DateField{
			Name:     "start_datetime",
			Required: true,
		})
		events.Fields.Add(&core.DateField{
			Name: "end_datetime",
		})
		events.Fields.Add(&core.RelationField{
			Name:         "place",
			CollectionId: places.Id,
			MaxSelect:    1,
		})
		events.Fields.Add(&core.JSONField{
			Name: "online_locations",
		})
		events.Fields.Add(&core.RelationField{
			Name:         "tags",
			CollectionId: tags.Id,
			MaxSelect:    99,
		})
		events.Fields.Add(&core.FileField{
			Name:      "image",
			MaxSelect: 1,
			MaxSize:   5 * 1024 * 1024,
			MimeTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif"},
			Thumbs:    []string{"100x100", "400x300", "800x600"},
		})
		events.Fields.Add(&core.TextField{
			Name: "author_email",
		})
		events.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"draft", "pending", "published", "cancelled"},
		})
		events.Fields.Add(&core.TextField{
			Name: "recurrence_rule",
		})
		events.Fields.Add(&core.RelationField{
			Name:         "parent_event",
			CollectionId: "", // self-reference, set after creation
			MaxSelect:    1,
		})
		events.Fields.Add(&core.TextField{
			Name: "ap_id",
		})
		events.Fields.Add(&core.TextField{
			Name: "edit_token",
		})
		// API rules
		publishedRule := "status = 'published'"
		events.ListRule = &publishedRule
		viewRule := "status = 'published'"
		events.ViewRule = &viewRule
		createRule := ""
		events.CreateRule = &createRule
		if err := app.Save(events); err != nil {
			return err
		}

		// Add author relation (needs users collection ID)
		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err == nil {
			events.Fields.Add(&core.RelationField{
				Name:         "author",
				CollectionId: usersCollection.Id,
				MaxSelect:    1,
			})
			// Update parent_event to self-reference
			for i, f := range events.Fields {
				if rf, ok := f.(*core.RelationField); ok && rf.Name == "parent_event" {
					rf.CollectionId = events.Id
					events.Fields[i] = rf
				}
			}
			if err := app.Save(events); err != nil {
				return err
			}
		}

		// Settings collection (singleton)
		settings := core.NewBaseCollection("settings")
		settings.Fields.Add(&core.TextField{
			Name: "instance_name",
		})
		settings.Fields.Add(&core.TextField{
			Name: "instance_description",
		})
		settings.Fields.Add(&core.BoolField{
			Name: "allow_anonymous",
		})
		settings.Fields.Add(&core.BoolField{
			Name: "require_moderation",
		})
		settings.Fields.Add(&core.TextField{
			Name: "custom_css",
		})
		settings.Fields.Add(&core.BoolField{
			Name: "ap_enabled",
		})
		settings.Fields.Add(&core.TextField{
			Name: "ap_private_key",
		})
		settings.Fields.Add(&core.TextField{
			Name: "ap_public_key",
		})
		// Only admins can access settings
		settings.ListRule = nil
		settings.ViewRule = nil
		if err := app.Save(settings); err != nil {
			return err
		}

		// AP Followers collection
		apFollowers := core.NewBaseCollection("ap_followers")
		apFollowers.Fields.Add(&core.TextField{
			Name:     "actor_url",
			Required: true,
		})
		apFollowers.Fields.Add(&core.TextField{
			Name:     "inbox_url",
			Required: true,
		})
		apFollowers.Fields.Add(&core.TextField{
			Name: "shared_inbox_url",
		})
		if err := app.Save(apFollowers); err != nil {
			return err
		}

		// AP Delivery Queue collection
		apQueue := core.NewBaseCollection("ap_delivery_queue")
		apQueue.Fields.Add(&core.JSONField{
			Name:     "activity",
			Required: true,
		})
		apQueue.Fields.Add(&core.TextField{
			Name:     "inbox_url",
			Required: true,
		})
		apQueue.Fields.Add(&core.NumberField{
			Name: "attempts",
		})
		apQueue.Fields.Add(&core.TextField{
			Name: "last_error",
		})
		apQueue.Fields.Add(&core.DateField{
			Name: "next_retry",
		})
		if err := app.Save(apQueue); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback
		collections := []string{"ap_delivery_queue", "ap_followers", "settings", "events", "places", "tags"}
		for _, name := range collections {
			col, err := app.FindCollectionByNameOrId(name)
			if err == nil {
				if err := app.Delete(col); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
```

**Step 3: Register migrations in main.go**

Update `main.go`:
```go
package main

import (
	"log"

	"github.com/pocketbase/pocketbase"

	_ "gather/migrations"
)

func main() {
	app := pocketbase.New()

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
```

**Step 4: Run migrations**

Run:
```bash
go run main.go migrate
```

Expected: "Applied X migration(s)"

**Step 5: Verify collections exist**

Run:
```bash
go run main.go serve
```

Then visit `http://127.0.0.1:8090/_/` and check Collections panel.

Expected: See tags, places, events, settings, ap_followers, ap_delivery_queue collections

**Step 6: Commit**

```bash
git add migrations/ main.go
git commit -m "feat: add PocketBase collection migrations"
```

---

### Task 3: Add User Role Field

**Files:**
- Create: `migrations/1709300001_user_role.go`

**Step 1: Create user role migration**

Create `migrations/1709300001_user_role.go`:
```go
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.Fields.Add(&core.SelectField{
			Name:     "role",
			Values:   []string{"user", "editor", "admin"},
		})
		users.Fields.Add(&core.TextField{
			Name: "display_name",
		})

		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.Fields.RemoveByName("role")
		users.Fields.RemoveByName("display_name")

		return app.Save(users)
	})
}
```

**Step 2: Run migration**

Run:
```bash
go run main.go migrate
```

Expected: Migration applied

**Step 3: Commit**

```bash
git add migrations/1709300001_user_role.go
git commit -m "feat: add role and display_name fields to users"
```

---

### Task 4: Create Project Directory Structure

**Files:**
- Create: `internal/hooks/events.go`
- Create: `internal/activitypub/actor.go`
- Create: `internal/ical/export.go`
- Create: `internal/rss/export.go`
- Create: `internal/recurrence/rrule.go`
- Create: `embed.go`

**Step 1: Create directory structure**

Run:
```bash
mkdir -p internal/hooks internal/activitypub internal/ical internal/rss internal/recurrence
```

**Step 2: Create placeholder files**

Create `internal/hooks/events.go`:
```go
package hooks

import (
	"github.com/pocketbase/pocketbase/core"
)

func RegisterEventHooks(app core.App) {
	// Event hooks will be registered here
}
```

Create `internal/activitypub/actor.go`:
```go
package activitypub

// ActivityPub actor implementation
```

Create `internal/ical/export.go`:
```go
package ical

// ICS feed generation
```

Create `internal/rss/export.go`:
```go
package rss

// RSS feed generation
```

Create `internal/recurrence/rrule.go`:
```go
package recurrence

// RRULE expansion
```

Create `embed.go`:
```go
package main

import "embed"

//go:embed frontend/dist/*
var frontendFS embed.FS
```

**Step 3: Commit**

```bash
git add internal/ embed.go
git commit -m "feat: create project directory structure"
```

---

## Phase 2: Frontend Setup

### Task 5: Initialize Preact Project

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/index.html`
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/app.tsx`

**Step 1: Create frontend directory**

Run:
```bash
mkdir -p frontend/src
```

**Step 2: Create package.json**

Create `frontend/package.json`:
```json
{
  "name": "gather-frontend",
  "private": true,
  "version": "0.0.1",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "preact": "^10.19.3",
    "preact-router": "^4.1.2",
    "pocketbase": "^0.21.1",
    "date-fns": "^3.3.1",
    "leaflet": "^1.9.4"
  },
  "devDependencies": {
    "@preact/preset-vite": "^2.8.1",
    "@types/leaflet": "^1.9.8",
    "typescript": "^5.3.3",
    "vite": "^5.1.0"
  }
}
```

**Step 3: Create vite.config.ts**

Create `frontend/vite.config.ts`:
```typescript
import { defineConfig } from 'vite'
import preact from '@preact/preset-vite'

export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8090',
      '/_': 'http://127.0.0.1:8090',
    },
  },
})
```

**Step 4: Create tsconfig.json**

Create `frontend/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "module": "ESNext",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "jsxImportSource": "preact",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["src"]
}
```

**Step 5: Create index.html**

Create `frontend/index.html`:
```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Gather</title>
    <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

**Step 6: Create main.tsx**

Create `frontend/src/main.tsx`:
```tsx
import { render } from 'preact'
import { App } from './app'
import './style.css'

render(<App />, document.getElementById('app')!)
```

**Step 7: Create app.tsx**

Create `frontend/src/app.tsx`:
```tsx
import Router from 'preact-router'

export function App() {
  return (
    <div class="app">
      <header>
        <h1>Gather</h1>
      </header>
      <main>
        <Router>
          <Home path="/" />
        </Router>
      </main>
    </div>
  )
}

function Home() {
  return (
    <div>
      <h2>Upcoming Events</h2>
      <p>Welcome to Gather!</p>
    </div>
  )
}
```

**Step 8: Create style.css**

Create `frontend/src/style.css`:
```css
:root {
  --color-primary: #2563eb;
  --color-text: #1f2937;
  --color-bg: #ffffff;
  --color-border: #e5e7eb;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: system-ui, -apple-system, sans-serif;
  color: var(--color-text);
  background: var(--color-bg);
  line-height: 1.5;
}

.app {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1rem;
}

header {
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 1rem;
  margin-bottom: 2rem;
}

header h1 {
  font-size: 1.5rem;
}
```

**Step 9: Install dependencies**

Run:
```bash
cd frontend && npm install
```

Expected: node_modules created, package-lock.json generated

**Step 10: Verify dev server works**

Run:
```bash
cd frontend && npm run dev
```

Expected: Vite dev server starts, app visible at http://localhost:5173

**Step 11: Build for production**

Run:
```bash
cd frontend && npm run build
```

Expected: `frontend/dist` directory created with built files

**Step 12: Add frontend to gitignore**

Create `.gitignore`:
```
# Dependencies
frontend/node_modules/

# Build output
frontend/dist/

# PocketBase data
pb_data/

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
```

**Step 13: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/vite.config.ts frontend/tsconfig.json frontend/index.html frontend/src/ .gitignore
git commit -m "feat: initialize Preact frontend with Vite"
```

---

### Task 6: Create PocketBase Client

**Files:**
- Create: `frontend/src/lib/pocketbase.ts`

**Step 1: Create lib directory**

Run:
```bash
mkdir -p frontend/src/lib
```

**Step 2: Create PocketBase client**

Create `frontend/src/lib/pocketbase.ts`:
```typescript
import PocketBase from 'pocketbase'

export const pb = new PocketBase('/')

// Types for our collections
export interface Event {
  id: string
  title: string
  description: string
  start_datetime: string
  end_datetime?: string
  place?: string
  online_locations?: string[]
  tags?: string[]
  image?: string
  author?: string
  author_email?: string
  status: 'draft' | 'pending' | 'published' | 'cancelled'
  recurrence_rule?: string
  parent_event?: string
  ap_id?: string
  created: string
  updated: string
  expand?: {
    place?: Place
    tags?: Tag[]
    author?: User
  }
}

export interface Place {
  id: string
  osm_id?: number
  osm_type?: 'node' | 'way' | 'relation'
  name: string
  address?: string
  latitude: number
  longitude: number
  city?: string
  country_code?: string
  osm_data?: Record<string, unknown>
}

export interface Tag {
  id: string
  name: string
  color?: string
}

export interface User {
  id: string
  email: string
  role?: 'user' | 'editor' | 'admin'
  display_name?: string
}

export interface Settings {
  id: string
  instance_name?: string
  instance_description?: string
  allow_anonymous?: boolean
  require_moderation?: boolean
  custom_css?: string
  ap_enabled?: boolean
}

// Helper to get image URL
export function getImageUrl(record: Event, thumb?: string): string | undefined {
  if (!record.image) return undefined
  return pb.files.getURL(record, record.image, { thumb })
}
```

**Step 3: Commit**

```bash
git add frontend/src/lib/pocketbase.ts
git commit -m "feat: add PocketBase client with types"
```

---

## Phase 3: Core Frontend Pages

### Task 7: Create Event Card Component

**Files:**
- Create: `frontend/src/components/EventCard.tsx`
- Create: `frontend/src/components/EventCard.css`

**Step 1: Create components directory**

Run:
```bash
mkdir -p frontend/src/components
```

**Step 2: Create EventCard component**

Create `frontend/src/components/EventCard.tsx`:
```tsx
import { format } from 'date-fns'
import { Event, getImageUrl } from '../lib/pocketbase'
import './EventCard.css'

interface Props {
  event: Event
}

export function EventCard({ event }: Props) {
  const startDate = new Date(event.start_datetime)
  const imageUrl = getImageUrl(event, '400x300')

  return (
    <a href={`/event/${event.id}`} class="event-card">
      {imageUrl && (
        <div class="event-card-image">
          <img src={imageUrl} alt="" />
        </div>
      )}
      <div class="event-card-content">
        <time class="event-card-date">
          {format(startDate, 'EEE, MMM d · h:mm a')}
        </time>
        <h3 class="event-card-title">{event.title}</h3>
        {event.expand?.place && (
          <div class="event-card-place">
            {event.expand.place.name}
          </div>
        )}
        {event.expand?.tags && event.expand.tags.length > 0 && (
          <div class="event-card-tags">
            {event.expand.tags.map(tag => (
              <span
                key={tag.id}
                class="tag"
                style={tag.color ? { backgroundColor: tag.color } : undefined}
              >
                {tag.name}
              </span>
            ))}
          </div>
        )}
      </div>
    </a>
  )
}
```

**Step 3: Create EventCard styles**

Create `frontend/src/components/EventCard.css`:
```css
.event-card {
  display: block;
  text-decoration: none;
  color: inherit;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
  transition: box-shadow 0.2s;
}

.event-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.event-card-image {
  aspect-ratio: 4/3;
  overflow: hidden;
}

.event-card-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.event-card-content {
  padding: 1rem;
}

.event-card-date {
  font-size: 0.875rem;
  color: var(--color-primary);
  font-weight: 500;
}

.event-card-title {
  font-size: 1.125rem;
  margin: 0.25rem 0;
}

.event-card-place {
  font-size: 0.875rem;
  color: #6b7280;
}

.event-card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.tag {
  font-size: 0.75rem;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  background: #f3f4f6;
  color: #374151;
}
```

**Step 4: Commit**

```bash
git add frontend/src/components/
git commit -m "feat: add EventCard component"
```

---

### Task 8: Create Home Page

**Files:**
- Create: `frontend/src/pages/Home.tsx`
- Create: `frontend/src/pages/Home.css`
- Modify: `frontend/src/app.tsx`

**Step 1: Create pages directory**

Run:
```bash
mkdir -p frontend/src/pages
```

**Step 2: Create Home page**

Create `frontend/src/pages/Home.tsx`:
```tsx
import { useEffect, useState } from 'preact/hooks'
import { pb, Event, Tag } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import './Home.css'

export function Home() {
  const [events, setEvents] = useState<Event[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function load() {
      try {
        const [eventsResult, tagsResult] = await Promise.all([
          pb.collection('events').getList<Event>(1, 20, {
            filter: `status = 'published' && start_datetime >= '${new Date().toISOString()}'`,
            sort: 'start_datetime',
            expand: 'place,tags',
          }),
          pb.collection('tags').getFullList<Tag>(),
        ])
        setEvents(eventsResult.items)
        setTags(tagsResult)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load events')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  if (loading) {
    return <div class="loading">Loading events...</div>
  }

  if (error) {
    return <div class="error">{error}</div>
  }

  return (
    <div class="home">
      <div class="home-main">
        <h2>Upcoming Events</h2>
        {events.length === 0 ? (
          <p class="no-events">No upcoming events</p>
        ) : (
          <div class="events-grid">
            {events.map(event => (
              <EventCard key={event.id} event={event} />
            ))}
          </div>
        )}
      </div>
      <aside class="home-sidebar">
        <div class="sidebar-section">
          <h3>Tags</h3>
          <div class="tag-cloud">
            {tags.map(tag => (
              <a
                key={tag.id}
                href={`/tag/${tag.name}`}
                class="tag"
                style={tag.color ? { backgroundColor: tag.color } : undefined}
              >
                {tag.name}
              </a>
            ))}
          </div>
        </div>
        <div class="sidebar-section">
          <a href="/submit" class="btn btn-primary">
            + Add Event
          </a>
        </div>
      </aside>
    </div>
  )
}
```

**Step 3: Create Home styles**

Create `frontend/src/pages/Home.css`:
```css
.home {
  display: grid;
  grid-template-columns: 1fr 280px;
  gap: 2rem;
}

@media (max-width: 768px) {
  .home {
    grid-template-columns: 1fr;
  }
  .home-sidebar {
    order: -1;
  }
}

.home-main h2 {
  margin-bottom: 1.5rem;
}

.events-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
}

.no-events {
  color: #6b7280;
  text-align: center;
  padding: 3rem;
}

.home-sidebar {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.sidebar-section h3 {
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #6b7280;
  margin-bottom: 0.75rem;
}

.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.tag-cloud .tag {
  text-decoration: none;
}

.btn {
  display: inline-block;
  padding: 0.75rem 1.5rem;
  border-radius: 6px;
  text-decoration: none;
  font-weight: 500;
  text-align: center;
  cursor: pointer;
  border: none;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-primary:hover {
  background: #1d4ed8;
}

.loading, .error {
  text-align: center;
  padding: 3rem;
  color: #6b7280;
}

.error {
  color: #dc2626;
}
```

**Step 4: Update app.tsx**

Replace `frontend/src/app.tsx`:
```tsx
import Router from 'preact-router'
import { Home } from './pages/Home'
import './style.css'

export function App() {
  return (
    <div class="app">
      <header>
        <a href="/" class="logo">Gather</a>
      </header>
      <main>
        <Router>
          <Home path="/" />
        </Router>
      </main>
    </div>
  )
}
```

**Step 5: Update style.css**

Update `frontend/src/style.css` to include:
```css
:root {
  --color-primary: #2563eb;
  --color-text: #1f2937;
  --color-bg: #ffffff;
  --color-border: #e5e7eb;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: system-ui, -apple-system, sans-serif;
  color: var(--color-text);
  background: var(--color-bg);
  line-height: 1.5;
}

.app {
  max-width: 1200px;
  margin: 0 auto;
  padding: 1rem;
}

header {
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 1rem;
  margin-bottom: 2rem;
}

.logo {
  font-size: 1.5rem;
  font-weight: bold;
  text-decoration: none;
  color: inherit;
}
```

**Step 6: Commit**

```bash
git add frontend/src/pages/ frontend/src/app.tsx frontend/src/style.css
git commit -m "feat: add Home page with event listing"
```

---

### Task 9: Create Event Detail Page

**Files:**
- Create: `frontend/src/pages/Event.tsx`
- Create: `frontend/src/pages/Event.css`
- Modify: `frontend/src/app.tsx`

**Step 1: Create Event page**

Create `frontend/src/pages/Event.tsx`:
```tsx
import { useEffect, useState, useRef } from 'preact/hooks'
import { format } from 'date-fns'
import L from 'leaflet'
import { pb, Event as EventType, getImageUrl } from '../lib/pocketbase'
import './Event.css'

interface Props {
  id: string
}

export function Event({ id }: Props) {
  const [event, setEvent] = useState<EventType | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const mapRef = useRef<HTMLDivElement>(null)
  const mapInstance = useRef<L.Map | null>(null)

  useEffect(() => {
    async function load() {
      try {
        const result = await pb.collection('events').getOne<EventType>(id, {
          expand: 'place,tags,author',
        })
        setEvent(result)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Event not found')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [id])

  useEffect(() => {
    if (!event?.expand?.place || !mapRef.current || mapInstance.current) return

    const place = event.expand.place
    const map = L.map(mapRef.current).setView([place.latitude, place.longitude], 15)
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; OpenStreetMap contributors',
    }).addTo(map)
    L.marker([place.latitude, place.longitude]).addTo(map)
    mapInstance.current = map

    return () => {
      map.remove()
      mapInstance.current = null
    }
  }, [event])

  if (loading) {
    return <div class="loading">Loading...</div>
  }

  if (error || !event) {
    return <div class="error">{error || 'Event not found'}</div>
  }

  const startDate = new Date(event.start_datetime)
  const endDate = event.end_datetime ? new Date(event.end_datetime) : null
  const imageUrl = getImageUrl(event, '800x600')

  return (
    <article class="event-page">
      {imageUrl && (
        <div class="event-hero">
          <img src={imageUrl} alt="" />
        </div>
      )}
      <div class="event-content">
        <header class="event-header">
          <time class="event-datetime">
            {format(startDate, 'EEEE, MMMM d, yyyy · h:mm a')}
            {endDate && ` - ${format(endDate, 'h:mm a')}`}
          </time>
          <h1>{event.title}</h1>
          {event.expand?.tags && event.expand.tags.length > 0 && (
            <div class="event-tags">
              {event.expand.tags.map(tag => (
                <a
                  key={tag.id}
                  href={`/tag/${tag.name}`}
                  class="tag"
                  style={tag.color ? { backgroundColor: tag.color } : undefined}
                >
                  {tag.name}
                </a>
              ))}
            </div>
          )}
        </header>

        {event.expand?.place && (
          <section class="event-location">
            <h2>Location</h2>
            <p class="place-name">{event.expand.place.name}</p>
            {event.expand.place.address && (
              <p class="place-address">{event.expand.place.address}</p>
            )}
            <div ref={mapRef} class="event-map" />
          </section>
        )}

        {event.description && (
          <section class="event-description">
            <h2>About</h2>
            <div dangerouslySetInnerHTML={{ __html: event.description }} />
          </section>
        )}

        <footer class="event-actions">
          <a href={`/event/${event.id}.ics`} class="btn">
            Download .ics
          </a>
        </footer>
      </div>
    </article>
  )
}
```

**Step 2: Create Event styles**

Create `frontend/src/pages/Event.css`:
```css
.event-page {
  max-width: 800px;
  margin: 0 auto;
}

.event-hero {
  aspect-ratio: 16/9;
  overflow: hidden;
  border-radius: 8px;
  margin-bottom: 2rem;
}

.event-hero img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.event-header {
  margin-bottom: 2rem;
}

.event-datetime {
  display: block;
  font-size: 1rem;
  color: var(--color-primary);
  font-weight: 500;
  margin-bottom: 0.5rem;
}

.event-header h1 {
  font-size: 2rem;
  margin-bottom: 1rem;
}

.event-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.event-tags .tag {
  text-decoration: none;
}

.event-location,
.event-description {
  margin-bottom: 2rem;
}

.event-location h2,
.event-description h2 {
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #6b7280;
  margin-bottom: 0.75rem;
}

.place-name {
  font-weight: 600;
}

.place-address {
  color: #6b7280;
  margin-bottom: 1rem;
}

.event-map {
  height: 300px;
  border-radius: 8px;
  overflow: hidden;
}

.event-description {
  line-height: 1.7;
}

.event-description p {
  margin-bottom: 1rem;
}

.event-actions {
  padding-top: 2rem;
  border-top: 1px solid var(--color-border);
}
```

**Step 3: Update app.tsx**

Update `frontend/src/app.tsx`:
```tsx
import Router from 'preact-router'
import { Home } from './pages/Home'
import { Event } from './pages/Event'
import './style.css'

export function App() {
  return (
    <div class="app">
      <header>
        <a href="/" class="logo">Gather</a>
      </header>
      <main>
        <Router>
          <Home path="/" />
          <Event path="/event/:id" />
        </Router>
      </main>
    </div>
  )
}
```

**Step 4: Commit**

```bash
git add frontend/src/pages/Event.tsx frontend/src/pages/Event.css frontend/src/app.tsx
git commit -m "feat: add Event detail page with map"
```

---

### Task 10: Create Submit Page

**Files:**
- Create: `frontend/src/pages/Submit.tsx`
- Create: `frontend/src/pages/Submit.css`
- Create: `frontend/src/components/PlaceSearch.tsx`
- Create: `frontend/src/components/TagPicker.tsx`
- Modify: `frontend/src/app.tsx`

**Step 1: Create PlaceSearch component**

Create `frontend/src/components/PlaceSearch.tsx`:
```tsx
import { useState, useCallback } from 'preact/hooks'
import { Place, pb } from '../lib/pocketbase'

interface Props {
  value: Place | null
  onChange: (place: Place | null) => void
}

interface NominatimResult {
  place_id: number
  osm_id: number
  osm_type: string
  display_name: string
  lat: string
  lon: string
  address?: {
    city?: string
    town?: string
    village?: string
    country_code?: string
  }
}

export function PlaceSearch({ value, onChange }: Props) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<NominatimResult[]>([])
  const [loading, setLoading] = useState(false)
  const [showManual, setShowManual] = useState(false)

  const search = useCallback(async (q: string) => {
    if (q.length < 3) {
      setResults([])
      return
    }
    setLoading(true)
    try {
      const response = await fetch(
        `https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(q)}&addressdetails=1&limit=5`,
        { headers: { 'User-Agent': 'Gather Community Calendar' } }
      )
      const data = await response.json()
      setResults(data)
    } catch {
      setResults([])
    } finally {
      setLoading(false)
    }
  }, [])

  const selectResult = async (result: NominatimResult) => {
    // Check if place already exists
    const osmType = result.osm_type === 'node' ? 'node' : result.osm_type === 'way' ? 'way' : 'relation'
    try {
      const existing = await pb.collection('places').getFirstListItem<Place>(
        `osm_id = ${result.osm_id} && osm_type = '${osmType}'`
      )
      onChange(existing)
    } catch {
      // Create new place
      const city = result.address?.city || result.address?.town || result.address?.village
      const newPlace: Partial<Place> = {
        osm_id: result.osm_id,
        osm_type: osmType as 'node' | 'way' | 'relation',
        name: result.display_name.split(',')[0],
        address: result.display_name,
        latitude: parseFloat(result.lat),
        longitude: parseFloat(result.lon),
        city,
        country_code: result.address?.country_code?.toUpperCase(),
        osm_data: result as unknown as Record<string, unknown>,
      }
      const created = await pb.collection('places').create<Place>(newPlace)
      onChange(created)
    }
    setQuery('')
    setResults([])
  }

  if (value) {
    return (
      <div class="place-selected">
        <span>{value.name}</span>
        <button type="button" onClick={() => onChange(null)}>×</button>
      </div>
    )
  }

  if (showManual) {
    return (
      <ManualPlaceForm
        onSubmit={onChange}
        onCancel={() => setShowManual(false)}
      />
    )
  }

  return (
    <div class="place-search">
      <input
        type="text"
        value={query}
        onInput={(e) => {
          const val = (e.target as HTMLInputElement).value
          setQuery(val)
          search(val)
        }}
        placeholder="Search for a location..."
      />
      {loading && <div class="place-loading">Searching...</div>}
      {results.length > 0 && (
        <ul class="place-results">
          {results.map((r) => (
            <li key={r.place_id}>
              <button type="button" onClick={() => selectResult(r)}>
                {r.display_name}
              </button>
            </li>
          ))}
        </ul>
      )}
      <button type="button" class="link" onClick={() => setShowManual(true)}>
        Can't find your location? Add manually
      </button>
    </div>
  )
}

function ManualPlaceForm({ onSubmit, onCancel }: { onSubmit: (p: Place) => void; onCancel: () => void }) {
  const [name, setName] = useState('')
  const [address, setAddress] = useState('')
  const [lat, setLat] = useState('')
  const [lon, setLon] = useState('')

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    const place = await pb.collection('places').create<Place>({
      name,
      address,
      latitude: parseFloat(lat),
      longitude: parseFloat(lon),
    })
    onSubmit(place)
  }

  return (
    <form onSubmit={handleSubmit} class="manual-place-form">
      <input
        type="text"
        value={name}
        onInput={(e) => setName((e.target as HTMLInputElement).value)}
        placeholder="Place name"
        required
      />
      <input
        type="text"
        value={address}
        onInput={(e) => setAddress((e.target as HTMLInputElement).value)}
        placeholder="Address"
      />
      <div class="coords">
        <input
          type="number"
          step="any"
          value={lat}
          onInput={(e) => setLat((e.target as HTMLInputElement).value)}
          placeholder="Latitude"
          required
        />
        <input
          type="number"
          step="any"
          value={lon}
          onInput={(e) => setLon((e.target as HTMLInputElement).value)}
          placeholder="Longitude"
          required
        />
      </div>
      <div class="form-actions">
        <button type="submit" class="btn btn-primary">Add Place</button>
        <button type="button" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  )
}
```

**Step 2: Create TagPicker component**

Create `frontend/src/components/TagPicker.tsx`:
```tsx
import { useState, useEffect } from 'preact/hooks'
import { Tag, pb } from '../lib/pocketbase'

interface Props {
  value: Tag[]
  onChange: (tags: Tag[]) => void
}

export function TagPicker({ value, onChange }: Props) {
  const [allTags, setAllTags] = useState<Tag[]>([])
  const [newTag, setNewTag] = useState('')

  useEffect(() => {
    pb.collection('tags').getFullList<Tag>().then(setAllTags)
  }, [])

  const toggleTag = (tag: Tag) => {
    if (value.some(t => t.id === tag.id)) {
      onChange(value.filter(t => t.id !== tag.id))
    } else {
      onChange([...value, tag])
    }
  }

  const addNewTag = async () => {
    if (!newTag.trim()) return
    try {
      const existing = allTags.find(t => t.name.toLowerCase() === newTag.toLowerCase())
      if (existing) {
        if (!value.some(t => t.id === existing.id)) {
          onChange([...value, existing])
        }
      } else {
        const created = await pb.collection('tags').create<Tag>({ name: newTag.trim() })
        setAllTags([...allTags, created])
        onChange([...value, created])
      }
      setNewTag('')
    } catch {
      // Tag might already exist
    }
  }

  return (
    <div class="tag-picker">
      <div class="tag-list">
        {allTags.map(tag => (
          <button
            key={tag.id}
            type="button"
            class={`tag ${value.some(t => t.id === tag.id) ? 'selected' : ''}`}
            style={tag.color ? { backgroundColor: tag.color } : undefined}
            onClick={() => toggleTag(tag)}
          >
            {tag.name}
          </button>
        ))}
      </div>
      <div class="tag-add">
        <input
          type="text"
          value={newTag}
          onInput={(e) => setNewTag((e.target as HTMLInputElement).value)}
          placeholder="Add new tag..."
          onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addNewTag())}
        />
        <button type="button" onClick={addNewTag}>+</button>
      </div>
    </div>
  )
}
```

**Step 3: Create Submit page**

Create `frontend/src/pages/Submit.tsx`:
```tsx
import { useState } from 'preact/hooks'
import { route } from 'preact-router'
import { pb, Place, Tag } from '../lib/pocketbase'
import { PlaceSearch } from '../components/PlaceSearch'
import { TagPicker } from '../components/TagPicker'
import './Submit.css'

export function Submit() {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [startDate, setStartDate] = useState('')
  const [startTime, setStartTime] = useState('')
  const [endDate, setEndDate] = useState('')
  const [endTime, setEndTime] = useState('')
  const [place, setPlace] = useState<Place | null>(null)
  const [tags, setTags] = useState<Tag[]>([])
  const [image, setImage] = useState<File | null>(null)
  const [email, setEmail] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const isLoggedIn = pb.authStore.isValid

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      const formData = new FormData()
      formData.append('title', title)
      formData.append('description', description)

      const startDatetime = new Date(`${startDate}T${startTime}`).toISOString()
      formData.append('start_datetime', startDatetime)

      if (endDate && endTime) {
        const endDatetime = new Date(`${endDate}T${endTime}`).toISOString()
        formData.append('end_datetime', endDatetime)
      }

      if (place) {
        formData.append('place', place.id)
      }

      if (tags.length > 0) {
        tags.forEach(t => formData.append('tags', t.id))
      }

      if (image) {
        formData.append('image', image)
      }

      if (!isLoggedIn && email) {
        formData.append('author_email', email)
      }

      // Set status based on auth and moderation settings
      formData.append('status', isLoggedIn ? 'published' : 'pending')

      const created = await pb.collection('events').create(formData)
      route(`/event/${created.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit event')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div class="submit-page">
      <h1>Submit an Event</h1>

      {error && <div class="error-message">{error}</div>}

      <form onSubmit={handleSubmit}>
        <div class="form-group">
          <label for="title">Event Title *</label>
          <input
            id="title"
            type="text"
            value={title}
            onInput={(e) => setTitle((e.target as HTMLInputElement).value)}
            required
            maxLength={200}
          />
        </div>

        <div class="form-group">
          <label for="description">Description</label>
          <textarea
            id="description"
            value={description}
            onInput={(e) => setDescription((e.target as HTMLTextAreaElement).value)}
            rows={6}
          />
        </div>

        <div class="form-row">
          <div class="form-group">
            <label>Start *</label>
            <div class="datetime-inputs">
              <input
                type="date"
                value={startDate}
                onInput={(e) => setStartDate((e.target as HTMLInputElement).value)}
                required
              />
              <input
                type="time"
                value={startTime}
                onInput={(e) => setStartTime((e.target as HTMLInputElement).value)}
                required
              />
            </div>
          </div>
          <div class="form-group">
            <label>End</label>
            <div class="datetime-inputs">
              <input
                type="date"
                value={endDate}
                onInput={(e) => setEndDate((e.target as HTMLInputElement).value)}
              />
              <input
                type="time"
                value={endTime}
                onInput={(e) => setEndTime((e.target as HTMLInputElement).value)}
              />
            </div>
          </div>
        </div>

        <div class="form-group">
          <label>Location</label>
          <PlaceSearch value={place} onChange={setPlace} />
        </div>

        <div class="form-group">
          <label>Tags</label>
          <TagPicker value={tags} onChange={setTags} />
        </div>

        <div class="form-group">
          <label for="image">Image</label>
          <input
            id="image"
            type="file"
            accept="image/*"
            onChange={(e) => setImage((e.target as HTMLInputElement).files?.[0] || null)}
          />
        </div>

        {!isLoggedIn && (
          <div class="form-group">
            <label for="email">Your Email (for edit link)</label>
            <input
              id="email"
              type="email"
              value={email}
              onInput={(e) => setEmail((e.target as HTMLInputElement).value)}
              placeholder="We'll send you a link to edit your event"
            />
          </div>
        )}

        <div class="form-actions">
          <button type="submit" class="btn btn-primary" disabled={submitting}>
            {submitting ? 'Submitting...' : 'Submit Event'}
          </button>
        </div>

        {!isLoggedIn && (
          <p class="moderation-note">
            Your event will be reviewed before publishing.
          </p>
        )}
      </form>
    </div>
  )
}
```

**Step 4: Create Submit styles**

Create `frontend/src/pages/Submit.css`:
```css
.submit-page {
  max-width: 600px;
  margin: 0 auto;
}

.submit-page h1 {
  margin-bottom: 2rem;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  font-weight: 500;
  margin-bottom: 0.5rem;
}

.form-group input[type="text"],
.form-group input[type="email"],
.form-group input[type="date"],
.form-group input[type="time"],
.form-group textarea {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font: inherit;
}

.form-group textarea {
  resize: vertical;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.datetime-inputs {
  display: flex;
  gap: 0.5rem;
}

.datetime-inputs input {
  flex: 1;
}

.form-actions {
  margin-top: 2rem;
}

.moderation-note {
  margin-top: 1rem;
  font-size: 0.875rem;
  color: #6b7280;
}

.error-message {
  background: #fef2f2;
  border: 1px solid #fecaca;
  color: #dc2626;
  padding: 1rem;
  border-radius: 6px;
  margin-bottom: 1.5rem;
}

/* PlaceSearch styles */
.place-search input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.place-results {
  list-style: none;
  border: 1px solid var(--color-border);
  border-top: none;
  border-radius: 0 0 6px 6px;
  max-height: 200px;
  overflow-y: auto;
}

.place-results button {
  width: 100%;
  padding: 0.75rem;
  text-align: left;
  background: none;
  border: none;
  cursor: pointer;
  font: inherit;
}

.place-results button:hover {
  background: #f3f4f6;
}

.place-selected {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem;
  background: #f3f4f6;
  border-radius: 6px;
}

.place-selected button {
  background: none;
  border: none;
  font-size: 1.25rem;
  cursor: pointer;
  color: #6b7280;
}

.place-search .link {
  display: block;
  margin-top: 0.5rem;
  font-size: 0.875rem;
  color: var(--color-primary);
  background: none;
  border: none;
  cursor: pointer;
}

.manual-place-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.manual-place-form .coords {
  display: flex;
  gap: 0.5rem;
}

.manual-place-form .form-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0;
}

/* TagPicker styles */
.tag-picker .tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.tag-picker .tag {
  cursor: pointer;
  border: 2px solid transparent;
}

.tag-picker .tag.selected {
  border-color: var(--color-primary);
}

.tag-add {
  display: flex;
  gap: 0.5rem;
}

.tag-add input {
  flex: 1;
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.tag-add button {
  padding: 0.5rem 1rem;
  background: #f3f4f6;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
}
```

**Step 5: Update app.tsx**

Update `frontend/src/app.tsx`:
```tsx
import Router from 'preact-router'
import { Home } from './pages/Home'
import { Event } from './pages/Event'
import { Submit } from './pages/Submit'
import './style.css'

export function App() {
  return (
    <div class="app">
      <header>
        <a href="/" class="logo">Gather</a>
      </header>
      <main>
        <Router>
          <Home path="/" />
          <Event path="/event/:id" />
          <Submit path="/submit" />
        </Router>
      </main>
    </div>
  )
}
```

**Step 6: Commit**

```bash
git add frontend/src/components/PlaceSearch.tsx frontend/src/components/TagPicker.tsx frontend/src/pages/Submit.tsx frontend/src/pages/Submit.css frontend/src/app.tsx
git commit -m "feat: add event submission form with place search and tags"
```

---

## Phase 4: Backend Features

### Task 11: Embed Frontend in Go Binary

**Files:**
- Modify: `embed.go`
- Modify: `main.go`

**Step 1: Update embed.go**

Replace `embed.go`:
```go
package main

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend/dist
var frontendFiles embed.FS

func frontendFS() (fs.FS, error) {
	return fs.Sub(frontendFiles, "frontend/dist")
}
```

**Step 2: Update main.go to serve frontend**

Replace `main.go`:
```go
package main

import (
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Serve embedded frontend
		frontend, err := frontendFS()
		if err != nil {
			return err
		}

		// Serve static files, fallback to index.html for SPA routing
		se.Router.GET("/{path...}", func(re *core.RequestEvent) error {
			path := re.Request.PathValue("path")

			// Skip API and admin routes
			if strings.HasPrefix(path, "api/") || strings.HasPrefix(path, "_/") {
				return re.Next()
			}

			// Try to serve the file
			f, err := frontend.Open(path)
			if err == nil {
				f.Close()
				return re.FileFS(frontend, path)
			}

			// Fallback to index.html for SPA
			return re.FileFS(frontend, "index.html")
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
```

**Step 3: Build frontend**

Run:
```bash
cd frontend && npm run build
```

Expected: `frontend/dist` created with built files

**Step 4: Build Go binary**

Run:
```bash
go build -o gather
```

Expected: `gather` binary created

**Step 5: Test embedded frontend**

Run:
```bash
./gather serve
```

Then visit http://127.0.0.1:8090/

Expected: Preact app loads from embedded files

**Step 6: Commit**

```bash
git add embed.go main.go
git commit -m "feat: embed frontend in Go binary"
```

---

### Task 12: Add RSS Feed

**Files:**
- Modify: `internal/rss/export.go`
- Modify: `main.go`

**Step 1: Create RSS feed generator**

Replace `internal/rss/export.go`:
```go
package rss

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate,omitempty"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func GenerateFeed(app core.App, baseURL string, filter string) ([]byte, error) {
	// Get settings
	settings, _ := app.FindFirstRecordByFilter("settings", "id != ''")

	instanceName := "Gather"
	instanceDesc := "Community Events"
	if settings != nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
		if desc := settings.GetString("instance_description"); desc != "" {
			instanceDesc = desc
		}
	}

	// Build filter
	fullFilter := "status = 'published'"
	if filter != "" {
		fullFilter += " && " + filter
	}

	// Get events
	events, err := app.FindRecordsByFilter(
		"events",
		fullFilter,
		"-start_datetime",
		50,
		0,
	)
	if err != nil {
		return nil, err
	}

	// Build items
	items := make([]Item, 0, len(events))
	for _, event := range events {
		startTime := event.GetDateTime("start_datetime").Time()

		items = append(items, Item{
			Title:       event.GetString("title"),
			Link:        fmt.Sprintf("%s/event/%s", baseURL, event.Id),
			Description: event.GetString("description"),
			PubDate:     event.GetDateTime("created").Time().Format(time.RFC1123Z),
			GUID:        fmt.Sprintf("%s/event/%s", baseURL, event.Id),
		})
	}

	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       instanceName,
			Link:        baseURL,
			Description: instanceDesc,
			Items:       items,
		},
	}

	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}
```

**Step 2: Add RSS routes to main.go**

Update `main.go`:
```go
package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
	"gather/internal/rss"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		baseURL := se.App.Settings().Meta.AppURL
		if baseURL == "" {
			baseURL = "http://127.0.0.1:8090"
		}

		// RSS feeds
		se.Router.GET("/feed.rss", func(re *core.RequestEvent) error {
			data, err := rss.GenerateFeed(se.App, baseURL, "")
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "application/rss+xml")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "application/rss+xml", data)
		})

		se.Router.GET("/feed/tag/{tagname}.rss", func(re *core.RequestEvent) error {
			tagname := re.Request.PathValue("tagname")
			tag, err := se.App.FindFirstRecordByFilter("tags", "name = {:name}", map[string]any{"name": tagname})
			if err != nil {
				return re.NotFoundError("Tag not found", err)
			}
			filter := "tags.id ?= '" + tag.Id + "'"
			data, err := rss.GenerateFeed(se.App, baseURL, filter)
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "application/rss+xml")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "application/rss+xml", data)
		})

		// Serve embedded frontend
		frontend, err := frontendFS()
		if err != nil {
			return err
		}

		se.Router.GET("/{path...}", func(re *core.RequestEvent) error {
			path := re.Request.PathValue("path")

			if strings.HasPrefix(path, "api/") || strings.HasPrefix(path, "_/") {
				return re.Next()
			}

			f, err := frontend.Open(path)
			if err == nil {
				f.Close()
				return re.FileFS(frontend, path)
			}

			return re.FileFS(frontend, "index.html")
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
```

**Step 3: Run go mod tidy**

Run:
```bash
go mod tidy
```

**Step 4: Test RSS feed**

Run:
```bash
go run . serve
```

Then visit http://127.0.0.1:8090/feed.rss

Expected: Valid RSS XML

**Step 5: Commit**

```bash
git add internal/rss/export.go main.go
git commit -m "feat: add RSS feed endpoints"
```

---

### Task 13: Add ICS Feed

**Files:**
- Modify: `internal/ical/export.go`
- Modify: `main.go`

**Step 1: Create ICS generator**

Replace `internal/ical/export.go`:
```go
package ical

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func formatICSTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func GenerateFeed(app core.App, baseURL string, filter string) ([]byte, error) {
	settings, _ := app.FindFirstRecordByFilter("settings", "id != ''")

	instanceName := "Gather"
	if settings != nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	fullFilter := "status = 'published'"
	if filter != "" {
		fullFilter += " && " + filter
	}

	events, err := app.FindRecordsByFilter(
		"events",
		fullFilter,
		"-start_datetime",
		100,
		0,
	)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString(fmt.Sprintf("PRODID:-//%s//Gather//EN\r\n", instanceName))
	sb.WriteString(fmt.Sprintf("X-WR-CALNAME:%s\r\n", instanceName))

	for _, event := range events {
		startTime := event.GetDateTime("start_datetime").Time()
		endTime := event.GetDateTime("end_datetime").Time()
		if endTime.IsZero() {
			endTime = startTime.Add(time.Hour)
		}

		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s@%s\r\n", event.Id, baseURL))
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICSTime(event.GetDateTime("created").Time())))
		sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICSTime(startTime)))
		sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICSTime(endTime)))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(event.GetString("title"))))

		if desc := event.GetString("description"); desc != "" {
			// Strip HTML for ICS
			desc = stripHTML(desc)
			sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(desc)))
		}

		sb.WriteString(fmt.Sprintf("URL:%s/event/%s\r\n", baseURL, event.Id))
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")
	return []byte(sb.String()), nil
}

func GenerateSingleEvent(app core.App, baseURL string, eventID string) ([]byte, error) {
	event, err := app.FindRecordById("events", eventID)
	if err != nil {
		return nil, err
	}

	settings, _ := app.FindFirstRecordByFilter("settings", "id != ''")
	instanceName := "Gather"
	if settings != nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	startTime := event.GetDateTime("start_datetime").Time()
	endTime := event.GetDateTime("end_datetime").Time()
	if endTime.IsZero() {
		endTime = startTime.Add(time.Hour)
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString(fmt.Sprintf("PRODID:-//%s//Gather//EN\r\n", instanceName))
	sb.WriteString("BEGIN:VEVENT\r\n")
	sb.WriteString(fmt.Sprintf("UID:%s@%s\r\n", event.Id, baseURL))
	sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICSTime(event.GetDateTime("created").Time())))
	sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICSTime(startTime)))
	sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICSTime(endTime)))
	sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(event.GetString("title"))))

	if desc := event.GetString("description"); desc != "" {
		desc = stripHTML(desc)
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(desc)))
	}

	sb.WriteString(fmt.Sprintf("URL:%s/event/%s\r\n", baseURL, event.Id))
	sb.WriteString("END:VEVENT\r\n")
	sb.WriteString("END:VCALENDAR\r\n")

	return []byte(sb.String()), nil
}

func stripHTML(s string) string {
	// Simple HTML stripping
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
```

**Step 2: Add ICS routes to main.go**

Add to `main.go` inside `OnServe()`:
```go
		// ICS feeds
		se.Router.GET("/feed.ics", func(re *core.RequestEvent) error {
			data, err := ical.GenerateFeed(se.App, baseURL, "")
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "text/calendar")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "text/calendar", data)
		})

		se.Router.GET("/feed/tag/{tagname}.ics", func(re *core.RequestEvent) error {
			tagname := re.Request.PathValue("tagname")
			tag, err := se.App.FindFirstRecordByFilter("tags", "name = {:name}", map[string]any{"name": tagname})
			if err != nil {
				return re.NotFoundError("Tag not found", err)
			}
			filter := "tags.id ?= '" + tag.Id + "'"
			data, err := ical.GenerateFeed(se.App, baseURL, filter)
			if err != nil {
				return re.InternalServerError("Failed to generate feed", err)
			}
			re.Response.Header().Set("Content-Type", "text/calendar")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "text/calendar", data)
		})

		se.Router.GET("/event/{id}.ics", func(re *core.RequestEvent) error {
			id := re.Request.PathValue("id")
			data, err := ical.GenerateSingleEvent(se.App, baseURL, id)
			if err != nil {
				return re.NotFoundError("Event not found", err)
			}
			re.Response.Header().Set("Content-Type", "text/calendar")
			re.Response.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=60")
			return re.Blob(200, "text/calendar", data)
		})
```

Update imports:
```go
import (
	"log"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
	"gather/internal/ical"
	"gather/internal/rss"
)
```

**Step 3: Test ICS feed**

Run:
```bash
go run . serve
```

Then visit http://127.0.0.1:8090/feed.ics

Expected: Valid ICS file

**Step 4: Commit**

```bash
git add internal/ical/export.go main.go
git commit -m "feat: add ICS calendar feed endpoints"
```

---

## Phase 5: ActivityPub

### Task 14: Generate Actor Keypair

**Files:**
- Modify: `internal/activitypub/actor.go`
- Create: `internal/activitypub/keys.go`
- Modify: `main.go`

**Step 1: Create key generation utility**

Create `internal/activitypub/keys.go`:
```go
package activitypub

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

func GenerateKeyPair() (privateKey string, publicKey string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	privateBytes := x509.MarshalPKCS1PrivateKey(key)
	privatePem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateBytes,
	})

	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", err
	}
	publicPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicBytes,
	})

	return string(privatePem), string(publicPem), nil
}

func ParsePrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, nil
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
```

**Step 2: Create actor implementation**

Replace `internal/activitypub/actor.go`:
```go
package activitypub

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
)

type Actor struct {
	Context           any        `json:"@context"`
	Type              string     `json:"type"`
	ID                string     `json:"id"`
	PreferredUsername string     `json:"preferredUsername"`
	Name              string     `json:"name"`
	Summary           string     `json:"summary,omitempty"`
	Inbox             string     `json:"inbox"`
	Outbox            string     `json:"outbox"`
	PublicKey         *PublicKey `json:"publicKey,omitempty"`
}

type PublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}

func GetActor(app core.App, baseURL string) (*Actor, error) {
	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		return nil, err
	}

	name := settings.GetString("instance_name")
	if name == "" {
		name = "Gather"
	}
	summary := settings.GetString("instance_description")
	publicKey := settings.GetString("ap_public_key")

	actorID := baseURL + "/ap/actor"

	actor := &Actor{
		Context: []any{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		Type:              "Application",
		ID:                actorID,
		PreferredUsername: "events",
		Name:              name,
		Summary:           summary,
		Inbox:             baseURL + "/ap/inbox",
		Outbox:            baseURL + "/ap/outbox",
	}

	if publicKey != "" {
		actor.PublicKey = &PublicKey{
			ID:           actorID + "#main-key",
			Owner:        actorID,
			PublicKeyPem: publicKey,
		}
	}

	return actor, nil
}

func (a *Actor) ToJSON() ([]byte, error) {
	return json.MarshalIndent(a, "", "  ")
}

// EnsureKeypair creates AP keys if they don't exist
func EnsureKeypair(app core.App) error {
	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		// Create settings record
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}
		settings = core.NewRecord(collection)
	}

	if settings.GetString("ap_private_key") != "" {
		return nil // Already has keys
	}

	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		return err
	}

	settings.Set("ap_private_key", privateKey)
	settings.Set("ap_public_key", publicKey)
	settings.Set("ap_enabled", true)

	return app.Save(settings)
}
```

**Step 3: Add actor route and key init to main.go**

Add to `main.go`:
```go
package main

import (
	"log"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
	"gather/internal/activitypub"
	"gather/internal/ical"
	"gather/internal/rss"
)

func main() {
	app := pocketbase.New()

	// Initialize AP keypair on first run
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := activitypub.EnsureKeypair(se.App); err != nil {
			log.Println("Warning: failed to ensure AP keypair:", err)
		}
		return se.Next()
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		baseURL := se.App.Settings().Meta.AppURL
		if baseURL == "" {
			baseURL = "http://127.0.0.1:8090"
		}

		// ActivityPub actor
		se.Router.GET("/ap/actor", func(re *core.RequestEvent) error {
			actor, err := activitypub.GetActor(se.App, baseURL)
			if err != nil {
				return re.InternalServerError("Failed to get actor", err)
			}
			data, err := actor.ToJSON()
			if err != nil {
				return re.InternalServerError("Failed to serialize actor", err)
			}
			re.Response.Header().Set("Content-Type", "application/activity+json")
			re.Response.Header().Set("Cache-Control", "public, max-age=3600")
			return re.Blob(200, "application/activity+json", data)
		})

		// ... rest of routes
```

**Step 4: Commit**

```bash
git add internal/activitypub/ main.go
git commit -m "feat: add ActivityPub actor with keypair generation"
```

---

### Task 15: Add Webfinger Endpoint

**Files:**
- Create: `internal/activitypub/webfinger.go`
- Modify: `main.go`

**Step 1: Create webfinger implementation**

Create `internal/activitypub/webfinger.go`:
```go
package activitypub

import (
	"encoding/json"
	"fmt"
	"strings"
)

type WebfingerResponse struct {
	Subject string          `json:"subject"`
	Links   []WebfingerLink `json:"links"`
}

type WebfingerLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
	Href string `json:"href,omitempty"`
}

func HandleWebfinger(resource string, baseURL string) ([]byte, error) {
	// Parse resource (format: acct:events@domain)
	if !strings.HasPrefix(resource, "acct:") {
		return nil, fmt.Errorf("invalid resource format")
	}

	parts := strings.SplitN(strings.TrimPrefix(resource, "acct:"), "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resource format")
	}

	username := parts[0]
	if username != "events" {
		return nil, fmt.Errorf("user not found")
	}

	response := WebfingerResponse{
		Subject: resource,
		Links: []WebfingerLink{
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: baseURL + "/ap/actor",
			},
		},
	}

	return json.MarshalIndent(response, "", "  ")
}
```

**Step 2: Add webfinger route to main.go**

Add to routes in `main.go`:
```go
		// Webfinger
		se.Router.GET("/.well-known/webfinger", func(re *core.RequestEvent) error {
			resource := re.Request.URL.Query().Get("resource")
			if resource == "" {
				return re.BadRequestError("Missing resource parameter", nil)
			}
			data, err := activitypub.HandleWebfinger(resource, baseURL)
			if err != nil {
				return re.NotFoundError("Resource not found", err)
			}
			re.Response.Header().Set("Content-Type", "application/jrd+json")
			re.Response.Header().Set("Cache-Control", "public, max-age=3600")
			return re.Blob(200, "application/jrd+json", data)
		})
```

**Step 3: Commit**

```bash
git add internal/activitypub/webfinger.go main.go
git commit -m "feat: add webfinger endpoint for ActivityPub discovery"
```

---

### Task 16: Add Outbox and Event Broadcasting

**Files:**
- Modify: `internal/activitypub/outbox.go`
- Create: `internal/activitypub/delivery.go`
- Modify: `internal/hooks/events.go`
- Modify: `main.go`

**Step 1: Create outbox implementation**

Create `internal/activitypub/outbox.go`:
```go
package activitypub

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type OrderedCollection struct {
	Context      string `json:"@context"`
	Type         string `json:"type"`
	ID           string `json:"id"`
	TotalItems   int    `json:"totalItems"`
	OrderedItems []any  `json:"orderedItems"`
}

type Activity struct {
	Context any    `json:"@context"`
	Type    string `json:"type"`
	ID      string `json:"id"`
	Actor   string `json:"actor"`
	Object  any    `json:"object"`
}

type Note struct {
	Context      any       `json:"@context,omitempty"`
	Type         string    `json:"type"`
	ID           string    `json:"id"`
	AttributedTo string    `json:"attributedTo"`
	Content      string    `json:"content"`
	Published    time.Time `json:"published"`
	URL          string    `json:"url"`
	To           []string  `json:"to"`
	Cc           []string  `json:"cc,omitempty"`
	Attachment   []any     `json:"attachment,omitempty"`
	Tag          []any     `json:"tag,omitempty"`
}

func GetOutbox(app core.App, baseURL string) ([]byte, error) {
	events, err := app.FindRecordsByFilter(
		"events",
		"status = 'published'",
		"-start_datetime",
		20,
		0,
	)
	if err != nil {
		return nil, err
	}

	items := make([]any, 0, len(events))
	for _, event := range events {
		note := eventToNote(event, baseURL)
		activity := Activity{
			Type:   "Create",
			ID:     fmt.Sprintf("%s/ap/activities/%s", baseURL, event.Id),
			Actor:  baseURL + "/ap/actor",
			Object: note,
		}
		items = append(items, activity)
	}

	collection := OrderedCollection{
		Context:      "https://www.w3.org/ns/activitystreams",
		Type:         "OrderedCollection",
		ID:           baseURL + "/ap/outbox",
		TotalItems:   len(items),
		OrderedItems: items,
	}

	return json.MarshalIndent(collection, "", "  ")
}

func eventToNote(event *core.Record, baseURL string) Note {
	startTime := event.GetDateTime("start_datetime").Time()

	// Build content HTML
	title := html.EscapeString(event.GetString("title"))
	desc := event.GetString("description")

	content := fmt.Sprintf(
		"<p><strong>%s</strong></p><p>%s</p><p>%s</p>",
		title,
		startTime.Format("Monday, January 2, 2006 at 3:04 PM"),
		desc,
	)

	note := Note{
		Type:         "Note",
		ID:           fmt.Sprintf("%s/ap/events/%s", baseURL, event.Id),
		AttributedTo: baseURL + "/ap/actor",
		Content:      content,
		Published:    event.GetDateTime("created").Time(),
		URL:          fmt.Sprintf("%s/event/%s", baseURL, event.Id),
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:           []string{baseURL + "/ap/actor/followers"},
	}

	return note
}

func CreateActivityForEvent(event *core.Record, baseURL string, activityType string) Activity {
	note := eventToNote(event, baseURL)

	return Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    activityType,
		ID:      fmt.Sprintf("%s/ap/activities/%s/%d", baseURL, event.Id, time.Now().Unix()),
		Actor:   baseURL + "/ap/actor",
		Object:  note,
	}
}
```

**Step 2: Create delivery implementation**

Create `internal/activitypub/delivery.go`:
```go
package activitypub

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func DeliverActivity(app core.App, activity Activity, inboxURL string) error {
	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		return err
	}

	privateKeyPem := settings.GetString("ap_private_key")
	if privateKeyPem == "" {
		return fmt.Errorf("no private key configured")
	}

	privateKey, err := ParsePrivateKey(privateKeyPem)
	if err != nil {
		return err
	}

	body, err := json.Marshal(activity)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", inboxURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/activity+json")
	req.Header.Set("Accept", "application/activity+json")

	// Sign the request
	if err := signRequest(req, privateKey, activity.Actor+"#main-key", body); err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("delivery failed with status %d", resp.StatusCode)
	}

	return nil
}

func signRequest(req *http.Request, privateKey *rsa.PrivateKey, keyID string, body []byte) error {
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)

	// Calculate digest
	h := sha256.Sum256(body)
	digest := "SHA-256=" + base64.StdEncoding.EncodeToString(h[:])
	req.Header.Set("Digest", digest)

	// Build signature string
	signedString := fmt.Sprintf(
		"(request-target): post %s\nhost: %s\ndate: %s\ndigest: %s",
		req.URL.Path,
		req.URL.Host,
		date,
		digest,
	)

	// Sign
	hashed := sha256.Sum256([]byte(signedString))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return err
	}

	sigHeader := fmt.Sprintf(
		`keyId="%s",algorithm="rsa-sha256",headers="(request-target) host date digest",signature="%s"`,
		keyID,
		base64.StdEncoding.EncodeToString(signature),
	)
	req.Header.Set("Signature", sigHeader)

	return nil
}

func QueueDeliveryToFollowers(app core.App, activity Activity) error {
	followers, err := app.FindRecordsByFilter("ap_followers", "", "", 0, 0)
	if err != nil {
		return err
	}

	// Group by shared inbox
	inboxes := make(map[string]bool)
	for _, follower := range followers {
		inbox := follower.GetString("shared_inbox_url")
		if inbox == "" {
			inbox = follower.GetString("inbox_url")
		}
		inboxes[inbox] = true
	}

	collection, err := app.FindCollectionByNameOrId("ap_delivery_queue")
	if err != nil {
		return err
	}

	for inbox := range inboxes {
		record := core.NewRecord(collection)
		activityJSON, _ := json.Marshal(activity)
		record.Set("activity", string(activityJSON))
		record.Set("inbox_url", inbox)
		record.Set("attempts", 0)
		record.Set("next_retry", time.Now())
		if err := app.Save(record); err != nil {
			return err
		}
	}

	return nil
}
```

**Step 3: Create event hooks**

Replace `internal/hooks/events.go`:
```go
package hooks

import (
	"gather/internal/activitypub"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterEventHooks(app core.App, baseURL string) {
	// After event is published, queue AP delivery
	app.OnRecordAfterUpdateSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		// Check if status changed to published
		oldStatus := e.Record.Original().GetString("status")
		newStatus := e.Record.GetString("status")

		if oldStatus != "published" && newStatus == "published" {
			// New publication
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Create")
			go activitypub.QueueDeliveryToFollowers(app, activity)
		} else if oldStatus == "published" && newStatus == "published" {
			// Update
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Update")
			go activitypub.QueueDeliveryToFollowers(app, activity)
		}

		return e.Next()
	})

	app.OnRecordAfterDeleteSuccess("events").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("status") == "published" {
			activity := activitypub.CreateActivityForEvent(e.Record, baseURL, "Delete")
			go activitypub.QueueDeliveryToFollowers(app, activity)
		}
		return e.Next()
	})
}
```

**Step 4: Add outbox route and register hooks in main.go**

Update `main.go`:
```go
		// ActivityPub outbox
		se.Router.GET("/ap/outbox", func(re *core.RequestEvent) error {
			data, err := activitypub.GetOutbox(se.App, baseURL)
			if err != nil {
				return re.InternalServerError("Failed to get outbox", err)
			}
			re.Response.Header().Set("Content-Type", "application/activity+json")
			re.Response.Header().Set("Cache-Control", "public, max-age=300")
			return re.Blob(200, "application/activity+json", data)
		})
```

And register hooks:
```go
	// Register event hooks
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		baseURL := se.App.Settings().Meta.AppURL
		if baseURL == "" {
			baseURL = "http://127.0.0.1:8090"
		}
		hooks.RegisterEventHooks(se.App, baseURL)
		return se.Next()
	})
```

Add import:
```go
	"gather/internal/hooks"
```

**Step 5: Commit**

```bash
git add internal/activitypub/ internal/hooks/ main.go
git commit -m "feat: add ActivityPub outbox and event delivery"
```

---

### Task 17: Add Inbox for Follow Requests

**Files:**
- Create: `internal/activitypub/inbox.go`
- Modify: `main.go`

**Step 1: Create inbox handler**

Create `internal/activitypub/inbox.go`:
```go
package activitypub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

type IncomingActivity struct {
	Type   string          `json:"type"`
	ID     string          `json:"id"`
	Actor  string          `json:"actor"`
	Object json.RawMessage `json:"object"`
}

func HandleInbox(app core.App, baseURL string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	var activity IncomingActivity
	if err := json.Unmarshal(data, &activity); err != nil {
		return err
	}

	switch activity.Type {
	case "Follow":
		return handleFollow(app, baseURL, activity)
	case "Undo":
		return handleUndo(app, activity)
	default:
		// Ignore other activity types
		return nil
	}
}

func handleFollow(app core.App, baseURL string, activity IncomingActivity) error {
	// Fetch actor info
	actorInfo, err := fetchActor(activity.Actor)
	if err != nil {
		return err
	}

	// Store follower
	collection, err := app.FindCollectionByNameOrId("ap_followers")
	if err != nil {
		return err
	}

	// Check if already following
	existing, _ := app.FindFirstRecordByFilter("ap_followers", "actor_url = {:url}", map[string]any{"url": activity.Actor})
	if existing != nil {
		return nil // Already following
	}

	record := core.NewRecord(collection)
	record.Set("actor_url", activity.Actor)
	record.Set("inbox_url", actorInfo.Inbox)
	record.Set("shared_inbox_url", actorInfo.SharedInbox)

	if err := app.Save(record); err != nil {
		return err
	}

	// Send Accept
	accept := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Accept",
		ID:      fmt.Sprintf("%s/ap/activities/accept/%s", baseURL, record.Id),
		Actor:   baseURL + "/ap/actor",
		Object:  activity,
	}

	go DeliverActivity(app, accept, actorInfo.Inbox)

	return nil
}

func handleUndo(app core.App, activity IncomingActivity) error {
	// Parse the object being undone
	var undoObject struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(activity.Object, &undoObject); err != nil {
		return err
	}

	if undoObject.Type == "Follow" {
		// Remove follower
		follower, err := app.FindFirstRecordByFilter("ap_followers", "actor_url = {:url}", map[string]any{"url": activity.Actor})
		if err != nil {
			return nil // Not found, already unfollowed
		}
		return app.Delete(follower)
	}

	return nil
}

type ActorInfo struct {
	Inbox       string
	SharedInbox string
}

func fetchActor(actorURL string) (*ActorInfo, error) {
	req, err := http.NewRequest("GET", actorURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var actor struct {
		Inbox     string `json:"inbox"`
		Endpoints struct {
			SharedInbox string `json:"sharedInbox"`
		} `json:"endpoints"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		return nil, err
	}

	return &ActorInfo{
		Inbox:       actor.Inbox,
		SharedInbox: actor.Endpoints.SharedInbox,
	}, nil
}
```

**Step 2: Add inbox route to main.go**

Add to routes:
```go
		// ActivityPub inbox
		se.Router.POST("/ap/inbox", func(re *core.RequestEvent) error {
			if err := activitypub.HandleInbox(se.App, baseURL, re.Request.Body); err != nil {
				return re.InternalServerError("Failed to process activity", err)
			}
			return re.NoContent(202)
		})
```

**Step 3: Commit**

```bash
git add internal/activitypub/inbox.go main.go
git commit -m "feat: add ActivityPub inbox for follow/unfollow"
```

---

## Phase 6: Polish

### Task 18: Add Tag and Place Pages

**Files:**
- Create: `frontend/src/pages/Tag.tsx`
- Create: `frontend/src/pages/Place.tsx`
- Modify: `frontend/src/app.tsx`

**Step 1: Create Tag page**

Create `frontend/src/pages/Tag.tsx`:
```tsx
import { useEffect, useState } from 'preact/hooks'
import { pb, Event, Tag as TagType } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import './Home.css'

interface Props {
  name: string
}

export function Tag({ name }: Props) {
  const [events, setEvents] = useState<Event[]>([])
  const [tag, setTag] = useState<TagType | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function load() {
      try {
        const tagRecord = await pb.collection('tags').getFirstListItem<TagType>(`name = '${name}'`)
        setTag(tagRecord)

        const eventsResult = await pb.collection('events').getList<Event>(1, 50, {
          filter: `status = 'published' && tags.id ?= '${tagRecord.id}'`,
          sort: 'start_datetime',
          expand: 'place,tags',
        })
        setEvents(eventsResult.items)
      } catch {
        // Tag not found
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [name])

  if (loading) return <div class="loading">Loading...</div>
  if (!tag) return <div class="error">Tag not found</div>

  return (
    <div class="tag-page">
      <h1>
        <span class="tag" style={tag.color ? { backgroundColor: tag.color } : undefined}>
          {tag.name}
        </span>
      </h1>
      {events.length === 0 ? (
        <p class="no-events">No events with this tag</p>
      ) : (
        <div class="events-grid">
          {events.map(event => (
            <EventCard key={event.id} event={event} />
          ))}
        </div>
      )}
    </div>
  )
}
```

**Step 2: Create Place page**

Create `frontend/src/pages/Place.tsx`:
```tsx
import { useEffect, useState, useRef } from 'preact/hooks'
import L from 'leaflet'
import { pb, Event, Place as PlaceType } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import './Home.css'

interface Props {
  id: string
}

export function Place({ id }: Props) {
  const [events, setEvents] = useState<Event[]>([])
  const [place, setPlace] = useState<PlaceType | null>(null)
  const [loading, setLoading] = useState(true)
  const mapRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    async function load() {
      try {
        const placeRecord = await pb.collection('places').getOne<PlaceType>(id)
        setPlace(placeRecord)

        const eventsResult = await pb.collection('events').getList<Event>(1, 50, {
          filter: `status = 'published' && place = '${id}'`,
          sort: 'start_datetime',
          expand: 'place,tags',
        })
        setEvents(eventsResult.items)
      } catch {
        // Place not found
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [id])

  useEffect(() => {
    if (!place || !mapRef.current) return
    const map = L.map(mapRef.current).setView([place.latitude, place.longitude], 15)
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; OpenStreetMap contributors',
    }).addTo(map)
    L.marker([place.latitude, place.longitude]).addTo(map)
    return () => map.remove()
  }, [place])

  if (loading) return <div class="loading">Loading...</div>
  if (!place) return <div class="error">Place not found</div>

  return (
    <div class="place-page">
      <h1>{place.name}</h1>
      {place.address && <p class="place-address">{place.address}</p>}
      <div ref={mapRef} style={{ height: '200px', marginBottom: '2rem', borderRadius: '8px' }} />
      {events.length === 0 ? (
        <p class="no-events">No events at this location</p>
      ) : (
        <div class="events-grid">
          {events.map(event => (
            <EventCard key={event.id} event={event} />
          ))}
        </div>
      )}
    </div>
  )
}
```

**Step 3: Update app.tsx**

Update `frontend/src/app.tsx`:
```tsx
import Router from 'preact-router'
import { Home } from './pages/Home'
import { Event } from './pages/Event'
import { Submit } from './pages/Submit'
import { Tag } from './pages/Tag'
import { Place } from './pages/Place'
import './style.css'

export function App() {
  return (
    <div class="app">
      <header>
        <a href="/" class="logo">Gather</a>
      </header>
      <main>
        <Router>
          <Home path="/" />
          <Event path="/event/:id" />
          <Submit path="/submit" />
          <Tag path="/tag/:name" />
          <Place path="/place/:id" />
        </Router>
      </main>
    </div>
  )
}
```

**Step 4: Commit**

```bash
git add frontend/src/pages/Tag.tsx frontend/src/pages/Place.tsx frontend/src/app.tsx
git commit -m "feat: add Tag and Place filter pages"
```

---

### Task 19: Add Login Page

**Files:**
- Create: `frontend/src/pages/Login.tsx`
- Create: `frontend/src/pages/Login.css`
- Modify: `frontend/src/app.tsx`
- Modify: `frontend/src/app.tsx` (header)

**Step 1: Create Login page**

Create `frontend/src/pages/Login.tsx`:
```tsx
import { useState } from 'preact/hooks'
import { route } from 'preact-router'
import { pb } from '../lib/pocketbase'
import './Login.css'

export function Login() {
  const [isRegister, setIsRegister] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [passwordConfirm, setPasswordConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      if (isRegister) {
        await pb.collection('users').create({
          email,
          password,
          passwordConfirm,
        })
      }
      await pb.collection('users').authWithPassword(email, password)
      route('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authentication failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div class="login-page">
      <h1>{isRegister ? 'Create Account' : 'Login'}</h1>

      {error && <div class="error-message">{error}</div>}

      <form onSubmit={handleSubmit}>
        <div class="form-group">
          <label for="email">Email</label>
          <input
            id="email"
            type="email"
            value={email}
            onInput={(e) => setEmail((e.target as HTMLInputElement).value)}
            required
          />
        </div>

        <div class="form-group">
          <label for="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onInput={(e) => setPassword((e.target as HTMLInputElement).value)}
            required
          />
        </div>

        {isRegister && (
          <div class="form-group">
            <label for="passwordConfirm">Confirm Password</label>
            <input
              id="passwordConfirm"
              type="password"
              value={passwordConfirm}
              onInput={(e) => setPasswordConfirm((e.target as HTMLInputElement).value)}
              required
            />
          </div>
        )}

        <button type="submit" class="btn btn-primary" disabled={loading}>
          {loading ? 'Please wait...' : isRegister ? 'Create Account' : 'Login'}
        </button>
      </form>

      <p class="toggle-mode">
        {isRegister ? 'Already have an account?' : "Don't have an account?"}{' '}
        <button type="button" class="link" onClick={() => setIsRegister(!isRegister)}>
          {isRegister ? 'Login' : 'Register'}
        </button>
      </p>
    </div>
  )
}
```

**Step 2: Create Login styles**

Create `frontend/src/pages/Login.css`:
```css
.login-page {
  max-width: 400px;
  margin: 2rem auto;
}

.login-page h1 {
  margin-bottom: 2rem;
}

.toggle-mode {
  margin-top: 1.5rem;
  text-align: center;
}

.toggle-mode .link {
  background: none;
  border: none;
  color: var(--color-primary);
  cursor: pointer;
  font: inherit;
}
```

**Step 3: Update app.tsx with auth header**

Replace `frontend/src/app.tsx`:
```tsx
import { useState, useEffect } from 'preact/hooks'
import Router from 'preact-router'
import { Home } from './pages/Home'
import { Event } from './pages/Event'
import { Submit } from './pages/Submit'
import { Tag } from './pages/Tag'
import { Place } from './pages/Place'
import { Login } from './pages/Login'
import { pb } from './lib/pocketbase'
import './style.css'

export function App() {
  const [user, setUser] = useState(pb.authStore.record)

  useEffect(() => {
    return pb.authStore.onChange(() => {
      setUser(pb.authStore.record)
    })
  }, [])

  const handleLogout = () => {
    pb.authStore.clear()
  }

  return (
    <div class="app">
      <header>
        <a href="/" class="logo">Gather</a>
        <nav>
          {user ? (
            <>
              <span>{user.email}</span>
              <button onClick={handleLogout} class="link">Logout</button>
            </>
          ) : (
            <a href="/login">Login</a>
          )}
        </nav>
      </header>
      <main>
        <Router>
          <Home path="/" />
          <Event path="/event/:id" />
          <Submit path="/submit" />
          <Tag path="/tag/:name" />
          <Place path="/place/:id" />
          <Login path="/login" />
        </Router>
      </main>
    </div>
  )
}
```

**Step 4: Update header styles**

Add to `frontend/src/style.css`:
```css
header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 1rem;
  margin-bottom: 2rem;
}

header nav {
  display: flex;
  align-items: center;
  gap: 1rem;
}

header nav a {
  color: var(--color-primary);
  text-decoration: none;
}

header .link {
  background: none;
  border: none;
  color: var(--color-primary);
  cursor: pointer;
  font: inherit;
}
```

**Step 5: Commit**

```bash
git add frontend/src/pages/Login.tsx frontend/src/pages/Login.css frontend/src/app.tsx frontend/src/style.css
git commit -m "feat: add login/register page with auth header"
```

---

### Task 20: Final Build and Test

**Step 1: Build frontend**

Run:
```bash
cd frontend && npm run build
```

**Step 2: Build Go binary**

Run:
```bash
go build -o gather
```

**Step 3: Run the application**

Run:
```bash
./gather serve
```

**Step 4: Create admin user**

Visit http://127.0.0.1:8090/_/ and create admin account

**Step 5: Test all features**

- [ ] Home page loads
- [ ] Can submit event (anonymous)
- [ ] Can register/login
- [ ] Can submit event (logged in)
- [ ] Event detail page works
- [ ] Map displays on event page
- [ ] RSS feed works (/feed.rss)
- [ ] ICS feed works (/feed.ics)
- [ ] ActivityPub actor works (/ap/actor)
- [ ] Webfinger works (/.well-known/webfinger?resource=acct:events@localhost)

**Step 6: Commit final state**

```bash
git add .
git commit -m "feat: complete Gather community calendar v1.0"
```

---

## Summary

This plan implements a Gancio-equivalent community calendar with:

- **18 tasks** across 6 phases
- **PocketBase** backend with custom Go hooks
- **Preact** frontend embedded in single binary
- **ActivityPub** outbound federation
- **RSS/ICS** feeds
- **OSM-based** place search
- **Anonymous + registered** event submission
- **Moderation** support

Estimated implementation: Tasks are bite-sized (2-5 minutes each step).
