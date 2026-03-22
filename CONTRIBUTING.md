# Contributing to Gather

## Getting Started

### Prerequisites

- Go 1.25+
- Node.js 18+
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
```

See `tests/README.md` for more details.

### Docker

```bash
make docker-build     # Build image locally
make docker-run       # Run container
make docker-stop      # Stop container
```

## Architecture

### Backend (Go + PocketBase)

- **main.go** — entry point, custom routes, hooks, frontend embedding
- **internal/hooks/** — PocketBase lifecycle hooks (users, moderation, image processing)
- **internal/activitypub/** — federation (actor, inbox, outbox, delivery, keys)
- **internal/rss/**, **internal/ical/** — feed generation
- **internal/recurrence/** — recurring event expansion
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
| `settings` | Singleton instance configuration |
| `ap_followers`, `ap_delivery_queue` | ActivityPub infrastructure |

### Key Patterns

- **Dev mode**: `DEV=1` env var proxies frontend requests to Vite at :5173
- **Production**: frontend is embedded into the Go binary via `embed.FS`
- **Moderation cascade**: events can't publish if their place/tags are still pending
- **ActivityPub**: hooks in main.go trigger AP delivery when events are published
