# Custom Pages Feature Design

**Date:** 2026-03-21

## Overview

Admins can create arbitrary markdown pages (e.g. About, FAQ) via the admin panel. Pages appear as links in the top nav and/or footer based on per-page toggle settings.

## Data Model

New `pages` PocketBase collection. New migration: `migrations/1709300007_pages.go`.

**Fields:**
- `title` — TextField, required
- `slug` — TextField, required, unique index (e.g. `about` → URL `/about`)
- `content` — TextField (markdown)
- `show_in_nav` — BoolField, default `true`
- `show_in_footer` — BoolField, default `true`

**Access rules:**
- ListRule / ViewRule: `""` (public)
- CreateRule / UpdateRule / DeleteRule: `@request.auth.role = 'admin'`

## Backend

Single migration file creates the collection with the above fields and access rules. No custom routes needed — the PocketBase REST API handles CRUD.

## Admin UI

- A "Pages" tab added to the Admin panel (`frontend/src/pages/Admin.tsx`), visible only when `user?.role === 'admin'` (not shown to editors).
- The tab has two states managed by local component state:
  - **List view:** table/list of all pages with title, slug, nav/footer status, and Edit/Delete actions. Includes a "New Page" button.
  - **Edit/Create form:** inline form with Title, Slug (auto-generated from title, user-editable), Content (using the existing `MarkdownEditor` component), Show in Nav checkbox (default checked), Show in Footer checkbox (default checked). Save/Cancel buttons.
- Slug is auto-generated from title (lowercase, hyphens) but remains editable.

## Frontend

**New component:** `frontend/src/pages/Page.tsx`
- Receives `slug` as a route param.
- Fetches the page record from PocketBase by slug.
- Renders the markdown content using the same renderer used by event descriptions.
- Shows a 404 message if no record is found.

**Routing (`frontend/src/app.tsx`):**
- `<Page path="/:slug" />` added as the last route in the Router so it only matches slugs not claimed by existing routes (`/submit`, `/login`, `/admin`, etc.).

**Nav and Footer (`frontend/src/app.tsx`):**
- On app load, fetch all pages from the `pages` collection (public, no auth needed).
- Pages with `show_in_nav = true` are rendered as `<a>` links in the header nav alongside existing links.
- Pages with `show_in_footer = true` are rendered as `<a>` links in the footer alongside existing content.
- Links use the pattern `href="/{slug}"`.

## TypeScript

A `Page` interface added to `frontend/src/lib/pocketbase.ts` matching the collection fields.

## Error Handling

- If the pages collection fetch fails on app load, nav/footer simply render no page links (graceful degradation).
- If a page slug is not found, `Page.tsx` renders a "Page not found" message.

## Out of Scope

- Page ordering/sorting (pages display in creation order)
- Draft/publish workflow for pages
- Editor role access to pages
