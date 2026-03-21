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

export async function loginViaAPIFull(email: string, password: string): Promise<{ token: string; record: Record<string, unknown> }> {
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

  return { token: data.token, record: data.record };
}

export async function setupAuthenticatedPage(page: Page, role: UserRole): Promise<void> {
  const user = TEST_USERS[role];
  const { token, record } = await loginViaAPIFull(user.email, user.password);

  // Set auth in localStorage (PocketBase stores auth there)
  await page.goto('/');
  await page.evaluate((authData) => {
    localStorage.setItem('pocketbase_auth', JSON.stringify({
      token: authData.token,
      model: authData.record,
    }));
  }, { token, record });

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

  return authData !== null && !!authData.token;
}
