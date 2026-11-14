# AGENTS.md

Guidance for coding agents (Claude Code, Codex, etc.) working in this repository. `CLAUDE.md` just points here.

## Project Overview

Gather is a community calendar application built with Go (PocketBase backend) and Preact (frontend). It supports event submission, moderation workflows, ActivityPub federation, RSS/iCal feeds, and location-based events using OSM data.

## Development Commands

### Quick Start
```bash
make dev          # Build backend, start Vite + proxied backend, auto-seed data
```

### Backend Development
```bash
make build-backend   # Build Go binary (includes embedded frontend)
make dev-backend     # Run backend in dev mode (proxies to Vite at :5173)
make run             # Start server (assumes already built)
```

### Frontend Development
```bash
cd frontend && npm run dev     # Or: make dev-frontend
cd frontend && npm run build   # Or: make build-frontend
```

### Data Management
```bash
make setup-admin   # Create admin user (idempotent)
make seed          # Seed dummy data (requires running server)
make reset         # Clean everything including database
```

### Production
```bash
make prod          # Build, setup admin, seed, and start server
```

**Dev URLs:**
- Backend: http://127.0.0.1:8090 (proxies frontend to Vite)
- Vite: http://localhost:5173 (direct)
- Admin dashboard: http://127.0.0.1:8090/_/
- Default credentials: admin@example.com / adminpassword123

### Testing
```bash
go test -race ./...   # what CI runs; use this, not make test-unit (that target swallows failures)
make test-frontend   # Vitest (frontend/src/**/*.test.ts, node environment, no DOM)
make test            # unit + frontend + API + Playwright E2E
```
Go test packages that need a database call `testutil.NewApp()` (`internal/testutil`): it migrates one template per test binary and clones it per test. Each such package needs a `TestMain` that runs `m.Run()`, then `testutil.Cleanup()`, then `os.Exit(code)`. Don't migrate per test; that is what made CI time out under `-race`. Frontend logic worth testing goes in `src/lib/*.ts` as pure functions (see `week.ts`).

### Formatting and linting

Go code must be [gofumpt](https://github.com/mvdan/gofumpt)-formatted (`make fmt`) and pass `make lint` (golangci-lint, config in `.golangci.yml`); the Lint workflow fails otherwise. Shipped migrations under `migrations/` are exempt from both (never edit them).

## Architecture

### Backend Structure

**main.go**: Entry point that:
- Initializes PocketBase app
- Registers custom routes (RSS, iCal, ActivityPub, Webfinger)
- Sets up hooks for business logic
- Embeds frontend assets (production) or proxies to Vite (dev mode via `DEV=1` env var)

**internal/**: Core business logic modules
- `hooks/`: PocketBase lifecycle hooks
  - `users.go`: Enforce role defaults on user registration
  - `moderation.go`: Auto-set status, cascade blocking (events can't publish if place/tags pending)
  - `events.go`: Event hooks (registered from main.go) for AP publishing and the Mobilizon relay; `notifications.go` sends moderator/author emails; `images.go` converts uploads to WebP; `placeslug.go`/`slugs.go` keep slugs unique
- `activitypub/`: Federation support (actor, inbox, outbox, delivery, keys, webfinger)
  - `event.go`: Mobilizon-shaped `Event` serializer and `/ap/events/{id}` (golden-tested against `testdata/mobilizon_event.json`)
  - `relay.go`: `relay@host` actor; Mobilizon follows it to list our events (needs an `Announce`)
  - `schedule.go`: Fediverse posts go out at 09:00 a set number of days before an event (Mobilizon relay stays immediate)
  - `subject*.go`: lazily-created `Group` actors per approved place and per town, each with its own keypair
  - See API.md "ActivityPub Federation" for shapes, lifecycle rules and why each piece exists
- `rss/`, `ical/`: Feed generation
- `recurrence/`: Recurring event logic
- `newsletter/`: "What's on this weekend" email (signup, schedule, rendering)
- `seo/`: Server-rendered HTML for crawlers, JSON-LD, sitemap
- `cron/`: Scheduled jobs (register through the local `addJob` helper so errors are logged)
- `api/`, `middleware/`: Custom API routes; response compression and `Vary` handling
- `images/`, `slug/`, `textutil/`: WebP conversion, slug and text helpers
- `testutil/`: Migrated PocketBase template cloned per test package (use `testutil.NewApp()`, and call `testutil.Cleanup()` from `TestMain`)

**migrations/**: Database schema definitions
Files are `17093000NN_<name>.go`. Don't guess the next number: derive it from `origin/main` as described in the numbering rules below. Never edit a shipped migration. `…000_collections.go` holds the core collections and later files add fields and collections.

**Migration numbering rules (MUST follow; this has gone wrong before):**
- Migrations run in filename order, and production has already applied everything on `origin/main`. A new migration must therefore be numbered **higher than every migration on `origin/main`**, never just higher than the ones on your branch. A branch cut before another PR merged will otherwise sort *before* migrations production has already run (this shipped broken numbering in v1.3.0 and has happened before).
- Pick the number only after `git fetch origin`, as one more than the highest on `origin/main`: `git ls-tree --name-only origin/main migrations/ | sort | tail -3`. Prefix is `17093000NN_`, contiguous, no gaps, no duplicates.
- Re-check before merging and before cutting a release: `git fetch origin`, list `origin/main` migrations again, and if anyone else landed one, **rename yours to the next free number** (and any references to the filename in docs). Do this on every merge of a long-lived branch, not just at creation.
- Never renumber or edit a migration that has shipped in a release. Fix forward with a new, higher-numbered migration instead.
- Migrations must be idempotent (check that a collection/field exists before adding it, and that it is gone before removing it), so a renumber or re-run is a no-op.
- Quick duplicate check: `ls migrations | awk -F_ '{print $1}' | sort | uniq -d` must print nothing.

### Frontend Structure

**Preact SPA** with TypeScript, Vite bundler, and client-side routing (preact-router).

**src/lib/pocketbase.ts**: PocketBase client setup, TypeScript interfaces for all collections, auth helpers
**src/lib/theme.ts**: Dark/light theme utilities
**src/lib/**: Pure helpers with Vitest tests next to them (`week.ts` week-start layout, `color.ts`, `lifecycle.ts`, `newsletter.ts`, `umami.ts`, ...)

**src/pages/**: Route components
- Home, Event, Submit, Submitted, Edit, Tag, Place, Search, Picks, PicksPost, Newsletter, Page, Login, Admin, NotFound

**src/components/**: Reusable UI components (EventCard, MarkdownEditor, MiniCalendar, SettingsForm, PlaceSearch, NewsletterSignup, etc.)

**src/app.tsx**: App shell with header, nav, router, footer with theme toggle. It loads the `settings` record once and passes what pages need as props (e.g. `weekStartsOn` to Home). Preact 11: `useRef` needs an initial value, and pages are `lazy()` routes under `Suspense`.

### Data Model

**Collections:**
- `users`: Has `role` field (user/editor/admin)
- `events`: Has `status` (draft/pending/published/cancelled), supports recurrence, AP federation
- `places`: Has `status` (pending/approved), OSM integration
- `tags`: Has `status` (pending/approved)
- `settings`: Singleton for instance config
- `ap_followers`, `ap_delivery_queue`: ActivityPub infrastructure
- `picks`, `pages`, `newsletter_subscribers`: Editorial picks posts, custom pages, email subscribers
- `ap_relay_followers`, `ap_subject_followers`, `ap_actors`: Mobilizon relay followers, place/town actor followers, and those actors' keypairs

**Access Control:**
- Published events visible to all
- Admins/editors see all events regardless of status
- Authors can update own events if not published
- Only admins/editors can delete events
- Users cannot set their own role (enforced via hooks)
- Places/tags created by non-superusers default to pending status
- Events cannot be published if associated place/tags are not approved

### Key Patterns

**Dev Mode Detection**: `DEV=1` environment variable switches backend to proxy frontend requests to Vite dev server on :5173

**Frontend Embedding**: Production builds embed `frontend/dist/` into Go binary via `embed.go`

**Moderation Workflow**: Places and tags have approval status; events are blocked from publishing if dependencies aren't approved

**ActivityPub Integration**: Events can federate via ActivityPub when published; event hooks (`internal/hooks/events.go`) queue Notes for Fediverse followers and the Mobilizon `Event` shape for relay and place/town followers. Inbox signature checks and the delivery worker are shared by all actors.

**Authentication**: PocketBase handles auth; frontend checks user role for conditional UI/features

## CI, Releases and Dependencies

- **Workflows** (`.github/workflows/`): `build.yml` (`go test -race ./...`, job `build`), `lint.yml` (golangci-lint + gofumpt, job `golangci-lint`), `docker-publish.yml` (multi-arch image to GHCR), `dependabot-automerge.yml`.
- **`main` is protected**: it requires `build` and `golangci-lint`; admins can still push (GitHub then notes a bypass). Prefer a PR when someone else's work is involved.
- **Dependabot** groups updates weekly for gomod, npm (`/frontend`), github-actions and docker. Patch/minor PRs auto-merge once the checks pass; any major in a group holds the whole PR for a manual check (for example, run the app and click through). It does not bump the `go` directive in `go.mod`.
- **Auto-merges don't trigger other workflows** (they use the workflow token), so `docker-publish.yml` also runs weekly and on `workflow_dispatch` to keep `:latest` fresh.
- **Releases**: `gh release create vX.Y.Z --target main` publishes the release, which triggers the Docker build. Image tags have no `v` prefix: `X.Y.Z`, `X.Y`, `sha-<commit>`, `latest`. Before cutting one, re-check migration numbering (see above) and that CI on `main` is green.
- **PRs from first-time contributors** need their workflow runs approved in the Actions tab (or via the API) before the checks start.
