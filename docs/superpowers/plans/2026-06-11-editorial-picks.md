# Editorial Picks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an "Editorial Picks" feature allowing editors and admins to publish curated posts spotlighting upcoming events, with a home page teaser, archive page at `/picks`, and individual post pages at `/picks/:slug`.

**Architecture:** New `picks` PocketBase collection with a multi-relation to events. Auto-archiving is query-derived — a post is "current" when any linked event is still upcoming and `hidden = false`. Frontend adds two new Preact pages and a new Admin tab, all following existing patterns. No custom backend routes — pure PocketBase REST.

**Tech Stack:** Go/PocketBase (migration), Preact/TypeScript (frontend), DOMPurify + marked (content rendering), date-fns (date formatting)

---

## File Map

**New files:**
- `migrations/1709300014_picks.go` — picks collection schema
- `frontend/src/pages/Picks.tsx` — `/picks` archive page
- `frontend/src/pages/Picks.css` — styles for archive page
- `frontend/src/pages/PicksPost.tsx` — `/picks/:slug` individual post page
- `frontend/src/pages/PicksPost.css` — styles for individual post page

**Modified files:**
- `frontend/src/lib/pocketbase.ts` — add `PicksRecord` interface
- `frontend/src/app.tsx` — add lazy imports and routes
- `frontend/src/pages/Home.tsx` — add picks teaser section
- `frontend/src/pages/Home.css` — teaser styles
- `frontend/src/pages/Admin.tsx` — add Picks tab

---

### Task 1: Database migration

**Files:**
- Create: `migrations/1709300014_picks.go`

- [ ] **Step 1: Create the migration file**

```go
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("events")
		if err != nil {
			return err
		}

		picks := core.NewBaseCollection("picks")

		picks.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
		})
		picks.Fields.Add(&core.TextField{
			Name:     "slug",
			Required: true,
		})
		picks.Fields.Add(&core.EditorField{
			Name: "blurb",
		})
		picks.Fields.Add(&core.RelationField{
			Name:         "events",
			CollectionId: events.Id,
			MaxSelect:    999,
		})
		picks.Fields.Add(&core.BoolField{
			Name: "hidden",
		})

		picks.Indexes = []string{
			"CREATE UNIQUE INDEX idx_picks_slug ON picks (slug)",
		}

		publicRule := ""
		editorRule := `@request.auth.role = 'admin' || @request.auth.role = 'editor'`

		picks.ListRule = &publicRule
		picks.ViewRule = &publicRule
		picks.CreateRule = &editorRule
		picks.UpdateRule = &editorRule
		picks.DeleteRule = &editorRule

		return app.Save(picks)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("picks")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
```

- [ ] **Step 2: Verify it builds**

Run: `make build-backend`
Expected: exits 0, no errors

- [ ] **Step 3: Commit**

```bash
git add migrations/1709300014_picks.go
git commit -m "feat: add picks collection migration"
```

---

### Task 2: TypeScript interface

**Files:**
- Modify: `frontend/src/lib/pocketbase.ts` (after line 88, end of `PageRecord`)

- [ ] **Step 1: Add PicksRecord interface**

After the closing `}` of `PageRecord` in `frontend/src/lib/pocketbase.ts`, add:

```typescript
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

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd frontend && npm run build`
Expected: exits 0

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/pocketbase.ts
git commit -m "feat: add PicksRecord TypeScript interface"
```

---

### Task 3: Individual post page

**Files:**
- Create: `frontend/src/pages/PicksPost.css`
- Create: `frontend/src/pages/PicksPost.tsx`

- [ ] **Step 1: Create PicksPost.css**

```css
.picks-post {
  max-width: 720px;
  margin: 0 auto;
  padding: var(--space-8) var(--space-4);
}

.picks-post-header {
  margin-bottom: var(--space-8);
}

.picks-post-header h1 {
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  letter-spacing: -0.025em;
  color: var(--color-text);
  margin: 0 0 var(--space-4);
}

.picks-post-blurb {
  font-size: var(--text-lg);
  color: var(--color-text-secondary);
  line-height: 1.7;
}

.picks-post-blurb p { margin: 0; }

.picks-post-events {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
```

- [ ] **Step 2: Create PicksPost.tsx**

```tsx
import { useEffect, useState } from 'preact/hooks'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { pb, PicksRecord, getImageUrl } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import './PicksPost.css'

interface Props {
  path?: string
  slug?: string
}

function setMetaTag(attr: string, value: string, content: string) {
  let el = document.querySelector(`meta[${attr}="${value}"]`)
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute(attr, value)
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

export function PicksPost({ slug }: Props) {
  const [post, setPost] = useState<PicksRecord | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!slug) {
      setLoading(false)
      return
    }
    async function load() {
      try {
        const record = await pb.collection('picks').getFirstListItem<PicksRecord>(
          pb.filter('slug = {:slug}', { slug }),
          { expand: 'events,events.place,events.tags' }
        )
        setPost(record)

        document.title = record.title
        const plainBlurb = record.blurb.replace(/<[^>]*>/g, '').slice(0, 160)
        setMetaTag('name', 'description', plainBlurb)
        setMetaTag('property', 'og:title', record.title)
        setMetaTag('property', 'og:description', plainBlurb)

        const firstEvent = record.expand?.events?.[0]
        if (firstEvent) {
          const imageUrl = getImageUrl(firstEvent, '800x600')
          if (imageUrl) setMetaTag('property', 'og:image', imageUrl)
        }
      } catch {
        setPost(null)
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [slug])

  if (loading) return <div class="loading">Loading...</div>

  if (!post) {
    return (
      <div class="page-not-found">
        <h1>Post not found</h1>
        <p><a href="/picks">Back to Picks</a></p>
      </div>
    )
  }

  return (
    <article class="picks-post">
      <header class="picks-post-header">
        <h1>{post.title}</h1>
        {post.blurb && (
          <div
            class="picks-post-blurb"
            dangerouslySetInnerHTML={{
              __html: DOMPurify.sanitize(marked.parse(post.blurb) as string)
            }}
          />
        )}
      </header>
      {post.expand?.events && post.expand.events.length > 0 && (
        <div class="picks-post-events">
          {post.expand.events.map(event => (
            <EventCard key={event.id} event={event} variant="featured" />
          ))}
        </div>
      )}
    </article>
  )
}
```

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd frontend && npm run build`
Expected: exits 0

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/PicksPost.tsx frontend/src/pages/PicksPost.css
git commit -m "feat: add PicksPost individual post page with SEO"
```

---

### Task 4: Archive page

**Files:**
- Create: `frontend/src/pages/Picks.css`
- Create: `frontend/src/pages/Picks.tsx`

- [ ] **Step 1: Create Picks.css**

```css
.picks-page {
  max-width: 720px;
  margin: 0 auto;
  padding: var(--space-8) var(--space-4);
}

.picks-page > h1 {
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  letter-spacing: -0.025em;
  color: var(--color-text);
  margin: 0 0 var(--space-8);
}

.picks-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-10);
}

.picks-item {
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-6);
}

.picks-item--current {
  border-top-color: var(--color-accent);
}

.picks-badge {
  display: inline-block;
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-accent);
  background: var(--color-accent-subtle);
  padding: 2px 8px;
  border-radius: 9999px;
  margin-bottom: var(--space-2);
}

.picks-item-header h2 {
  font-size: var(--text-xl);
  font-weight: var(--font-bold);
  margin: 0 0 var(--space-3);
}

.picks-item-header h2 a {
  color: var(--color-text);
  text-decoration: none;
}

.picks-item-header h2 a:hover {
  color: var(--color-accent);
}

.picks-item-blurb {
  font-size: var(--text-base);
  color: var(--color-text-secondary);
  line-height: 1.6;
}

.picks-item-blurb p { margin: 0; }

.picks-item-events {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-top: var(--space-4);
}
```

- [ ] **Step 2: Create Picks.tsx**

```tsx
import { useEffect, useState } from 'preact/hooks'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { pb, PicksRecord } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import './Picks.css'

interface Props {
  path?: string
}

function isCurrentPost(post: PicksRecord): boolean {
  if (post.hidden) return false
  const now = new Date().toISOString().replace('T', ' ').slice(0, 19)
  return (post.expand?.events ?? []).some(e => {
    const cutoff = e.end_datetime ?? e.start_datetime
    return cutoff >= now
  })
}

export function Picks(_props: Props) {
  const [posts, setPosts] = useState<PicksRecord[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    document.title = 'Picks'
    async function load() {
      try {
        const records = await pb.collection('picks').getFullList<PicksRecord>({
          sort: '-created',
          expand: 'events,events.place,events.tags',
        })
        setPosts(records)
      } catch {
        // show empty state on failure
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  if (loading) return <div class="loading">Loading...</div>

  return (
    <div class="picks-page">
      <h1>Picks</h1>
      {posts.length === 0 ? (
        <p class="no-events">No picks yet.</p>
      ) : (
        <div class="picks-list">
          {posts.map(post => {
            const current = isCurrentPost(post)
            return (
              <article key={post.id} class={`picks-item${current ? ' picks-item--current' : ''}`}>
                <header class="picks-item-header">
                  {current && <span class="picks-badge">This Weekend</span>}
                  <h2><a href={`/picks/${post.slug}`}>{post.title}</a></h2>
                  {post.blurb && (
                    <div
                      class="picks-item-blurb"
                      dangerouslySetInnerHTML={{
                        __html: DOMPurify.sanitize(marked.parse(post.blurb) as string)
                      }}
                    />
                  )}
                </header>
                {post.expand?.events && post.expand.events.length > 0 && (
                  <div class="picks-item-events">
                    {post.expand.events.map(event => (
                      <EventCard key={event.id} event={event} variant="compact" />
                    ))}
                  </div>
                )}
              </article>
            )
          })}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd frontend && npm run build`
Expected: exits 0

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/Picks.tsx frontend/src/pages/Picks.css
git commit -m "feat: add Picks archive page"
```

---

### Task 5: Home page teaser

**Files:**
- Modify: `frontend/src/pages/Home.tsx`
- Modify: `frontend/src/pages/Home.css`

- [ ] **Step 1: Update import in Home.tsx**

Replace line 2 of `frontend/src/pages/Home.tsx`:
```typescript
import { pb, Event, Tag } from '../lib/pocketbase'
```
with:
```typescript
import { pb, Event, Tag, PicksRecord } from '../lib/pocketbase'
```

- [ ] **Step 2: Add picks state**

After `const today = new Date().toISOString().split('T')[0]` (line 32 of Home.tsx), add:
```typescript
const [currentPicks, setCurrentPicks] = useState<PicksRecord | null>(null)
```

- [ ] **Step 3: Add picks fetch effect**

After the sidebar metadata `useEffect` (the one ending around line 96 with `return () => controller.abort()`), add:
```typescript
useEffect(() => {
  const now = new Date().toISOString().replace('T', ' ').slice(0, 19)
  pb.collection('picks').getList<PicksRecord>(1, 1, {
    filter: pb.filter(
      'hidden = false && (events.end_datetime >= {:now} || events.start_datetime >= {:now})',
      { now }
    ),
    sort: '-created',
    fields: 'id,title,slug,blurb',
  }).then(result => {
    setCurrentPicks(result.items[0] ?? null)
  }).catch(() => {})
}, [])
```

- [ ] **Step 4: Add teaser to the render output**

In the `return (...)` block, inside `<div class="home-main">`, add the teaser after `{mobileFilterBar}` and before `<div class="events-header">`:

```tsx
{currentPicks && (
  <div class="picks-teaser">
    <div class="picks-teaser-label">Editor's Picks</div>
    <h2 class="picks-teaser-title">
      <a href={`/picks/${currentPicks.slug}`}>{currentPicks.title}</a>
    </h2>
    {currentPicks.blurb && (
      <p class="picks-teaser-blurb">
        {currentPicks.blurb.replace(/<[^>]*>/g, '').slice(0, 200)}
      </p>
    )}
    <a href="/picks" class="picks-teaser-link">See all picks →</a>
  </div>
)}
```

- [ ] **Step 5: Add teaser styles to Home.css**

Append to `frontend/src/pages/Home.css`:
```css
/* ── Picks teaser ──────────────────────────────── */
.picks-teaser {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-left: 3px solid var(--color-accent);
  border-radius: var(--radius-lg, 8px);
  padding: var(--space-4) var(--space-5);
  margin-bottom: var(--space-6);
}

.picks-teaser-label {
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-accent);
  margin-bottom: var(--space-2);
}

.picks-teaser-title {
  font-size: var(--text-xl);
  font-weight: var(--font-bold);
  margin: 0 0 var(--space-2);
}

.picks-teaser-title a {
  color: var(--color-text);
  text-decoration: none;
}

.picks-teaser-title a:hover { color: var(--color-accent); }

.picks-teaser-blurb {
  font-size: var(--text-sm);
  color: var(--color-text-secondary);
  margin: 0 0 var(--space-3);
  line-height: 1.6;
}

.picks-teaser-link {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-accent);
  text-decoration: none;
}

.picks-teaser-link:hover { text-decoration: underline; }
```

- [ ] **Step 6: Verify TypeScript compiles**

Run: `cd frontend && npm run build`
Expected: exits 0

- [ ] **Step 7: Commit**

```bash
git add frontend/src/pages/Home.tsx frontend/src/pages/Home.css
git commit -m "feat: add picks teaser to home page"
```

---

### Task 6: Routing

**Files:**
- Modify: `frontend/src/app.tsx`

- [ ] **Step 1: Add lazy imports**

In `frontend/src/app.tsx`, after line 27 (`const Page = lazy(...)`), add:
```typescript
const Picks = lazy(() => import('./pages/Picks').then(m => ({ default: m.Picks })))
const PicksPost = lazy(() => import('./pages/PicksPost').then(m => ({ default: m.PicksPost })))
```

- [ ] **Step 2: Add routes**

In the `<Router>` block (around line 218), add the two new routes before `<Page path="/:slug" />`:
```tsx
<Picks path="/picks" />
<PicksPost path="/picks/:slug" />
```

The full Router block becomes:
```tsx
<Router>
  <Home path="/" />
  <Event path="/event/:id" />
  <Submit path="/submit" />
  <Tag path="/tag/:name" />
  <Place path="/place/:id" />
  <Login path="/login" />
  <Admin path="/admin" />
  <Edit path="/edit/:id" />
  <Search path="/search" />
  <Picks path="/picks" />
  <PicksPost path="/picks/:slug" />
  <Page path="/:slug" />
</Router>
```

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd frontend && npm run build`
Expected: exits 0

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app.tsx
git commit -m "feat: add /picks and /picks/:slug routes"
```

---

### Task 7: Admin Picks tab

**Files:**
- Modify: `frontend/src/pages/Admin.tsx`

- [ ] **Step 1: Update imports, TabType, and RESERVED_SLUGS**

Change line 5 from:
```typescript
import type { PageRecord } from '../lib/pocketbase'
```
to:
```typescript
import type { PageRecord, PicksRecord } from '../lib/pocketbase'
```

Change line 18 from:
```typescript
type TabType = 'events' | 'places' | 'tags' | 'settings' | 'pages'
```
to:
```typescript
type TabType = 'events' | 'places' | 'tags' | 'settings' | 'pages' | 'picks'
```

Change line 20 from:
```typescript
const RESERVED_SLUGS = ['submit', 'login', 'admin', 'event', 'tag', 'place', 'edit']
```
to:
```typescript
const RESERVED_SLUGS = ['submit', 'login', 'admin', 'event', 'tag', 'place', 'edit', 'picks']
```

- [ ] **Step 2: Add picks state**

After the `pageSaving` state declaration (around line 44), add:
```typescript
const [picks, setPicks] = useState<PicksRecord[]>([])
const [picksLoaded, setPicksLoaded] = useState(false)
const [showPicksForm, setShowPicksForm] = useState(false)
const [editingPicksId, setEditingPicksId] = useState<string | null>(null)
const [picksForm, setPicksForm] = useState({
  title: '',
  slug: '',
  blurb: '',
  events: [] as string[],
  hidden: false,
})
const [picksFormError, setPicksFormError] = useState<string | null>(null)
const [picksSaving, setPicksSaving] = useState(false)
const [picksEventSearch, setPicksEventSearch] = useState('')
```

- [ ] **Step 3: Add picks load effect**

After the pages `useEffect` block (the one that ends around line 110), add:
```typescript
useEffect(() => {
  if (activeTab !== 'picks' || picksLoaded) return
  async function loadPicks() {
    try {
      const records = await pb.collection('picks').getFullList<PicksRecord>({
        sort: '-created',
        expand: 'events',
      })
      setPicks(records)
    } catch (err) {
      console.error('Failed to load picks:', err)
    } finally {
      setPicksLoaded(true)
    }
  }
  loadPicks()
}, [activeTab, picksLoaded])
```

- [ ] **Step 4: Add picks handlers**

After `handlePageMove` (around line 317), add:
```typescript
const handlePicksNew = () => {
  setEditingPicksId(null)
  setPicksForm({ title: '', slug: '', blurb: '', events: [], hidden: false })
  setPicksFormError(null)
  setPicksEventSearch('')
  setShowPicksForm(true)
}

const handlePicksEdit = (post: PicksRecord) => {
  setEditingPicksId(post.id)
  setPicksForm({
    title: post.title,
    slug: post.slug,
    blurb: post.blurb,
    events: post.events ?? [],
    hidden: post.hidden,
  })
  setPicksFormError(null)
  setPicksEventSearch('')
  setShowPicksForm(true)
}

const handlePicksSave = async () => {
  if (!picksForm.title.trim() || !picksForm.slug.trim()) {
    setPicksFormError('Title and slug are required.')
    return
  }
  setPicksSaving(true)
  setPicksFormError(null)
  try {
    if (editingPicksId) {
      const updated = await pb.collection('picks').update<PicksRecord>(
        editingPicksId,
        picksForm,
        { expand: 'events' }
      )
      setPicks(prev => prev.map(p => p.id === editingPicksId ? updated : p))
    } else {
      const created = await pb.collection('picks').create<PicksRecord>(
        picksForm,
        { expand: 'events' }
      )
      setPicks(prev => [created, ...prev])
    }
    setShowPicksForm(false)
  } catch (err: any) {
    setPicksFormError(err?.data?.data?.slug?.message || 'Failed to save post.')
  } finally {
    setPicksSaving(false)
  }
}

const handlePicksDelete = async (postId: string) => {
  if (!confirm('Delete this picks post? This cannot be undone.')) return
  try {
    await pb.collection('picks').delete(postId)
    setPicks(prev => prev.filter(p => p.id !== postId))
  } catch {
    alert('Failed to delete post.')
  }
}

const handlePicksToggleHidden = async (post: PicksRecord) => {
  try {
    const updated = await pb.collection('picks').update<PicksRecord>(post.id, { hidden: !post.hidden })
    setPicks(prev => prev.map(p => p.id === post.id ? { ...p, hidden: updated.hidden } : p))
  } catch {
    alert('Failed to update post.')
  }
}

const handlePicksEventToggle = (eventId: string) => {
  setPicksForm(f => ({
    ...f,
    events: f.events.includes(eventId)
      ? f.events.filter(id => id !== eventId)
      : [...f.events, eventId],
  }))
}
```

- [ ] **Step 5: Add the Picks tab button**

In the `admin-tabs` div, after the Pages tab button block (around line 373), add:
```tsx
{canModerate() && (
  <button
    class={`tab ${activeTab === 'picks' ? 'active' : ''}`}
    onClick={() => setActiveTab('picks')}
  >
    Picks
  </button>
)}
```

- [ ] **Step 6: Add picks tab content**

After `{activeTab === 'settings' && <SettingsForm />}` (around line 542), add:
```tsx
{activeTab === 'picks' && (
  <div class="pages-admin">
    {!showPicksForm ? (
      <>
        <div class="pages-list-header">
          <button class="btn btn-primary" onClick={handlePicksNew}>New Post</button>
        </div>
        {picks.length === 0 ? (
          <p class="no-events">No picks posts yet.</p>
        ) : (
          <div class="items-list">
            {picks.map(post => (
              <div key={post.id} class="admin-item-card">
                <div class="item-info">
                  <h3>{post.title}</h3>
                  <p class="item-detail">
                    /picks/{post.slug}
                    {' · '}
                    {post.expand?.events?.length ?? 0} event{(post.expand?.events?.length ?? 0) !== 1 ? 's' : ''}
                    {post.hidden && ' · hidden'}
                  </p>
                </div>
                <div class="admin-event-actions">
                  <button class="btn btn-secondary" onClick={() => handlePicksToggleHidden(post)}>
                    {post.hidden ? 'Unhide' : 'Hide'}
                  </button>
                  <a href={`/picks/${post.slug}`} target="_blank" class="btn btn-secondary">View</a>
                  <button class="btn btn-secondary" onClick={() => handlePicksEdit(post)}>Edit</button>
                  <button class="btn btn-danger" onClick={() => handlePicksDelete(post.id)}>Delete</button>
                </div>
              </div>
            ))}
          </div>
        )}
      </>
    ) : (
      <div class="page-form">
        <h2>{editingPicksId ? 'Edit Post' : 'New Post'}</h2>
        {picksFormError && <div class="error">{picksFormError}</div>}
        <div class="form-group">
          <label for="picks-title">Title</label>
          <input
            type="text"
            id="picks-title"
            value={picksForm.title}
            onInput={(e) => {
              const title = (e.target as HTMLInputElement).value
              setPicksForm(f => ({
                ...f,
                title,
                slug: editingPicksId ? f.slug : slugify(title),
              }))
            }}
            disabled={picksSaving}
            required
          />
        </div>
        <div class="form-group">
          <label for="picks-slug">Slug (URL path)</label>
          <input
            type="text"
            id="picks-slug"
            value={picksForm.slug}
            onInput={(e) => setPicksForm(f => ({ ...f, slug: (e.target as HTMLInputElement).value }))}
            disabled={picksSaving}
            required
          />
          <small>Post will be at /picks/{picksForm.slug}</small>
        </div>
        <div class="form-group">
          <label>Blurb</label>
          <MarkdownEditor
            value={picksForm.blurb}
            onChange={(blurb) => setPicksForm(f => ({ ...f, blurb }))}
          />
        </div>
        <div class="form-group">
          <label>Featured Events</label>
          <input
            type="text"
            placeholder="Search events by title..."
            value={picksEventSearch}
            onInput={(e) => setPicksEventSearch((e.target as HTMLInputElement).value)}
          />
          <div style={{ marginTop: 'var(--space-2)', maxHeight: '300px', overflowY: 'auto', border: '1px solid var(--color-border)', borderRadius: '4px', padding: 'var(--space-2)' }}>
            {allEvents
              .filter(e =>
                e.status === 'published' &&
                e.title.toLowerCase().includes(picksEventSearch.toLowerCase())
              )
              .map(event => (
                <label key={event.id} style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)', padding: '4px 0', cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={picksForm.events.includes(event.id)}
                    onChange={() => handlePicksEventToggle(event.id)}
                  />
                  <span>{event.title}</span>
                  <span style={{ color: 'var(--color-text-muted)', fontSize: 'var(--text-sm)' }}>
                    {new Date(event.start_datetime).toLocaleDateString()}
                  </span>
                </label>
              ))
            }
          </div>
          <small>{picksForm.events.length} event{picksForm.events.length !== 1 ? 's' : ''} selected</small>
        </div>
        <div class="form-group">
          <label>
            <input
              type="checkbox"
              checked={picksForm.hidden}
              onChange={(e) => setPicksForm(f => ({ ...f, hidden: (e.target as HTMLInputElement).checked }))}
              disabled={picksSaving}
            />
            {' '}Hidden (force-archive this post)
          </label>
        </div>
        <div class="form-actions">
          <button type="button" class="btn btn-secondary" onClick={() => setShowPicksForm(false)} disabled={picksSaving}>
            Cancel
          </button>
          <button type="button" class="btn btn-primary" onClick={handlePicksSave} disabled={picksSaving}>
            {picksSaving ? 'Saving...' : 'Save Post'}
          </button>
        </div>
      </div>
    )}
  </div>
)}
```

- [ ] **Step 7: Verify TypeScript compiles**

Run: `cd frontend && npm run build`
Expected: exits 0

- [ ] **Step 8: Commit**

```bash
git add frontend/src/pages/Admin.tsx
git commit -m "feat: add Picks admin tab"
```

---

### Task 8: Manual verification

- [ ] **Step 1: Start the dev server**

Run: `make dev`
Expected: Vite starts at http://localhost:5173, backend at http://127.0.0.1:8090

- [ ] **Step 2: Confirm migration applied**

Navigate to http://127.0.0.1:8090/_/, log in with admin@example.com / adminpassword123.
Expected: `picks` collection exists under Collections with fields: title, slug, blurb, events (relation to events), hidden.

- [ ] **Step 3: Create a test picks post**

Navigate to http://localhost:5173/admin → Picks tab → New Post.
Fill in: title "Weekend Picks: Test", blurb "Some test blurb text", select at least one published event, leave hidden unchecked. Save.
Expected: post appears in the list with correct event count.

- [ ] **Step 4: Verify the individual post page**

Navigate to http://localhost:5173/picks/weekend-picks-test (or whatever slug was generated).
Expected: title renders as `<h1>`, blurb renders below it, event card(s) appear with image/title/date/place. Browser tab title shows the post title.

- [ ] **Step 5: Verify the archive page**

Navigate to http://localhost:5173/picks.
Expected: post list renders. The test post has a "This Weekend" badge (since its event is upcoming).

- [ ] **Step 6: Verify the home page teaser**

Navigate to http://localhost:5173/.
Expected: "Editor's Picks" teaser appears above the event timeline, showing the post title, blurb (plain text), and "See all picks →" link.

- [ ] **Step 7: Verify the hidden override**

In the Admin Picks tab, click "Hide" on the test post. Reload http://localhost:5173/.
Expected: teaser disappears. Navigate to /picks — post shows without the "This Weekend" badge.
Click "Unhide" to restore.

- [ ] **Step 8: Final commit if needed**

If any stray changes remain unstaged:
```bash
git status
git add <any remaining files>
git commit -m "feat: editorial picks feature complete"
```
