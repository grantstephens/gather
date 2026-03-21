# Custom Pages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow admins to create arbitrary markdown pages (e.g. About, FAQ) via the admin panel, with links appearing in the nav and/or footer.

**Architecture:** New `pages` PocketBase collection with a migration; `PageRecord` TypeScript interface in `pocketbase.ts`; a new `Page.tsx` route component that renders markdown by slug; `app.tsx` fetches pages on load and injects nav/footer links; `Admin.tsx` gets a Pages tab (admin-only) with inline list + create/edit form using the existing `MarkdownEditor`.

**Tech Stack:** Go (PocketBase migrations), Preact/TypeScript, `marked` + `DOMPurify` for markdown rendering, Playwright for e2e tests.

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `migrations/1709300007_pages.go` | DB schema for `pages` collection |
| Modify | `frontend/src/lib/pocketbase.ts` | Add `PageRecord` interface |
| Create | `frontend/src/pages/Page.tsx` | Public page viewer route |
| Modify | `frontend/src/app.tsx` | Add route + nav/footer dynamic links |
| Modify | `frontend/src/pages/Admin.tsx` | Add Pages management tab |
| Create | `tests/e2e/helpers/pages.ts` | Test helper to create/delete pages via API |
| Create | `tests/e2e/specs/pages.spec.ts` | E2e tests for the pages feature |

---

### Task 1: Create the `pages` migration

**Files:**
- Create: `migrations/1709300007_pages.go`

> Note: verify that `1709300007` is still the next available number. If `migrations/1709300007_*.go` already exists, use the next free number. The last known migration is `1709300006_settings_custom_head.go`.

- [ ] **Step 1: Check the next available migration number**

```bash
ls migrations/ | sort
```

Expected: last entry should be `1709300006_settings_custom_head.go`. If a `1709300007_*.go` exists, increment accordingly.

- [ ] **Step 2: Create the migration file**

Create `migrations/1709300007_pages.go`:

```go
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		pages := core.NewBaseCollection("pages")

		pages.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
		})
		pages.Fields.Add(&core.TextField{
			Name:     "slug",
			Required: true,
		})
		pages.Fields.Add(&core.EditorField{
			Name: "content",
		})
		pages.Fields.Add(&core.BoolField{
			Name: "show_in_nav",
		})
		pages.Fields.Add(&core.BoolField{
			Name: "show_in_footer",
		})

		pages.Indexes = []string{
			"CREATE UNIQUE INDEX idx_pages_slug ON pages (slug)",
		}

		publicRule := ""
		pages.ListRule = &publicRule
		pages.ViewRule = &publicRule

		adminRule := `@request.auth.role = "admin"`
		pages.CreateRule = &adminRule
		pages.UpdateRule = &adminRule
		pages.DeleteRule = &adminRule

		return app.Save(pages)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("pages")
		if err != nil {
			return nil // already gone
		}
		return app.Delete(col)
	})
}
```

- [ ] **Step 3: Build to verify the migration compiles**

```bash
make build-backend
```

Expected: binary builds with no errors. If you see `undefined: core.EditorField`, check that the PocketBase version in `go.mod` supports it (it is the same type used for `events.description`).

- [ ] **Step 4: Run the migration (start the server once)**

```bash
./gather serve &
sleep 3
kill %1
```

Expected: server starts, runs migration `1709300007`, exits cleanly. Check output for `applying migration`.

- [ ] **Step 5: Commit**

```bash
git add migrations/1709300007_pages.go
git commit -m "feat: add pages collection migration"
```

---

### Task 2: Add `PageRecord` TypeScript interface

**Files:**
- Modify: `frontend/src/lib/pocketbase.ts` (after line 75, after the `Settings` interface)

- [ ] **Step 1: Add the interface**

In `frontend/src/lib/pocketbase.ts`, add after the `Settings` interface (after line 75):

```ts
export interface PageRecord extends BaseModel {
  title: string
  slug: string
  content: string
  show_in_nav: boolean
  show_in_footer: boolean
}
```

`BaseModel` is already imported on line 1 (`import PocketBase, { type BaseModel } from 'pocketbase'`), so no new import is needed.

- [ ] **Step 2: Build to verify no TypeScript errors**

```bash
cd frontend && npm run build
```

Expected: build succeeds with no type errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/pocketbase.ts
git commit -m "feat: add PageRecord TypeScript interface"
```

---

### Task 3: Create the `Page.tsx` route component

**Files:**
- Create: `frontend/src/pages/Page.tsx`

This component fetches a page by slug and renders its markdown content. It also acts as the app-wide 404 handler for unrecognised paths.

- [ ] **Step 1: Create the component**

Create `frontend/src/pages/Page.tsx`:

```tsx
import { useEffect, useState } from 'preact/hooks'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { pb } from '../lib/pocketbase'
import type { PageRecord } from '../lib/pocketbase'

interface Props {
  path?: string
  slug?: string
}

export function Page({ slug }: Props) {
  const [page, setPage] = useState<PageRecord | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!slug) {
      setLoading(false)
      return
    }
    async function load() {
      try {
        const record = await pb.collection('pages').getFirstListItem<PageRecord>(
          `slug = "${slug}"`
        )
        setPage(record)
      } catch {
        setPage(null)
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [slug])

  if (loading) return <div class="loading">Loading...</div>

  if (!page) {
    return (
      <div class="page-not-found">
        <h1>Page not found</h1>
        <p><a href="/">Return to home</a></p>
      </div>
    )
  }

  return (
    <article class="custom-page">
      <h1>{page.title}</h1>
      {page.content && (
        <div
          class="page-content"
          dangerouslySetInnerHTML={{
            __html: DOMPurify.sanitize(marked.parse(page.content) as string)
          }}
        />
      )}
    </article>
  )
}
```

- [ ] **Step 2: Build to verify no TypeScript errors**

```bash
cd frontend && npm run build
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/Page.tsx
git commit -m "feat: add Page component for custom page rendering"
```

---

### Task 4: Wire up routing and nav/footer links in `app.tsx`

**Files:**
- Modify: `frontend/src/app.tsx`

Changes needed:
1. Lazy-import the `Page` component.
2. Fetch pages on app load and store in state.
3. Render nav links for pages with `show_in_nav = true`.
4. Render footer links for pages with `show_in_footer = true`.
5. Add `<Page path="/:slug" />` as the **last** route.

- [ ] **Step 1: Add the lazy import**

In `frontend/src/app.tsx`, add after the `Edit` lazy import (after line 16):

```ts
const Page = lazy(() => import('./pages/Page').then(m => ({ default: m.Page })))
```

- [ ] **Step 2: Add pages state**

In the `App` function body, after the existing `useState` declarations, add:

```ts
const [navPages, setNavPages] = useState<import('./lib/pocketbase').PageRecord[]>([])
const [footerPages, setFooterPages] = useState<import('./lib/pocketbase').PageRecord[]>([])
```

- [ ] **Step 3: Add the pages fetch useEffect**

After the existing `useEffect` that loads settings (after line 83), add:

```ts
useEffect(() => {
  async function loadPages() {
    try {
      const records = await pb.collection('pages').getFullList<import('./lib/pocketbase').PageRecord>({
        sort: 'created',
      })
      setNavPages(records.filter(p => p.show_in_nav))
      setFooterPages(records.filter(p => p.show_in_footer))
    } catch {
      // Graceful degradation: no page links rendered
    }
  }
  loadPages()
}, [])
```

- [ ] **Step 4: Add nav links**

In the nav `<div class="nav-items ...">` block, after the `<a href="/submit"...>` link (after line 148), add:

```tsx
{navPages.map(p => (
  <a key={p.id} href={`/${p.slug}`} onClick={handleNavClick}>{p.title}</a>
))}
```

- [ ] **Step 5: Add footer links**

In `<footer class="app-footer">`, before the fediverse button (before line 180), add:

```tsx
{footerPages.map(p => (
  <a key={p.id} href={`/${p.slug}`} class="footer-page-link">{p.title}</a>
))}
```

- [ ] **Step 6: Add the catch-all route**

In the `<Router>` block, add `<Page path="/:slug" />` as the **last** entry (after `<Edit path="/edit/:id" />`):

```tsx
<Page path="/:slug" />
```

- [ ] **Step 7: Build to verify no TypeScript errors**

```bash
cd frontend && npm run build
```

Expected: build succeeds with no type errors.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/app.tsx
git commit -m "feat: add custom page routing and nav/footer links"
```

---

### Task 5: Add Pages management tab to Admin panel

**Files:**
- Modify: `frontend/src/pages/Admin.tsx`

This is the largest change. We add a Pages tab (visible only to admins) with a list view and an inline create/edit form using `MarkdownEditor`.

- [ ] **Step 1: Update imports in `Admin.tsx`**

Change line 3 from:
```ts
import { pb, Event, Place, Tag, canModerate, eventPath } from '../lib/pocketbase'
```
to:
```ts
import { pb, Event, Place, Tag, canModerate, eventPath, isAdmin } from '../lib/pocketbase'
import type { PageRecord } from '../lib/pocketbase'
import { MarkdownEditor } from '../components/MarkdownEditor'
```

- [ ] **Step 2: Update the `TabType` union**

Change line 12 from:
```ts
type TabType = 'pending-events' | 'pending-places' | 'pending-tags' | 'all-events' | 'settings'
```
to:
```ts
type TabType = 'pending-events' | 'pending-places' | 'pending-tags' | 'all-events' | 'settings' | 'pages'
```

- [ ] **Step 3: Add pages state variables**

After the existing `useState` declarations (after line 21), add:

```tsx
const [pages, setPages] = useState<PageRecord[]>([])
const [pagesLoaded, setPagesLoaded] = useState(false)
const [showPageForm, setShowPageForm] = useState(false)
const [editingPageId, setEditingPageId] = useState<string | null>(null)
const [pageForm, setPageForm] = useState({
  title: '',
  slug: '',
  content: '',
  show_in_nav: true,
  show_in_footer: true,
})
const [pageFormError, setPageFormError] = useState<string | null>(null)
const [pageSaving, setPageSaving] = useState(false)
```

- [ ] **Step 4: Add reserved slugs constant and slug helper**

After the `useState` block, add:

```tsx
const RESERVED_SLUGS = ['submit', 'login', 'admin', 'event', 'tag', 'place', 'edit']

function slugify(title: string): string {
  return title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
}
```

- [ ] **Step 5: Add pages loading useEffect**

After the existing `useEffect` that loads events/places/tags (after line 76), add:

```tsx
useEffect(() => {
  if (activeTab !== 'pages' || !isAdmin() || pagesLoaded) return
  async function loadPages() {
    try {
      const records = await pb.collection('pages').getFullList<PageRecord>({ sort: 'created' })
      setPages(records)
    } catch (err) {
      console.error('Failed to load pages:', err)
    } finally {
      setPagesLoaded(true)
    }
  }
  loadPages()
}, [activeTab, pagesLoaded])
```

- [ ] **Step 6: Add page CRUD handlers**

After the existing `handleRejectTag` handler (after line 176), add:

```tsx
const handlePageNew = () => {
  setEditingPageId(null)
  setPageForm({ title: '', slug: '', content: '', show_in_nav: true, show_in_footer: true })
  setPageFormError(null)
  setShowPageForm(true)
}

const handlePageEdit = (page: PageRecord) => {
  setEditingPageId(page.id)
  setPageForm({
    title: page.title,
    slug: page.slug,
    content: page.content,
    show_in_nav: page.show_in_nav,
    show_in_footer: page.show_in_footer,
  })
  setPageFormError(null)
  setShowPageForm(true)
}

const handlePageSave = async () => {
  if (RESERVED_SLUGS.includes(pageForm.slug)) {
    setPageFormError(`"${pageForm.slug}" is a reserved slug and cannot be used.`)
    return
  }
  if (!pageForm.title.trim() || !pageForm.slug.trim()) {
    setPageFormError('Title and slug are required.')
    return
  }
  setPageSaving(true)
  setPageFormError(null)
  try {
    if (editingPageId) {
      const updated = await pb.collection('pages').update<PageRecord>(editingPageId, pageForm)
      setPages(prev => prev.map(p => p.id === editingPageId ? updated : p))
    } else {
      const created = await pb.collection('pages').create<PageRecord>(pageForm)
      setPages(prev => [...prev, created])
    }
    setShowPageForm(false)
  } catch (err: any) {
    setPageFormError(err?.data?.data?.slug?.message || 'Failed to save page.')
  } finally {
    setPageSaving(false)
  }
}

const handlePageDelete = async (pageId: string) => {
  if (!confirm('Delete this page? This cannot be undone.')) return
  try {
    await pb.collection('pages').delete(pageId)
    setPages(prev => prev.filter(p => p.id !== pageId))
  } catch {
    alert('Failed to delete page.')
  }
}
```

- [ ] **Step 7: Add the Pages tab button**

In the `<div class="admin-tabs">` block, after the Settings tab button (after line 218), add:

```tsx
{isAdmin() && (
  <button
    class={`tab ${activeTab === 'pages' ? 'active' : ''}`}
    onClick={() => setActiveTab('pages')}
  >
    Pages
  </button>
)}
```

- [ ] **Step 8: Add the Pages tab content**

After the `{activeTab === 'settings' && <SettingsForm />}` line (after line 332), add:

```tsx
{activeTab === 'pages' && (
  <div class="pages-admin">
    {!showPageForm ? (
      <>
        <div class="pages-list-header">
          <button class="btn btn-primary" onClick={handlePageNew}>New Page</button>
        </div>
        {pages.length === 0 ? (
          <p class="no-events">No pages yet. Create your first page above.</p>
        ) : (
          <div class="items-list">
            {pages.map(page => (
              <div key={page.id} class="admin-item-card">
                <div class="item-info">
                  <h3>{page.title}</h3>
                  <p class="item-detail">
                    /{page.slug}
                    {' · '}
                    {page.show_in_nav ? 'Nav' : ''}
                    {page.show_in_nav && page.show_in_footer ? ' · ' : ''}
                    {page.show_in_footer ? 'Footer' : ''}
                  </p>
                </div>
                <div class="admin-event-actions">
                  <a href={`/${page.slug}`} target="_blank" class="btn btn-secondary">View</a>
                  <button class="btn btn-secondary" onClick={() => handlePageEdit(page)}>Edit</button>
                  <button class="btn btn-danger" onClick={() => handlePageDelete(page.id)}>Delete</button>
                </div>
              </div>
            ))}
          </div>
        )}
      </>
    ) : (
      <div class="page-form">
        <h2>{editingPageId ? 'Edit Page' : 'New Page'}</h2>
        {pageFormError && <div class="error">{pageFormError}</div>}
        <div class="form-group">
          <label for="page-title">Title</label>
          <input
            type="text"
            id="page-title"
            value={pageForm.title}
            onInput={(e) => {
              const title = (e.target as HTMLInputElement).value
              setPageForm(f => ({
                ...f,
                title,
                slug: editingPageId ? f.slug : slugify(title),
              }))
            }}
            disabled={pageSaving}
            required
          />
        </div>
        <div class="form-group">
          <label for="page-slug">Slug (URL path)</label>
          <input
            type="text"
            id="page-slug"
            value={pageForm.slug}
            onInput={(e) => setPageForm(f => ({ ...f, slug: (e.target as HTMLInputElement).value }))}
            disabled={pageSaving}
            required
          />
          <small>Page will be accessible at /{pageForm.slug}</small>
        </div>
        <div class="form-group">
          <label>Content</label>
          <MarkdownEditor
            value={pageForm.content}
            onChange={(content) => setPageForm(f => ({ ...f, content }))}
          />
        </div>
        <div class="form-group">
          <label>
            <input
              type="checkbox"
              checked={pageForm.show_in_nav}
              onChange={(e) => setPageForm(f => ({ ...f, show_in_nav: (e.target as HTMLInputElement).checked }))}
              disabled={pageSaving}
            />
            {' '}Show in navigation
          </label>
        </div>
        <div class="form-group">
          <label>
            <input
              type="checkbox"
              checked={pageForm.show_in_footer}
              onChange={(e) => setPageForm(f => ({ ...f, show_in_footer: (e.target as HTMLInputElement).checked }))}
              disabled={pageSaving}
            />
            {' '}Show in footer
          </label>
        </div>
        <div class="form-actions">
          <button type="button" class="btn btn-secondary" onClick={() => setShowPageForm(false)} disabled={pageSaving}>
            Cancel
          </button>
          <button type="button" class="btn btn-primary" onClick={handlePageSave} disabled={pageSaving}>
            {pageSaving ? 'Saving...' : 'Save Page'}
          </button>
        </div>
      </div>
    )}
  </div>
)}
```

- [ ] **Step 9: Build to verify no TypeScript errors**

```bash
cd frontend && npm run build
```

Expected: build succeeds with no type errors. Fix any before proceeding.

- [ ] **Step 10: Commit**

```bash
git add frontend/src/pages/Admin.tsx
git commit -m "feat: add Pages management tab to admin panel"
```

---

### Task 6: E2e tests

**Files:**
- Create: `tests/e2e/helpers/pages.ts`
- Create: `tests/e2e/specs/pages.spec.ts`

- [ ] **Step 1: Create the pages test helper**

Create `tests/e2e/helpers/pages.ts`:

```ts
import { request } from '@playwright/test';
import { loginViaAPI } from './auth';
import { TEST_USERS } from '../fixtures/users';

export interface TestPage {
  id: string;
  title: string;
  slug: string;
}

export async function createTestPage(
  authToken: string,
  slug: string = `test-page-${Date.now()}`,
  title: string = 'Test Page',
  options: { show_in_nav?: boolean; show_in_footer?: boolean; content?: string } = {}
): Promise<TestPage> {
  const apiContext = await request.newContext({
    baseURL: 'http://127.0.0.1:8090',
    extraHTTPHeaders: { Authorization: authToken },
  });

  const response = await apiContext.post('/api/collections/pages/records', {
    data: {
      title,
      slug,
      content: options.content ?? '## Hello\n\nThis is a test page.',
      show_in_nav: options.show_in_nav ?? true,
      show_in_footer: options.show_in_footer ?? true,
    },
  });

  if (!response.ok()) {
    throw new Error(`Failed to create test page: ${response.status()} ${await response.text()}`);
  }

  const data = await response.json();
  await apiContext.dispose();
  return { id: data.id, title: data.title, slug: data.slug };
}

export async function deleteTestPage(authToken: string, pageId: string): Promise<void> {
  const apiContext = await request.newContext({
    baseURL: 'http://127.0.0.1:8090',
    extraHTTPHeaders: { Authorization: authToken },
  });
  await apiContext.delete(`/api/collections/pages/records/${pageId}`);
  await apiContext.dispose();
}

export async function getAdminToken(): Promise<string> {
  return loginViaAPI(TEST_USERS.admin.email, TEST_USERS.admin.password);
}
```

- [ ] **Step 2: Create the pages spec**

Create `tests/e2e/specs/pages.spec.ts`:

```ts
import { test, expect } from '@playwright/test';
import { setupAuthenticatedPage } from '../helpers/auth';
import { createTestPage, deleteTestPage, getAdminToken } from '../helpers/pages';

test.describe('Custom Pages', () => {
  let adminToken: string;
  let createdPageIds: string[] = [];

  test.beforeAll(async () => {
    adminToken = await getAdminToken();
  });

  test.afterAll(async () => {
    for (const id of createdPageIds) {
      await deleteTestPage(adminToken, id).catch(() => {});
    }
  });

  test('admin can create a page and it appears in nav and footer', async ({ page }) => {
    const slug = `about-${Date.now()}`;
    const title = 'About Us';

    // Create page via API
    const created = await createTestPage(adminToken, slug, title, {
      show_in_nav: true,
      show_in_footer: true,
      content: '## About\n\nWelcome to our community.',
    });
    createdPageIds.push(created.id);

    // Visit home page and check nav link
    await page.goto('/');
    await expect(page.locator(`nav a[href="/${slug}"]`)).toBeVisible();

    // Check footer link
    await expect(page.locator(`footer a[href="/${slug}"]`)).toBeVisible();
  });

  test('public can view a custom page', async ({ page }) => {
    const slug = `faq-${Date.now()}`;
    const created = await createTestPage(adminToken, slug, 'FAQ', {
      content: '## Frequently Asked Questions\n\nSome answers here.',
    });
    createdPageIds.push(created.id);

    await page.goto(`/${slug}`);
    await expect(page.locator('h1')).toContainText('FAQ');
    await expect(page.locator('.page-content')).toBeVisible();
  });

  test('visiting unknown slug shows page not found', async ({ page }) => {
    await page.goto('/this-slug-does-not-exist-xyz');
    await expect(page.locator('h1')).toContainText('Page not found');
  });

  test('page with show_in_nav=false does not appear in nav', async ({ page }) => {
    const slug = `hidden-nav-${Date.now()}`;
    const created = await createTestPage(adminToken, slug, 'Hidden Nav', {
      show_in_nav: false,
      show_in_footer: true,
    });
    createdPageIds.push(created.id);

    await page.goto('/');
    await expect(page.locator(`nav a[href="/${slug}"]`)).not.toBeVisible();
    await expect(page.locator(`footer a[href="/${slug}"]`)).toBeVisible();
  });

  test('admin sees Pages tab in admin panel', async ({ page }) => {
    await setupAuthenticatedPage(page, 'admin');
    await page.goto('/admin');
    await expect(page.locator('button.tab', { hasText: 'Pages' })).toBeVisible();
  });

  test('editor does not see Pages tab in admin panel', async ({ page }) => {
    await setupAuthenticatedPage(page, 'editor');
    await page.goto('/admin');
    await expect(page.locator('button.tab', { hasText: 'Pages' })).not.toBeVisible();
  });

  test('admin can create a page via the admin UI', async ({ page }) => {
    const slug = `ui-created-${Date.now()}`;
    await setupAuthenticatedPage(page, 'admin');
    await page.goto('/admin');

    // Click Pages tab
    await page.click('button.tab:has-text("Pages")');

    // Click New Page
    await page.click('button:has-text("New Page")');

    // Fill in form
    await page.fill('#page-title', 'UI Created Page');
    // Wait for slug auto-fill
    await expect(page.locator('#page-slug')).toHaveValue('ui-created-page');
    // Override slug with our unique one
    await page.fill('#page-slug', slug);

    // Save
    await page.click('button:has-text("Save Page")');

    // Should return to list and show the page
    await expect(page.locator(`.item-info h3:has-text("UI Created Page")`)).toBeVisible();

    // Clean up: find the created page id via API
    const apiContext = await page.request.newContext({
      baseURL: 'http://127.0.0.1:8090',
      extraHTTPHeaders: { Authorization: adminToken },
    });
    const res = await apiContext.get(`/api/collections/pages/records?filter=slug="${slug}"`);
    const data = await res.json();
    if (data.items?.[0]) createdPageIds.push(data.items[0].id);
    await apiContext.dispose();
  });
});
```

- [ ] **Step 3: Build the backend and run the full app, then run the e2e tests**

```bash
make build-backend && ./gather serve &
sleep 5
cd tests/e2e && npx playwright test specs/pages.spec.ts --reporter=list
kill %1
```

Expected: all 7 tests pass. If a test fails:
- "admin can create" nav/footer test: check that `app.tsx` fetches pages and filters by `show_in_nav`/`show_in_footer`
- "public can view" test: check `Page.tsx` slugify filter syntax
- "page not found" test: check `Page.tsx` catch block sets `page` to null
- "admin sees Pages tab" test: check that `isAdmin()` returns true when user.role === 'admin'

- [ ] **Step 4: Commit**

```bash
git add tests/e2e/helpers/pages.ts tests/e2e/specs/pages.spec.ts
git commit -m "test: add e2e tests for custom pages feature"
```

---

## Final Build Check

- [ ] **Build the full project**

```bash
make build-backend
```

Expected: binary builds cleanly, no errors.
