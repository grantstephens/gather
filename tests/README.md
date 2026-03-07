# Testing Guide

This directory contains integration tests for the Gather community calendar application.

## Structure

```
tests/
├── e2e/              # Playwright end-to-end tests
│   ├── specs/        # Test specifications
│   ├── fixtures/     # Test data fixtures
│   └── helpers/      # Test utilities
└── README.md
```

## Running Tests

### All Tests

```bash
make test
```

### Specific Test Suites

```bash
make test-unit        # Go unit tests
make test-api         # Go API integration tests
make test-e2e         # Playwright E2E tests (headless)
make test-e2e-ui      # Playwright E2E tests (UI mode)
```

### Watch Mode

```bash
make test-watch       # Auto-rerun E2E tests on changes
```

## E2E Tests (Playwright)

### Prerequisites

Test users must exist in the database:

```bash
make seed-test-users
```

This creates:
- `test-admin@test.local` / `testpass123` (admin)
- `test-editor@test.local` / `testpass123` (editor)
- `test-user@test.local` / `testpass123` (user)

### Running Specific Tests

```bash
cd tests/e2e
npm test auth.spec.ts              # Run specific file
npm test -- --grep "should login"  # Run tests matching pattern
npm run test:debug                 # Debug mode
```

### Writing New Tests

1. Create spec file in `tests/e2e/specs/`
2. Use helpers from `tests/e2e/helpers/`
3. Use fixtures from `tests/e2e/fixtures/`
4. Always cleanup test data in `afterEach`

Example:

```typescript
import { test, expect } from '@playwright/test';
import { setupAuthenticatedPage, loginViaAPI } from '../helpers/auth';
import { TestCleanup } from '../helpers/cleanup';

test.describe('My Feature', () => {
  let cleanup: TestCleanup;

  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page, 'user');
    const token = await loginViaAPI('test-user@test.local', 'testpass123');
    cleanup = new TestCleanup(token);
  });

  test.afterEach(async () => {
    await cleanup.cleanup();
  });

  test('should do something', async ({ page }) => {
    // Your test here
  });
});
```

## API Tests (Go)

### Running Tests

```bash
go test ./internal/api/... -v
```

### Writing New Tests

1. Create test file in appropriate `internal/` package
2. Use helpers from `internal/api/testhelpers.go`
3. Always cleanup records with `defer`

Example:

```go
func TestMyFeature(t *testing.T) {
    app := NewTestApp(t)
    defer app.Cleanup()

    user := CreateTestUser(t, app, "test@example.com", "pass", "user")
    defer CleanupRecords(t, app, "users", user.ID)

    // Your test logic
}
```

## Test Data Management

### Database Strategy

Tests use the shared dev database (`pb_data/`) with cleanup after each test.

### Cleanup Pattern

**E2E Tests:**
- Track created records with `TestCleanup`
- Delete in reverse order in `afterEach`

**API Tests:**
- Use `defer CleanupRecords()` for guaranteed cleanup

### Test Isolation

Use unique identifiers to avoid conflicts:
- Email: `test-${Date.now()}@test.local`
- Titles: `[TEST] Event ${Date.now()}`
- Names: `test-tag-${Date.now()}`

## CI/CD

Tests run automatically in GitHub Actions on every push and PR.

### Local CI Simulation

```bash
CI=true make test
```

## Troubleshooting

### Tests fail with "Cannot find collection"

Run migrations first:

```bash
./gather serve
# Wait for migrations to complete, then Ctrl+C
make test
```

### E2E tests timeout

Ensure server is running:

```bash
# Terminal 1
DEV=1 ./gather serve

# Terminal 2
cd tests/e2e && npm test
```

### Test users don't exist

Seed them:

```bash
make seed-test-users
```

### Cleanup failures

Manually clean test data:

```bash
# Delete all test events
curl -X GET "http://127.0.0.1:8090/api/collections/events/records?filter=title~'[TEST]'" \
  -H "Authorization: YOUR_ADMIN_TOKEN"
# Then delete each ID
```

## Coverage

Generate coverage report:

```bash
make test-coverage
open coverage.html
```

## Best Practices

1. **DRY**: Use helpers and fixtures
2. **YAGNI**: Test what matters, not edge cases you'll never hit
3. **TDD**: Write tests before implementation when possible
4. **Fast**: Use API auth, not UI login for every test
5. **Isolated**: Clean up your data
6. **Descriptive**: Name tests clearly: `should allow admin to publish event`
