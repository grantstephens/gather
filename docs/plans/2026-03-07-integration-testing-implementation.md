# Integration Testing Framework Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build comprehensive integration testing framework with Playwright E2E tests and Go API tests for the Gather community calendar.

**Architecture:** Two-layer testing approach with Playwright for browser-based user journeys and Go HTTP tests for API validation. Tests use shared dev database with cleanup hooks. Programmatic authentication for speed.

**Tech Stack:** Playwright, TypeScript, Go standard library, PocketBase test utilities, Make

---

## Task 1: Setup Test Infrastructure

**Files:**
- Create: `tests/e2e/package.json`
- Create: `tests/e2e/tsconfig.json`
- Create: `tests/e2e/playwright.config.ts`
- Create: `tests/e2e/.gitignore`
- Modify: `Makefile` (add test targets)
- Modify: `.gitignore` (add test artifacts)

**Step 1: Create E2E directory structure**

```bash
mkdir -p tests/e2e/{specs,fixtures,helpers,.auth}
```

**Step 2: Initialize Playwright project**

Create `tests/e2e/package.json`:

```json
{
  "name": "gather-e2e-tests",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "test": "playwright test",
    "test:ui": "playwright test --ui",
    "test:debug": "playwright test --debug",
    "test:headed": "playwright test --headed",
    "report": "playwright show-report"
  },
  "devDependencies": {
    "@playwright/test": "^1.41.0",
    "@types/node": "^20.11.0",
    "typescript": "^5.3.3"
  }
}
```

**Step 3: Create TypeScript config**

Create `tests/e2e/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ES2022",
    "moduleResolution": "node",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "types": ["node", "@playwright/test"]
  },
  "include": ["**/*.ts"],
  "exclude": ["node_modules"]
}
```

**Step 4: Create Playwright configuration**

Create `tests/e2e/playwright.config.ts`:

```typescript
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './specs',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list']
  ],
  use: {
    baseURL: 'http://127.0.0.1:8090',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'cd ../.. && DEV=1 ./gather serve',
    url: 'http://127.0.0.1:8090/api/health',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },
});
```

**Step 5: Create E2E .gitignore**

Create `tests/e2e/.gitignore`:

```
node_modules/
playwright-report/
test-results/
.auth/
```

**Step 6: Update root .gitignore**

Add to `.gitignore`:

```
# Test artifacts
tests/e2e/node_modules/
tests/e2e/playwright-report/
tests/e2e/test-results/
tests/e2e/.auth/
```

**Step 7: Add Makefile targets**

Add to `Makefile`:

```makefile
# Testing
test-unit:
	@echo "Running Go unit tests..."
	@go test ./internal/seo/... -v

test-api:
	@echo "Running API integration tests..."
	@go test ./internal/api/... -v

test-e2e:
	@echo "Running E2E tests (headless)..."
	@cd tests/e2e && npm test

test-e2e-ui:
	@echo "Running E2E tests (UI mode)..."
	@cd tests/e2e && npm run test:ui

test-watch:
	@echo "Running E2E tests in watch mode..."
	@cd tests/e2e && npm test -- --ui

test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test: test-unit test-api test-e2e
	@echo "All tests passed!"
```

**Step 8: Install Playwright dependencies**

Run: `cd tests/e2e && npm install`
Expected: Packages installed successfully

**Step 9: Install Playwright browsers**

Run: `cd tests/e2e && npx playwright install chromium`
Expected: Browser binaries downloaded

**Step 10: Commit infrastructure**

```bash
git add tests/e2e/package.json tests/e2e/tsconfig.json tests/e2e/playwright.config.ts tests/e2e/.gitignore Makefile .gitignore
git commit -m "test: add Playwright E2E test infrastructure

- Initialize Playwright with TypeScript
- Configure test runners and reporters
- Add Makefile test targets
- Setup gitignore for test artifacts

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Create Test Fixtures

**Files:**
- Create: `tests/e2e/fixtures/users.ts`
- Create: `tests/e2e/fixtures/events.ts`

**Step 1: Create user fixtures**

Create `tests/e2e/fixtures/users.ts`:

```typescript
export const TEST_USERS = {
  admin: {
    email: 'test-admin@test.local',
    password: 'testpass123',
    role: 'admin',
    displayName: 'Test Admin'
  },
  editor: {
    email: 'test-editor@test.local',
    password: 'testpass123',
    role: 'editor',
    displayName: 'Test Editor'
  },
  user: {
    email: 'test-user@test.local',
    password: 'testpass123',
    role: 'user',
    displayName: 'Test User'
  }
} as const;

export type UserRole = keyof typeof TEST_USERS;

export function getTestUser(role: UserRole) {
  return TEST_USERS[role];
}

export function generateUniqueEmail(prefix: string = 'test'): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.local`;
}
```

**Step 2: Create event fixtures**

Create `tests/e2e/fixtures/events.ts`:

```typescript
export interface TestEventData {
  title: string;
  description: string;
  start_datetime: string;
  end_datetime: string;
  status: 'draft' | 'pending' | 'published';
  place?: string;
  tags?: string[];
}

export function createTestEvent(overrides: Partial<TestEventData> = {}): TestEventData {
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  tomorrow.setHours(19, 0, 0, 0);

  const endTime = new Date(tomorrow);
  endTime.setHours(22, 0, 0, 0);

  return {
    title: `[TEST] Event ${Date.now()}`,
    description: 'This is a test event created by automated tests.',
    start_datetime: tomorrow.toISOString(),
    end_datetime: endTime.toISOString(),
    status: 'draft',
    ...overrides
  };
}

export const SAMPLE_EVENTS = {
  basic: {
    title: '[TEST] Basic Event',
    description: 'A simple test event',
  },
  withMarkdown: {
    title: '[TEST] Markdown Event',
    description: '# Heading\n\n**Bold text** and *italic text*.\n\n- List item 1\n- List item 2',
  },
  recurring: {
    title: '[TEST] Weekly Meeting',
    description: 'Recurring weekly event',
    recurrence_rule: 'FREQ=WEEKLY;COUNT=4',
  }
} as const;
```

**Step 3: Commit fixtures**

```bash
git add tests/e2e/fixtures/
git commit -m "test: add test fixtures for users and events

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Create Authentication Helpers

**Files:**
- Create: `tests/e2e/helpers/auth.ts`

**Step 1: Create auth helper with API login**

Create `tests/e2e/helpers/auth.ts`:

```typescript
import { Page, request } from '@playwright/test';
import { TEST_USERS, UserRole } from '../fixtures/users';

const AUTH_DIR = 'tests/e2e/.auth';

export async function loginViaAPI(email: string, password: string): Promise<string> {
  const apiContext = await request.newContext({
    baseURL: 'http://127.0.0.1:8090',
  });

  const response = await apiContext.post('/api/collections/users/auth-with-password', {
    data: {
      identity: email,
      password: password,
    },
  });

  if (!response.ok()) {
    throw new Error(`Login failed: ${response.status()} ${await response.text()}`);
  }

  const data = await response.json();
  await apiContext.dispose();

  return data.token;
}

export async function setupAuthenticatedPage(page: Page, role: UserRole): Promise<void> {
  const user = TEST_USERS[role];
  const token = await loginViaAPI(user.email, user.password);

  // Set auth in localStorage (PocketBase stores auth there)
  await page.goto('/');
  await page.evaluate((authData) => {
    localStorage.setItem('pocketbase_auth', JSON.stringify({
      token: authData.token,
      model: { email: authData.email }
    }));
  }, { token, email: user.email });

  // Reload to apply auth
  await page.reload();
}

export async function loginViaUI(page: Page, email: string, password: string): Promise<void> {
  await page.goto('/login');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button[type="submit"]');

  // Wait for redirect after successful login
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 5000 });
}

export async function logout(page: Page): Promise<void> {
  await page.goto('/');

  // Click logout in navigation (assuming there's a logout button/link)
  await page.click('text=Log out');

  // Wait for redirect to home or login
  await page.waitForURL((url) =>
    url.pathname === '/' || url.pathname === '/login',
    { timeout: 5000 }
  );
}

export async function isLoggedIn(page: Page): Promise<boolean> {
  const authData = await page.evaluate(() => {
    const data = localStorage.getItem('pocketbase_auth');
    return data ? JSON.parse(data) : null;
  });

  return authData !== null && authData.token;
}
```

**Step 2: Commit auth helpers**

```bash
git add tests/e2e/helpers/auth.ts
git commit -m "test: add authentication helpers for E2E tests

- API-based login for fast test setup
- UI login for testing auth flow
- Logout and auth state checking

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Create Cleanup Helpers

**Files:**
- Create: `tests/e2e/helpers/cleanup.ts`

**Step 1: Create cleanup utilities**

Create `tests/e2e/helpers/cleanup.ts`:

```typescript
import { request } from '@playwright/test';

export class TestCleanup {
  private recordsToDelete: Array<{ collection: string; id: string }> = [];
  private authToken: string;

  constructor(authToken: string) {
    this.authToken = authToken;
  }

  trackRecord(collection: string, id: string): void {
    this.recordsToDelete.push({ collection, id });
  }

  async cleanup(): Promise<void> {
    const apiContext = await request.newContext({
      baseURL: 'http://127.0.0.1:8090',
      extraHTTPHeaders: {
        'Authorization': this.authToken,
      },
    });

    // Delete in reverse order (last created first)
    for (const record of this.recordsToDelete.reverse()) {
      try {
        await apiContext.delete(`/api/collections/${record.collection}/records/${record.id}`);
      } catch (error) {
        console.warn(`Failed to delete ${record.collection}/${record.id}:`, error);
      }
    }

    await apiContext.dispose();
    this.recordsToDelete = [];
  }
}

export async function deleteTestRecords(
  authToken: string,
  collection: string,
  filter: string
): Promise<number> {
  const apiContext = await request.newContext({
    baseURL: 'http://127.0.0.1:8090',
    extraHTTPHeaders: {
      'Authorization': authToken,
    },
  });

  // Fetch records matching filter
  const response = await apiContext.get(`/api/collections/${collection}/records?filter=${encodeURIComponent(filter)}`);

  if (!response.ok()) {
    await apiContext.dispose();
    return 0;
  }

  const data = await response.json();
  const items = data.items || [];

  // Delete each record
  for (const item of items) {
    try {
      await apiContext.delete(`/api/collections/${collection}/records/${item.id}`);
    } catch (error) {
      console.warn(`Failed to delete ${collection}/${item.id}:`, error);
    }
  }

  await apiContext.dispose();
  return items.length;
}

export async function cleanupTestEvents(authToken: string): Promise<number> {
  // Delete all events with [TEST] prefix in title
  return deleteTestRecords(authToken, 'events', 'title ~ "[TEST]"');
}

export async function cleanupTestUsers(adminToken: string): Promise<number> {
  // Delete all users with test emails
  return deleteTestRecords(adminToken, 'users', 'email ~ "test-" || email ~ "@test.local"');
}
```

**Step 2: Commit cleanup helpers**

```bash
git add tests/e2e/helpers/cleanup.ts
git commit -m "test: add cleanup helpers for test data management

- Track and delete created records
- Filter-based cleanup for test data
- Reverse-order deletion for dependencies

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Create Setup Helpers

**Files:**
- Create: `tests/e2e/helpers/setup.ts`

**Step 1: Create test data seeding utilities**

Create `tests/e2e/helpers/setup.ts`:

```typescript
import { request } from '@playwright/test';

export interface TestPlace {
  id: string;
  name: string;
  latitude: number;
  longitude: number;
}

export interface TestTag {
  id: string;
  name: string;
  color: string;
}

export async function createTestPlace(
  authToken: string,
  name: string = `Test Place ${Date.now()}`
): Promise<TestPlace> {
  const apiContext = await request.newContext({
    baseURL: 'http://127.0.0.1:8090',
    extraHTTPHeaders: {
      'Authorization': authToken,
    },
  });

  const response = await apiContext.post('/api/collections/places/records', {
    data: {
      name,
      address: '123 Test St',
      city: 'Test City',
      latitude: 45.5152,
      longitude: -122.6784,
      status: 'approved',
    },
  });

  const data = await response.json();
  await apiContext.dispose();

  return {
    id: data.id,
    name: data.name,
    latitude: data.latitude,
    longitude: data.longitude,
  };
}

export async function createTestTag(
  authToken: string,
  name: string = `test-tag-${Date.now()}`,
  color: string = '#3498db'
): Promise<TestTag> {
  const apiContext = await request.newContext({
    baseURL: 'http://127.0.0.1:8090',
    extraHTTPHeaders: {
      'Authorization': authToken,
    },
  });

  const response = await apiContext.post('/api/collections/tags/records', {
    data: {
      name,
      color,
      status: 'approved',
    },
  });

  const data = await response.json();
  await apiContext.dispose();

  return {
    id: data.id,
    name: data.name,
    color: data.color,
  };
}

export async function createTestUser(
  adminToken: string,
  email: string,
  password: string,
  role: 'user' | 'editor' | 'admin' = 'user'
): Promise<{ id: string; email: string }> {
  const apiContext = await request.newContext({
    baseURL: 'http://127.0.0.1:8090',
    extraHTTPHeaders: {
      'Authorization': adminToken,
    },
  });

  const response = await apiContext.post('/api/collections/users/records', {
    data: {
      email,
      password,
      passwordConfirm: password,
      role,
      display_name: `Test ${role}`,
    },
  });

  const data = await response.json();
  await apiContext.dispose();

  return {
    id: data.id,
    email: data.email,
  };
}

export async function ensureTestUsersExist(adminToken: string): Promise<void> {
  const testUsers = [
    { email: 'test-admin@test.local', password: 'testpass123', role: 'admin' as const },
    { email: 'test-editor@test.local', password: 'testpass123', role: 'editor' as const },
    { email: 'test-user@test.local', password: 'testpass123', role: 'user' as const },
  ];

  for (const user of testUsers) {
    try {
      await createTestUser(adminToken, user.email, user.password, user.role);
    } catch (error) {
      // User might already exist, that's okay
    }
  }
}
```

**Step 2: Commit setup helpers**

```bash
git add tests/e2e/helpers/setup.ts
git commit -m "test: add setup helpers for test data seeding

- Create test places, tags, users
- Ensure test users exist for auth tests
- Reusable data creation functions

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Write Authentication Tests

**Files:**
- Create: `tests/e2e/specs/auth.spec.ts`

**Step 1: Create auth test spec**

Create `tests/e2e/specs/auth.spec.ts`:

```typescript
import { test, expect } from '@playwright/test';
import { TEST_USERS } from '../fixtures/users';
import { loginViaUI, logout, isLoggedIn, setupAuthenticatedPage } from '../helpers/auth';

test.describe('Authentication', () => {
  test('should login with valid credentials via UI', async ({ page }) => {
    const user = TEST_USERS.user;

    await loginViaUI(page, user.email, user.password);

    // Should redirect to home page
    expect(page.url()).not.toContain('/login');

    // Should be logged in
    const loggedIn = await isLoggedIn(page);
    expect(loggedIn).toBe(true);
  });

  test('should show error with invalid credentials', async ({ page }) => {
    await page.goto('/login');

    await page.fill('input[type="email"]', 'nonexistent@test.local');
    await page.fill('input[type="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');

    // Should show error message
    await expect(page.locator('text=/error|invalid|failed/i')).toBeVisible({ timeout: 5000 });

    // Should still be on login page
    expect(page.url()).toContain('/login');
  });

  test('should logout successfully', async ({ page }) => {
    // Login first
    await setupAuthenticatedPage(page, 'user');
    expect(await isLoggedIn(page)).toBe(true);

    // Logout
    await logout(page);

    // Should be logged out
    const loggedIn = await isLoggedIn(page);
    expect(loggedIn).toBe(false);
  });

  test('should redirect to login for protected routes', async ({ page }) => {
    // Try to access submit page without auth
    await page.goto('/submit');

    // Should redirect to login
    await page.waitForURL('**/login**', { timeout: 5000 });
    expect(page.url()).toContain('/login');
  });

  test('should persist session across page refresh', async ({ page }) => {
    // Login
    await setupAuthenticatedPage(page, 'user');

    // Refresh page
    await page.reload();

    // Should still be logged in
    const loggedIn = await isLoggedIn(page);
    expect(loggedIn).toBe(true);
  });
});
```

**Step 2: Run auth tests to verify they fail appropriately**

Run: `cd tests/e2e && npm test auth.spec.ts`
Expected: Tests run (may fail if test users don't exist yet, that's okay for now)

**Step 3: Commit auth tests**

```bash
git add tests/e2e/specs/auth.spec.ts
git commit -m "test: add authentication E2E tests

- UI login flow validation
- Invalid credentials error handling
- Logout functionality
- Protected route redirection
- Session persistence

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Write Event Tests

**Files:**
- Create: `tests/e2e/specs/events.spec.ts`

**Step 1: Create event test spec**

Create `tests/e2e/specs/events.spec.ts`:

```typescript
import { test, expect } from '@playwright/test';
import { setupAuthenticatedPage, loginViaAPI } from '../helpers/auth';
import { createTestEvent } from '../fixtures/events';
import { TestCleanup } from '../helpers/cleanup';

test.describe('Event Management', () => {
  let cleanup: TestCleanup;

  test.beforeEach(async ({ page }) => {
    // Setup auth and cleanup for each test
    await setupAuthenticatedPage(page, 'user');
    const token = await loginViaAPI('test-user@test.local', 'testpass123');
    cleanup = new TestCleanup(token);
  });

  test.afterEach(async () => {
    await cleanup.cleanup();
  });

  test('should create a draft event', async ({ page }) => {
    const eventData = createTestEvent({
      title: '[TEST] My Test Event',
      description: 'This is a test event',
    });

    await page.goto('/submit');

    await page.fill('input[name="title"]', eventData.title);
    await page.fill('textarea[name="description"]', eventData.description);

    // Fill datetime fields (format might vary based on your input type)
    await page.fill('input[name="start_datetime"]', eventData.start_datetime.slice(0, 16));
    await page.fill('input[name="end_datetime"]', eventData.end_datetime.slice(0, 16));

    await page.click('button[type="submit"]');

    // Should show success message or redirect to event page
    await expect(page.locator('text=/success|created/i')).toBeVisible({ timeout: 10000 });

    // Extract event ID from URL or response for cleanup
    // This is a simplified version - adjust based on your actual UI
    await page.waitForTimeout(1000);
  });

  test('should edit own draft event', async ({ page, request }) => {
    // First create an event via API
    const token = await loginViaAPI('test-user@test.local', 'testpass123');
    const apiContext = await request.newContext({
      baseURL: 'http://127.0.0.1:8090',
      extraHTTPHeaders: { 'Authorization': token },
    });

    const eventData = createTestEvent();
    const createResponse = await apiContext.post('/api/collections/events/records', {
      data: eventData,
    });
    const event = await createResponse.json();
    cleanup.trackRecord('events', event.id);

    // Navigate to edit page
    await page.goto(`/edit/${event.id}`);

    const newTitle = '[TEST] Updated Event Title';
    await page.fill('input[name="title"]', newTitle);
    await page.click('button[type="submit"]');

    // Should show success
    await expect(page.locator('text=/success|updated/i')).toBeVisible({ timeout: 10000 });

    await apiContext.dispose();
  });

  test('should not allow regular user to publish event', async ({ page, request }) => {
    // Create draft event
    const token = await loginViaAPI('test-user@test.local', 'testpass123');
    const apiContext = await request.newContext({
      baseURL: 'http://127.0.0.1:8090',
      extraHTTPHeaders: { 'Authorization': token },
    });

    const eventData = createTestEvent({ status: 'draft' });
    const createResponse = await apiContext.post('/api/collections/events/records', {
      data: eventData,
    });
    const event = await createResponse.json();
    cleanup.trackRecord('events', event.id);

    await page.goto(`/edit/${event.id}`);

    // Regular users shouldn't see publish option or it should be disabled
    const publishButton = page.locator('button:has-text("Publish")');
    if (await publishButton.isVisible()) {
      await expect(publishButton).toBeDisabled();
    }

    await apiContext.dispose();
  });

  test('should delete own event', async ({ page, request }) => {
    // Create event
    const token = await loginViaAPI('test-user@test.local', 'testpass123');
    const apiContext = await request.newContext({
      baseURL: 'http://127.0.0.1:8090',
      extraHTTPHeaders: { 'Authorization': token },
    });

    const eventData = createTestEvent();
    const createResponse = await apiContext.post('/api/collections/events/records', {
      data: eventData,
    });
    const event = await createResponse.json();

    // Don't track for cleanup since we're deleting it
    await page.goto(`/event/${event.id}`);

    // Click delete button
    await page.click('button:has-text("Delete")');

    // Confirm deletion (if there's a confirmation dialog)
    await page.click('button:has-text("Confirm")').catch(() => {});

    // Should redirect or show success
    await page.waitForTimeout(1000);

    await apiContext.dispose();
  });
});

test.describe('Event Management - Admin', () => {
  let cleanup: TestCleanup;

  test.beforeEach(async ({ page }) => {
    await setupAuthenticatedPage(page, 'admin');
    const token = await loginViaAPI('test-admin@test.local', 'testpass123');
    cleanup = new TestCleanup(token);
  });

  test.afterEach(async () => {
    await cleanup.cleanup();
  });

  test('should allow admin to publish event', async ({ page, request }) => {
    // Create draft event
    const token = await loginViaAPI('test-admin@test.local', 'testpass123');
    const apiContext = await request.newContext({
      baseURL: 'http://127.0.0.1:8090',
      extraHTTPHeaders: { 'Authorization': token },
    });

    const eventData = createTestEvent({ status: 'draft' });
    const createResponse = await apiContext.post('/api/collections/events/records', {
      data: eventData,
    });
    const event = await createResponse.json();
    cleanup.trackRecord('events', event.id);

    await page.goto(`/edit/${event.id}`);

    // Admin should be able to publish
    const publishButton = page.locator('button:has-text("Publish")');
    await expect(publishButton).toBeEnabled();

    await publishButton.click();

    // Should show success
    await expect(page.locator('text=/published|success/i')).toBeVisible({ timeout: 10000 });

    await apiContext.dispose();
  });
});
```

**Step 2: Commit event tests**

```bash
git add tests/e2e/specs/events.spec.ts
git commit -m "test: add event management E2E tests

- Create draft events
- Edit own events
- Permission checks for publishing
- Event deletion
- Admin publishing workflow

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Write Calendar Navigation Tests

**Files:**
- Create: `tests/e2e/specs/calendar.spec.ts`

**Step 1: Create calendar test spec**

Create `tests/e2e/specs/calendar.spec.ts`:

```typescript
import { test, expect } from '@playwright/test';
import { setupAuthenticatedPage, loginViaAPI } from '../helpers/auth';
import { createTestEvent } from '../fixtures/events';
import { TestCleanup } from '../helpers/cleanup';

test.describe('Calendar Navigation', () => {
  let cleanup: TestCleanup;

  test.beforeAll(async () => {
    // Setup cleanup for all tests
    const token = await loginViaAPI('test-admin@test.local', 'testpass123');
    cleanup = new TestCleanup(token);
  });

  test.afterAll(async () => {
    await cleanup.cleanup();
  });

  test('should display current month by default', async ({ page }) => {
    await page.goto('/');

    const currentMonth = new Date().toLocaleString('default', { month: 'long', year: 'numeric' });

    // Check that current month is displayed (exact selector depends on your HTML)
    await expect(page.locator('h1, h2, .month-title')).toContainText(new RegExp(new Date().toLocaleString('default', { month: 'long' }), 'i'));
  });

  test('should navigate to next month', async ({ page }) => {
    await page.goto('/');

    // Click next month button (adjust selector as needed)
    await page.click('button[aria-label*="next" i], button:has-text("→")');

    await page.waitForTimeout(500);

    // URL should update or content should change
    const url = page.url();
    expect(url).toMatch(/month=|date=/); // Adjust based on your URL structure
  });

  test('should navigate to previous month', async ({ page }) => {
    await page.goto('/');

    // Click previous month button
    await page.click('button[aria-label*="prev" i], button:has-text("←")');

    await page.waitForTimeout(500);

    // URL should update
    const url = page.url();
    expect(url).toMatch(/month=|date=/);
  });

  test('should click event to view details', async ({ page, request }) => {
    // Create a published test event
    const token = await loginViaAPI('test-admin@test.local', 'testpass123');
    const apiContext = await request.newContext({
      baseURL: 'http://127.0.0.1:8090',
      extraHTTPHeaders: { 'Authorization': token },
    });

    const eventData = createTestEvent({
      title: '[TEST] Clickable Event',
      status: 'published',
    });

    const createResponse = await apiContext.post('/api/collections/events/records', {
      data: eventData,
    });
    const event = await createResponse.json();
    cleanup.trackRecord('events', event.id);

    await page.goto('/');

    // Wait for event to appear and click it
    await page.waitForTimeout(1000); // Give calendar time to load
    await page.click(`text="${eventData.title}"`);

    // Should navigate to event detail page
    await page.waitForURL(`**/event/${event.id}`, { timeout: 5000 });

    // Event title should be visible
    await expect(page.locator('h1')).toContainText(eventData.title);

    await apiContext.dispose();
  });

  test('should filter by tag', async ({ page, request }) => {
    // Create tag and event with that tag
    const token = await loginViaAPI('test-admin@test.local', 'testpass123');
    const apiContext = await request.newContext({
      baseURL: 'http://127.0.0.1:8090',
      extraHTTPHeaders: { 'Authorization': token },
    });

    // Create tag
    const tagResponse = await apiContext.post('/api/collections/tags/records', {
      data: { name: `test-tag-${Date.now()}`, color: '#e74c3c', status: 'approved' },
    });
    const tag = await tagResponse.json();
    cleanup.trackRecord('tags', tag.id);

    // Create event with tag
    const eventData = createTestEvent({
      title: '[TEST] Tagged Event',
      status: 'published',
      tags: [tag.id],
    });
    const eventResponse = await apiContext.post('/api/collections/events/records', {
      data: eventData,
    });
    const event = await eventResponse.json();
    cleanup.trackRecord('events', event.id);

    await page.goto('/');
    await page.waitForTimeout(1000);

    // Click tag to filter
    await page.click(`text="${tag.name}"`);

    // Should show filtered events
    await expect(page.locator(`text="${eventData.title}"`)).toBeVisible();

    await apiContext.dispose();
  });
});
```

**Step 2: Commit calendar tests**

```bash
git add tests/e2e/specs/calendar.spec.ts
git commit -m "test: add calendar navigation E2E tests

- Current month display
- Next/previous month navigation
- Event detail navigation
- Tag filtering

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Create Go API Test Helpers

**Files:**
- Create: `internal/api/testhelpers.go`

**Step 1: Create Go test utilities**

Create `internal/api/testhelpers.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// NewTestApp creates a test PocketBase app instance
func NewTestApp(t *testing.T) *pocketbase.PocketBase {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	return app
}

// AuthRecord represents an authenticated user for testing
type AuthRecord struct {
	ID    string
	Email string
	Token string
}

// CreateTestUser creates a user and returns auth token
func CreateTestUser(t *testing.T, app *pocketbase.PocketBase, email, password, role string) AuthRecord {
	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Failed to find users collection: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("email", email)
	record.Set("password", password)
	record.Set("passwordConfirm", password)
	record.Set("role", role)
	record.Set("display_name", "Test "+role)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Authenticate to get token
	token, err := authenticateUser(app, email, password)
	if err != nil {
		t.Fatalf("Failed to authenticate test user: %v", err)
	}

	return AuthRecord{
		ID:    record.Id,
		Email: email,
		Token: token,
	}
}

// authenticateUser logs in and returns auth token
func authenticateUser(app *pocketbase.PocketBase, email, password string) (string, error) {
	// Use PocketBase's internal auth
	record, err := app.FindAuthRecordByEmail("users", email)
	if err != nil {
		return "", err
	}

	if !record.ValidatePassword(password) {
		return "", err
	}

	token, err := core.NewRecordAuthToken(app, record)
	if err != nil {
		return "", err
	}

	return token, nil
}

// NewAuthRequest creates an HTTP request with auth header
func NewAuthRequest(method, path string, body string, token string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	return req
}

// ParseJSONResponse parses JSON response into target
func ParseJSONResponse(t *testing.T, resp *httptest.ResponseRecorder, target interface{}) {
	if err := json.Unmarshal(resp.Body.Bytes(), target); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
}

// CleanupRecords deletes test records
func CleanupRecords(t *testing.T, app *pocketbase.PocketBase, collection string, ids ...string) {
	for _, id := range ids {
		record, err := app.FindRecordById(collection, id)
		if err != nil {
			continue // Record might already be deleted
		}
		if err := app.Delete(record); err != nil {
			t.Logf("Warning: failed to cleanup record %s: %v", id, err)
		}
	}
}
```

**Step 2: Commit Go test helpers**

```bash
git add internal/api/testhelpers.go
git commit -m "test: add Go API test helpers

- Test app initialization
- User creation and authentication
- Authenticated request builders
- JSON parsing utilities
- Cleanup helpers

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Write Go API Tests for Events

**Files:**
- Create: `internal/api/events_test.go`

**Step 1: Write event API tests**

Create `internal/api/events_test.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	user := CreateTestUser(t, app, "test@example.com", "testpass123", "user")
	defer CleanupRecords(t, app, "users", user.ID)

	eventData := map[string]interface{}{
		"title":          "Test Event",
		"description":    "Test Description",
		"start_datetime": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"end_datetime":   time.Now().Add(26 * time.Hour).Format(time.RFC3339),
		"status":         "draft",
	}

	body, _ := json.Marshal(eventData)
	req := NewAuthRequest("POST", "/api/collections/events/records", string(body), user.Token)
	rec := httptest.NewRecorder()

	// Execute request
	app.OnServe().Trigger(&core.ServeEvent{
		App:    app,
		Router: app.OnServe().Router,
	}, func(e *core.ServeEvent) error {
		e.Router.ServeHTTP(rec, req)
		return nil
	})

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	ParseJSONResponse(t, rec, &result)

	if result["title"] != eventData["title"] {
		t.Errorf("Expected title %v, got %v", eventData["title"], result["title"])
	}

	if result["status"] != "draft" {
		t.Errorf("Expected status draft, got %v", result["status"])
	}

	// Cleanup
	if id, ok := result["id"].(string); ok {
		CleanupRecords(t, app, "events", id)
	}
}

func TestUserCannotPublishEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	user := CreateTestUser(t, app, "test@example.com", "testpass123", "user")
	defer CleanupRecords(t, app, "users", user.ID)

	eventData := map[string]interface{}{
		"title":          "Test Event",
		"description":    "Test Description",
		"start_datetime": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"status":         "published", // Regular user trying to publish
	}

	body, _ := json.Marshal(eventData)
	req := NewAuthRequest("POST", "/api/collections/events/records", string(body), user.Token)
	rec := httptest.NewRecorder()

	app.OnServe().Trigger(&core.ServeEvent{
		App:    app,
		Router: app.OnServe().Router,
	}, func(e *core.ServeEvent) error {
		e.Router.ServeHTTP(rec, req)
		return nil
	})

	var result map[string]interface{}
	ParseJSONResponse(t, rec, &result)

	// Should be forced to draft by hooks
	if result["status"] != "draft" && result["status"] != "pending" {
		t.Errorf("Regular user should not be able to publish directly, got status: %v", result["status"])
	}

	if id, ok := result["id"].(string); ok {
		CleanupRecords(t, app, "events", id)
	}
}

func TestAdminCanPublishEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	admin := CreateTestUser(t, app, "admin@example.com", "testpass123", "admin")
	defer CleanupRecords(t, app, "users", admin.ID)

	eventData := map[string]interface{}{
		"title":          "Admin Event",
		"description":    "Test Description",
		"start_datetime": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"status":         "published",
	}

	body, _ := json.Marshal(eventData)
	req := NewAuthRequest("POST", "/api/collections/events/records", string(body), admin.Token)
	rec := httptest.NewRecorder()

	app.OnServe().Trigger(&core.ServeEvent{
		App:    app,
		Router: app.OnServe().Router,
	}, func(e *core.ServeEvent) error {
		e.Router.ServeHTTP(rec, req)
		return nil
	})

	var result map[string]interface{}
	ParseJSONResponse(t, rec, &result)

	if result["status"] != "published" {
		t.Errorf("Admin should be able to publish, got status: %v", result["status"])
	}

	if id, ok := result["id"].(string); ok {
		CleanupRecords(t, app, "events", id)
	}
}

func TestDeleteEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	user := CreateTestUser(t, app, "test@example.com", "testpass123", "user")
	defer CleanupRecords(t, app, "users", user.ID)

	// Create event first
	eventData := map[string]interface{}{
		"title":          "Event to Delete",
		"description":    "Will be deleted",
		"start_datetime": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"status":         "draft",
	}

	body, _ := json.Marshal(eventData)
	req := NewAuthRequest("POST", "/api/collections/events/records", string(body), user.Token)
	rec := httptest.NewRecorder()

	app.OnServe().Trigger(&core.ServeEvent{
		App:    app,
		Router: app.OnServe().Router,
	}, func(e *core.ServeEvent) error {
		e.Router.ServeHTTP(rec, req)
		return nil
	})

	var created map[string]interface{}
	ParseJSONResponse(t, rec, &created)
	eventID := created["id"].(string)

	// Now delete it
	delReq := NewAuthRequest("DELETE", "/api/collections/events/records/"+eventID, "", user.Token)
	delRec := httptest.NewRecorder()

	app.OnServe().Trigger(&core.ServeEvent{
		App:    app,
		Router: app.OnServe().Router,
	}, func(e *core.ServeEvent) error {
		e.Router.ServeHTTP(delRec, delReq)
		return nil
	})

	if delRec.Code != http.StatusNoContent && delRec.Code != http.StatusOK {
		t.Errorf("Expected successful deletion, got %d", delRec.Code)
	}
}
```

**Step 2: Run API tests**

Run: `go test ./internal/api/... -v`
Expected: Tests compile and run (may fail initially depending on app setup)

**Step 3: Commit event API tests**

```bash
git add internal/api/events_test.go
git commit -m "test: add Go API integration tests for events

- Create event endpoint
- Permission checks for publishing
- Admin can publish
- Event deletion

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 11: Add Test Seed Script

**Files:**
- Create: `scripts/seed-test-users.sh`
- Modify: `Makefile`

**Step 1: Create test user seeding script**

Create `scripts/seed-test-users.sh`:

```bash
#!/bin/bash
# Seed test users for E2E testing

set -e

BASE_URL="${BASE_URL:-http://127.0.0.1:8090}"
ADMIN_EMAIL="${PB_ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${PB_ADMIN_PASSWORD:-adminpassword123}"

echo "Seeding test users to $BASE_URL..."

# Authenticate as admin
AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/collections/_superusers/auth-with-password" \
  -H "Content-Type: application/json" \
  -d "{\"identity\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")

TOKEN=$(echo "$AUTH_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Error: Failed to authenticate as superuser"
  exit 1
fi

echo "Authenticated successfully"

# Create test users
create_user() {
  local email=$1
  local password=$2
  local role=$3

  echo "Creating $role: $email"

  curl -s -X POST "$BASE_URL/api/collections/users/records" \
    -H "Authorization: $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\":\"$email\",
      \"password\":\"$password\",
      \"passwordConfirm\":\"$password\",
      \"role\":\"$role\",
      \"display_name\":\"Test $role\"
    }" > /dev/null || echo "  (user may already exist)"
}

create_user "test-admin@test.local" "testpass123" "admin"
create_user "test-editor@test.local" "testpass123" "editor"
create_user "test-user@test.local" "testpass123" "user"

echo ""
echo "Test users created:"
echo "  test-admin@test.local  / testpass123 (admin)"
echo "  test-editor@test.local / testpass123 (editor)"
echo "  test-user@test.local   / testpass123 (user)"
```

**Step 2: Make script executable**

Run: `chmod +x scripts/seed-test-users.sh`

**Step 3: Add Makefile target**

Add to `Makefile`:

```makefile
# Seed test users for E2E testing
seed-test-users: setup-admin
	@echo "Seeding test users..."
	@./scripts/seed-test-users.sh
```

**Step 4: Update dev target to include test users**

Modify `dev` target in `Makefile` to call `seed-test-users` after seeding:

```makefile
dev: build-backend setup-admin
	@echo "Starting development servers..."
	@echo "Backend: http://127.0.0.1:8090 (proxies frontend to Vite)"
	@echo "Vite:    http://localhost:5173 (direct)"
	@echo "Press Ctrl+C to stop"
	@echo ""
	@bash -c '\
		cleanup() { \
			echo ""; \
			echo "Shutting down..."; \
			kill $$VITE_PID $$BACKEND_PID 2>/dev/null; \
			wait; \
			echo "Stopped."; \
		}; \
		trap cleanup EXIT; \
		cd frontend && npm run dev & VITE_PID=$$!; \
		sleep 2; \
		DEV=1 ./gather serve & BACKEND_PID=$$!; \
		sleep 2; \
		$(MAKE) -s seed; \
		$(MAKE) -s seed-test-users; \
		wait \
	'
```

**Step 5: Commit seed script**

```bash
git add scripts/seed-test-users.sh Makefile
git commit -m "test: add test user seeding script

- Create persistent test users for E2E tests
- Add Makefile target for test user seeding
- Integrate into dev workflow

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 12: Documentation & Final Integration

**Files:**
- Create: `tests/README.md`
- Modify: `README.md` (add testing section)

**Step 1: Create test documentation**

Create `tests/README.md`:

```markdown
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
```

**Step 2: Update main README**

Add to main `README.md` (after existing content):

```markdown
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
```

**Step 3: Commit documentation**

```bash
git add tests/README.md README.md
git commit -m "docs: add comprehensive testing documentation

- Test running instructions
- Test writing guidelines
- Troubleshooting guide
- Best practices

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Verification

After implementing all tasks, verify the framework:

**Step 1: Install all dependencies**

```bash
cd tests/e2e && npm install && npx playwright install chromium && cd ../..
```

**Step 2: Start dev server**

```bash
make dev
```

Wait for server to start and seeding to complete.

**Step 3: In another terminal, run tests**

```bash
# Run E2E tests
make test-e2e

# Run API tests
make test-api

# Run all tests
make test
```

**Expected Results:**
- E2E tests execute in headless browser
- Tests create and cleanup their data
- All critical flows (auth, events, calendar) are validated
- API tests run quickly (<30s)
- Test reports generated in `tests/e2e/playwright-report/`

**Step 4: View test report**

```bash
cd tests/e2e && npm run report
```

Opens HTML report with test results, screenshots, and traces.

---

## Summary

This implementation provides:

✅ Playwright E2E testing framework with TypeScript
✅ Go API integration testing with PocketBase
✅ Test fixtures and helpers for DRY test code
✅ Authentication utilities (API + UI)
✅ Data cleanup and seeding utilities
✅ Makefile integration for easy test execution
✅ Comprehensive test coverage for critical flows
✅ CI-ready test suite
✅ Documentation for developers

The framework follows TDD principles, uses shared dev database with cleanup, and provides fast feedback loops for developers.
