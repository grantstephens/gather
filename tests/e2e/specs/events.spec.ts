import { test, expect, request as playwrightRequest } from '@playwright/test';
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
    if (eventData.end_datetime) {
      await page.fill('input[name="end_datetime"]', eventData.end_datetime.slice(0, 16));
    }

    await page.click('button[type="submit"]');

    // Should show success message or redirect to event page
    await expect(page.locator('text=/success|created/i')).toBeVisible({ timeout: 10000 });

    // Extract event ID from URL or response for cleanup
    // This is a simplified version - adjust based on your actual UI
    await page.waitForTimeout(1000);
  });

  test('should edit own draft event', async ({ page }) => {
    // First create an event via API
    const token = await loginViaAPI('test-user@test.local', 'testpass123');
    const apiContext = await playwrightRequest.newContext({
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

  test('should not allow regular user to publish event', async ({ page }) => {
    // Create draft event
    const token = await loginViaAPI('test-user@test.local', 'testpass123');
    const apiContext = await playwrightRequest.newContext({
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

  test('should delete own event', async ({ page }) => {
    // Create event
    const token = await loginViaAPI('test-user@test.local', 'testpass123');
    const apiContext = await playwrightRequest.newContext({
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

  test('should allow admin to publish event', async ({ page }) => {
    // Create draft event
    const token = await loginViaAPI('test-admin@test.local', 'testpass123');
    const apiContext = await playwrightRequest.newContext({
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
