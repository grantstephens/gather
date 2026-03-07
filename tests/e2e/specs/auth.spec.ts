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
