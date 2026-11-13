# Contributing to Gather

## Getting Started

### Prerequisites

- Go 1.27+
- Node.js 22.12+
- libwebp-dev (for image processing)
- Python 3 (for seed scripts)

### Setup

```bash
git clone https://github.com/grantstephens/gather.git
cd gather
make dev
```

This builds the backend, starts Vite (frontend hot-reload) and the Go server, creates an admin user, and seeds test data.

### URLs

| URL | Purpose |
|-----|---------|
| http://127.0.0.1:8090 | Backend (proxies frontend to Vite) |
| http://localhost:5173 | Vite dev server (direct) |
| http://127.0.0.1:8090/_/ | PocketBase admin dashboard |

### Test Accounts

| Email | Password | Role |
|-------|----------|------|
| admin@example.com | adminpassword123 | Admin |
| editor@example.com | editorpassword123 | Editor |
| user@example.com | userpassword123 | User |

## Development Commands

### Build

```bash
make build-backend    # Go binary (embeds frontend)
make build-frontend   # Frontend only
make build            # Both
```

### Run

```bash
make dev              # Full dev environment with hot reload
make dev-backend      # Backend only (assumes Vite running separately)
make dev-frontend     # Vite only
make run              # Start server (assumes already built)
```

### Data

```bash
make setup-admin      # Create admin user (idempotent)
make seed             # Seed test data (requires running server)
make reset            # Delete database and build artifacts
```

### Testing

```bash
make test             # All tests (unit, API, E2E)
make test-unit        # Go unit tests
make test-api         # API integration tests
make test-e2e         # Playwright E2E tests (headless)
make test-e2e-ui      # Playwright interactive UI
make test-coverage    # Coverage report
make test-frontend    # Frontend unit tests (Vitest)
```

See `tests/README.md` for more details.

### Formatting and linting

Go code must be [gofumpt](https://github.com/mvdan/gofumpt)-formatted and pass golangci-lint; CI fails otherwise.

```bash
make fmt              # gofumpt (go install mvdan.cc/gofumpt@v0.9.2)
make lint             # golangci-lint, config in .golangci.yml (v2.15.0)
```

Shipped migrations under `migrations/` are exempt: never edit them.

### Docker

```bash
make docker-build     # Build image locally
make docker-run       # Run container
make docker-stop      # Stop container
```

## Pull Requests and CI

- `main` requires the **build** (`go test -race ./...`) and **golangci-lint** checks.
- Dependabot patch/minor PRs auto-merge once those pass; major bumps wait for a manual check.
- **Migrations:** number a new one higher than every migration on `origin/main` (`git fetch origin`, then `git ls-tree --name-only origin/main migrations/ | sort | tail -3`), keep it idempotent, and never edit one that has shipped.

## Architecture

### Backend (Go + PocketBase)

- **main.go** — entry point, custom routes, hooks, frontend embedding
- **internal/hooks/** — PocketBase lifecycle hooks (users, moderation, image processing)
- **internal/activitypub/** — federation (actor, inbox, outbox, delivery, keys)
- **internal/rss/**, **internal/ical/** — feed generation
- **internal/recurrence/** — recurring event expansion
- **internal/newsletter/** — "What's on this weekend" email (signup, scheduling, rendering)
- **internal/seo/** — server-rendered HTML for crawlers, JSON-LD, sitemap
- **internal/cron/** — scheduled jobs
- **internal/api/**, **internal/middleware/** — custom routes and rate limiting
- **internal/testutil/** — migrated PocketBase template shared by the test suites
- **migrations/** — database schema (PocketBase collections)

### Frontend (Preact + TypeScript + Vite)

- **src/app.tsx** — app shell, routing, header/footer
- **src/pages/** — route components (Home, Event, Submit, Edit, Admin, etc.)
- **src/components/** — reusable UI (EventCard, MarkdownEditor, MiniCalendar, etc.)
- **src/lib/pocketbase.ts** — PocketBase client, TypeScript interfaces, auth helpers
- **src/lib/theme.ts** — dark/light mode
- **src/style.css** — design tokens and global styles

### Data Model

| Collection | Purpose |
|------------|---------|
| `users` | Accounts with role field (user/editor/admin) |
| `events` | Status workflow (draft/pending/published/cancelled), recurrence, AP |
| `places` | Locations with OSM data, moderation status |
| `tags` | Categories with moderation status |
| `pages` | Custom static pages (Markdown) |
| `picks` | Editorial Picks posts |
| `settings` | Singleton instance configuration |
| `newsletter_subscribers` | Double opt-in email subscribers |
| `ap_followers`, `ap_delivery_queue` | ActivityPub infrastructure |
| `ap_relay_followers`, `ap_subject_followers`, `ap_actors` | Mobilizon relay, place/town actors and their keys |

### Key Patterns

- **Dev mode**: `DEV=1` env var proxies frontend requests to Vite at :5173
- **Production**: frontend is embedded into the Go binary via `embed.FS`
- **Moderation cascade**: events can't publish if their place/tags are still pending
- **ActivityPub**: hooks in main.go trigger AP delivery when events are published
