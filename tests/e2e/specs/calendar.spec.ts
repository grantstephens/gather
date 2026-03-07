import { test, expect, request as playwrightRequest } from '@playwright/test';
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

  test('should click event to view details', async ({ page }) => {
    // Create a published test event
    const token = await loginViaAPI('test-admin@test.local', 'testpass123');
    const apiContext = await playwrightRequest.newContext({
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

  test('should filter by tag', async ({ page }) => {
    // Create tag and event with that tag
    const token = await loginViaAPI('test-admin@test.local', 'testpass123');
    const apiContext = await playwrightRequest.newContext({
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
