# Custom Pages Feature Design

**Date:** 2026-03-21

## Overview

Admins can create arbitrary markdown pages (e.g. About, FAQ) via the admin panel. Pages appear as links in the top nav and/or footer based on per-page toggle settings.

## Data Model

New `pages` PocketBase collection. New migration file: next available number after `1709300006_settings_custom_head.go` — verify at implementation time and increment by one (do not assume `7` is still free).

**Fields:**
- `title` — TextField, required
- `slug` — TextField, required, unique index (e.g. `about` → URL `/about`)
- `content` — EditorField (markdown; consistent with how `events.description` is stored)
- `show_in_nav` — BoolField, default `true`
- `show_in_footer` — BoolField, default `true`

**Access rules:**
- ListRule / ViewRule: `""` (public)
- CreateRule / UpdateRule / DeleteRule: `@request.auth.role = "admin"` (double quotes — PocketBase filter syntax)

**Migration:** include both the up function (creates the collection) and a down/rollback function that deletes the collection (consistent with the pattern in migrations `1709300002`–`1709300006`).

## Backend

Single migration file creates the collection with the above fields and access rules. No custom routes needed — the PocketBase REST API handles CRUD.

## Admin UI

- A "Pages" tab added to the Admin panel (`frontend/src/pages/Admin.tsx`), visible only when `user?.role === 'admin'` (not shown to editors).
- The tab only fetches page data when `isAdmin()` returns true (not `canModerate()`), to ensure editor-role users do not trigger a permission error against the pages collection.
- The tab has two states managed by local component state:
  - **List view:** list of all pages with title, slug, nav/footer status, and Edit/Delete actions. Includes a "New Page" button.
  - **Edit/Create form:** inline form with Title, Slug (auto-generated from title as lowercase-hyphenated, user-editable), Content (using the existing `MarkdownEditor` component), Show in Nav checkbox (default checked), Show in Footer checkbox (default checked). Save/Cancel buttons.
- **Reserved slug validation:** the admin UI must reject slugs that conflict with existing frontend routes. Reserved slugs: `submit`, `login`, `admin`, `event`, `tag`, `place`, `edit`. Show a clear error message if the user enters one of these.

## Frontend

**TypeScript interface:** a `PageRecord` interface added to `frontend/src/lib/pocketbase.ts` (named `PageRecord` to avoid collision with the `Page` route component). It extends `BaseModel` to inherit `id`, `created`, and `updated` system fields (consistent with the existing `Settings` interface). Additional fields: `title`, `slug`, `content`, `show_in_nav`, `show_in_footer`.

**New component:** `frontend/src/pages/Page.tsx`
- Receives `slug` as a route param.
- Fetches the page record from PocketBase by slug using `getFirstListItem`.
- Renders the markdown content using the same renderer used by event descriptions.
- Shows a "Page not found" message if no record is found (404 case).
- This component also serves as the app-wide 404 handler — any unrecognised path hits this component and displays the not-found message.

**Routing (`frontend/src/app.tsx`):**
- `<Page path="/:slug" />` added as the last route in the Router so it only matches slugs not claimed by existing routes.

**Nav and Footer (`frontend/src/app.tsx`):**
- On app load, fetch all pages using `getFullList` with `sort: 'created'` (explicit sort for deterministic ordering; pages display in creation order).
- Pages with `show_in_nav = true` are rendered as `<a>` links in the header nav alongside existing links.
- Pages with `show_in_footer = true` are rendered as `<a>` links in the footer alongside existing content.
- Links use the pattern `href="/{slug}"`.
- If the fetch fails, nav/footer render no page links (graceful degradation).

## Error Handling

- Pages collection fetch failure on app load: silently ignored, no page links rendered.
- Unknown slug: `Page.tsx` renders "Page not found."
- Reserved slug entered in admin: inline validation error, form not submitted.

## Out of Scope

- Page ordering/sorting beyond creation order
- Draft/publish workflow for pages
- Editor role access to pages
