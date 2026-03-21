import { test, expect, request } from '@playwright/test';
import { setupAuthenticatedPage } from '../helpers/auth';
import { createTestPage, deleteTestPage, getAdminToken } from '../helpers/pages';

test.describe('Custom Pages', () => {
  let adminToken: string;
  let createdPageIds: string[] = [];

  test.beforeAll(async () => {
    adminToken = await getAdminToken();
  });

  test.afterAll(async () => {
    for (const id of createdPageIds) {
      await deleteTestPage(adminToken, id).catch(() => {});
    }
  });

  test('admin can create a page and it appears in nav and footer', async ({ page }) => {
    const slug = `about-${Date.now()}`;
    const title = 'About Us';

    // Create page via API
    const created = await createTestPage(adminToken, slug, title, {
      show_in_nav: true,
      show_in_footer: true,
      content: '## About\n\nWelcome to our community.',
    });
    createdPageIds.push(created.id);

    // Visit home page and check nav link
    await page.goto('/');
    await expect(page.locator(`nav a[href="/${slug}"]`)).toBeVisible();

    // Check footer link
    await expect(page.locator(`footer a[href="/${slug}"]`)).toBeVisible();
  });

  test('public can view a custom page', async ({ page }) => {
    const slug = `faq-${Date.now()}`;
    const created = await createTestPage(adminToken, slug, 'FAQ', {
      content: '## Frequently Asked Questions\n\nSome answers here.',
    });
    createdPageIds.push(created.id);

    await page.goto(`/${slug}`);
    await expect(page.locator('h1')).toContainText('FAQ');
    await expect(page.locator('.page-content')).toBeVisible();
  });

  test('visiting unknown slug shows page not found', async ({ page }) => {
    await page.goto('/this-slug-does-not-exist-xyz');
    await expect(page.locator('h1')).toContainText('Page not found');
  });

  test('page with show_in_nav=false does not appear in nav', async ({ page }) => {
    const slug = `hidden-nav-${Date.now()}`;
    const created = await createTestPage(adminToken, slug, 'Hidden Nav', {
      show_in_nav: false,
      show_in_footer: true,
    });
    createdPageIds.push(created.id);

    await page.goto('/');
    await expect(page.locator(`nav a[href="/${slug}"]`)).not.toBeVisible();
    await expect(page.locator(`footer a[href="/${slug}"]`)).toBeVisible();
  });

  test('admin sees Pages tab in admin panel', async ({ page }) => {
    await setupAuthenticatedPage(page, 'admin');
    await page.goto('/admin');
    await expect(page.locator('button.tab', { hasText: 'Pages' })).toBeVisible();
  });

  test('editor does not see Pages tab in admin panel', async ({ page }) => {
    await setupAuthenticatedPage(page, 'editor');
    await page.goto('/admin');
    await expect(page.locator('button.tab', { hasText: 'Pages' })).not.toBeVisible();
  });

  test('admin can create a page via the admin UI', async ({ page }) => {
    const slug = `ui-created-${Date.now()}`;
    await setupAuthenticatedPage(page, 'admin');
    await page.goto('/admin');

    // Click Pages tab
    await page.click('button.tab:has-text("Pages")');

    // Click New Page
    await page.click('button:has-text("New Page")');

    // Fill in form
    await page.fill('#page-title', 'UI Created Page');
    // Wait for slug auto-fill
    await expect(page.locator('#page-slug')).toHaveValue('ui-created-page');
    // Override slug with our unique one
    await page.fill('#page-slug', slug);

    // Save
    await page.click('button:has-text("Save Page")');

    // Should return to list and show the page
    await expect(page.locator(`.item-info h3:has-text("UI Created Page")`).first()).toBeVisible();

    // Clean up: find the created page id via API
    const apiContext = await request.newContext({
      baseURL: 'http://127.0.0.1:8090',
      extraHTTPHeaders: { Authorization: adminToken },
    });
    const res = await apiContext.get(`/api/collections/pages/records?filter=slug="${slug}"`);
    const data = await res.json();
    if (data.items?.[0]) createdPageIds.push(data.items[0].id);
    await apiContext.dispose();
  });
});
