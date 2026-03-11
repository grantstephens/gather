# Gather - Community Calendar

A community calendar application built with Go (PocketBase backend) and Preact (frontend). Features event submission, moderation workflows, ActivityPub federation, RSS/iCal feeds, and location-based events using OSM data.

## Quick Start

```bash
make dev
```

This will:
- Build the backend
- Start the Vite dev server (frontend)
- Start the backend server (proxies to Vite)
- Create admin user
- Seed dummy data
- Create test users

**URLs:**
- Backend: http://127.0.0.1:8090
- Frontend (Vite): http://localhost:5173
- Admin dashboard: http://127.0.0.1:8090/_/

**Default credentials:**
- Admin: admin@example.com / adminpassword123

## Development

### Backend

```bash
make build-backend   # Build Go binary (includes embedded frontend)
make dev-backend     # Run backend in dev mode (proxies to Vite at :5173)
make run             # Start server (assumes already built)
```

### Frontend

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

### Docker

#### Using Published Images (GHCR)

```bash
# Pull and run the latest image
docker run -d \
  --name gather \
  -p 8090:8090 \
  -v gather-data:/app/pb_data \
  ghcr.io/[owner]/gather:latest

# Or use a specific version (short SHA)
docker run -d \
  --name gather \
  -p 8090:8090 \
  -v gather-data:/app/pb_data \
  ghcr.io/[owner]/gather:sha-abc1234
```

#### Building Locally

```bash
make docker-build  # Build Docker image
make docker-run    # Run container with persistent volume
make docker-stop   # Stop and remove container
```

**Note:** Replace `[owner]` with your GitHub username or organization name.

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run specific test suites
make test-unit        # Go unit tests
make test-api         # Go API integration tests
make test-e2e         # Playwright E2E tests
make test-e2e-ui      # Interactive E2E test UI
```

### Test Coverage

```bash
make test-coverage    # Generate coverage report
```

See `tests/README.md` for detailed testing documentation.

## Architecture

### Backend (Go + PocketBase)

- **main.go**: Entry point, registers routes and hooks
- **internal/**: Core business logic
  - `hooks/`: PocketBase lifecycle hooks (users, moderation, events)
  - `activitypub/`: Federation support
  - `rss/`, `ical/`: Feed generation
  - `recurrence/`: Recurring event logic
- **migrations/**: Database schema definitions

### Frontend (Preact + TypeScript)

- **src/pages/**: Route components (Home, Event, Submit, Edit, etc.)
- **src/components/**: Reusable UI components
- **src/lib/**: PocketBase client, theme utilities
- **src/app.tsx**: App shell with routing

### Data Model

**Collections:**
- `users`: Role-based access (user/editor/admin)
- `events`: Support draft/pending/published/cancelled states
- `places`: OSM integration, moderation workflow
- `tags`: Moderation workflow
- `settings`: Instance configuration
- `ap_followers`, `ap_delivery_queue`: ActivityPub infrastructure

## Features

- Event submission and moderation
- Role-based access control (user/editor/admin)
- ActivityPub federation
- RSS and iCal feeds
- Location-based events with OSM data
- Recurring events
- Dark/light theme
- Markdown support

## License

[Your license here]
