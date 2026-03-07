# Integration Testing Framework Design

**Date:** 2026-03-07
**Status:** Approved

## Overview

Implement a comprehensive integration testing framework for the Gather community calendar application, focusing on end-to-end browser tests with Playwright and HTTP API tests in Go.

## Goals

- Test critical user journeys (login, event creation, moderation workflows)
- Validate frontend/backend integration points
- Ensure role-based permissions work correctly
- Provide fast feedback loop for developers
- Enable CI/CD automation

## Architecture

### Test Layers

1. **E2E Tests (Primary)** - Playwright + TypeScript at `/tests/e2e/`
   - Full browser automation of user flows
   - Tests against running server (localhost:8090)
   - Validates UI rendering, routing, and backend integration

2. **API Integration Tests (Secondary)** - Go HTTP tests at `/internal/api/*_test.go`
   - Direct HTTP endpoint testing using `net/http/httptest`
   - PocketBase API validation
   - Custom routes (RSS, iCal, ActivityPub, Webfinger)

3. **Unit Tests (Existing)** - Pure function tests at `internal/*/test.go`
   - Already established pattern in `internal/seo/`
   - Continue for isolated business logic

### Technology Choices

- **E2E Framework:** Playwright (modern, fast, excellent TypeScript support)
- **API Testing:** Go standard library + PocketBase test utilities
- **Database:** Shared dev database (`pb_data/`) with cleanup hooks
- **Authentication:** Programmatic API login for speed, one UI login test for validation

## Test Structure

```
/tests/
  /e2e/
    /specs/
      auth.spec.ts          # Login, logout, session
      events.spec.ts        # Event CRUD operations
      calendar.spec.ts      # Navigation, filtering
      moderation.spec.ts    # Admin/editor workflows
      feeds.spec.ts         # RSS, iCal generation
    /fixtures/
      users.ts              # Test user data
      events.ts             # Sample event data
    /helpers/
      auth.ts               # Login utilities
      cleanup.ts            # Database cleanup
      setup.ts              # Test data seeding
    playwright.config.ts
    tsconfig.json
    package.json

/internal/
  /api/                     # New package
    handlers_test.go        # General API tests
    events_test.go          # Event endpoints
    places_test.go          # Places endpoints
    tags_test.go            # Tags endpoints
    testhelpers.go          # Shared Go utilities
```

## Data Management

### Database Strategy

- Use existing `pb_data/` dev database
- No isolated test database (allows debugging failed tests)
- Cleanup after each test, not global wipes

### E2E Approach

- Tag test data with unique identifiers (timestamps, UUIDs)
- Delete created records via PocketBase API after each test
- Example: `test-${Date.now()}@example.com`

### API Approach

- Test app instances point to shared `pb_data/`
- Use Go `defer` statements for guaranteed cleanup
- Unique usernames/titles for parallel safety

### Test Users

Pre-seeded persistent test users:
- `test-admin@test.local` / `testpass123` (admin)
- `test-editor@test.local` / `testpass123` (editor)
- `test-user@test.local` / `testpass123` (user)

## Authentication Strategy

### E2E Tests

1. Login via PocketBase API: `POST /api/collections/users/auth-with-password`
2. Store auth token in Playwright storage state (`.auth/admin.json`, etc.)
3. Reuse authenticated context across test suite
4. One UI login test in `auth.spec.ts` validates login form

### API Tests

- Direct PocketBase authentication in Go
- Helper functions: `CreateAuthenticatedUser(role)`, `AuthRequest(token)`
- Pre-built request helpers: `NewAdminRequest()`, `NewEditorRequest()`

## Test Coverage

### E2E Scenarios

**auth.spec.ts:**
- Login with valid/invalid credentials
- Logout clears session
- Protected routes redirect
- Session persistence

**events.spec.ts:**
- Create draft event
- Edit own draft
- Publish event (admin only)
- Delete with confirmation
- Recurring events
- Events with places/tags

**calendar.spec.ts:**
- Month navigation
- Event details view
- Tag/place filtering
- Date selection

**moderation.spec.ts:**
- Approve pending places/tags
- Block event publishing with pending dependencies
- Moderation dashboard

**feeds.spec.ts:**
- RSS/iCal valid output
- Published events in feeds
- Drafts excluded

### API Test Coverage

- Event CRUD + status transitions
- Place/tag CRUD + moderation
- Permission checks per role
- Custom route handlers
- Validation and error cases

### Coverage Goals

- E2E: 80%+ critical user paths
- API: 90%+ custom handlers
- Unit: 100% pure functions

## Makefile Integration

```makefile
test:               # Run all tests
test-unit:          # Go unit tests
test-api:           # Go API integration tests
test-e2e:           # Playwright headless
test-e2e-ui:        # Playwright UI mode
test-watch:         # Watch mode for development
test-coverage:      # Generate coverage reports
```

## CI/CD Integration

### Local Development

```bash
make test-e2e-ui    # Interactive debugging
npx playwright test --debug
npx playwright codegen  # Record tests
```

### GitHub Actions (example)

```yaml
jobs:
  test-api:
    - Setup Go, run migrations
    - make test-unit && make test-api

  test-e2e:
    - Setup Go + Node
    - Build backend, start server
    - Run Playwright tests
    - Upload artifacts (screenshots, videos)
```

### Test Execution Order

1. Unit tests (~1-5s) - fail fast
2. API tests (~10-30s) - medium feedback
3. E2E tests (~1-3min) - comprehensive validation

## Dependencies

### New Packages

**Frontend:**
- `@playwright/test` - E2E test framework
- `@playwright/test-reporter` - HTML/JSON reports

**Backend:**
- No new dependencies (use stdlib + PocketBase)

## Success Metrics

- All critical user flows have E2E coverage
- Tests run in CI on every PR
- Test suite completes in <5 minutes
- Flaky test rate <5%
- Developers can run tests locally without setup

## Future Enhancements

- Visual regression testing with Playwright screenshots
- Load testing for ActivityPub federation
- Contract tests for API versioning
- Frontend component unit tests (Preact Testing Library)
