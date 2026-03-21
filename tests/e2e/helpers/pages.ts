import { request } from '@playwright/test';
import { loginViaAPI } from './auth';
import { TEST_USERS } from '../fixtures/users';

export interface TestPage {
  id: string;
  title: string;
  slug: string;
}

export async function createTestPage(
  authToken: string,
  slug: string = `test-page-${Date.now()}`,
  title: string = 'Test Page',
  options: { show_in_nav?: boolean; show_in_footer?: boolean; content?: string } = {}
): Promise<TestPage> {
  const apiContext = await request.newContext({
    baseURL: 'http://127.0.0.1:8090',
    extraHTTPHeaders: { Authorization: authToken },
  });

  const response = await apiContext.post('/api/collections/pages/records', {
    data: {
      title,
      slug,
      content: options.content ?? '## Hello\n\nThis is a test page.',
      show_in_nav: options.show_in_nav ?? true,
      show_in_footer: options.show_in_footer ?? true,
    },
  });

  if (!response.ok()) {
    throw new Error(`Failed to create test page: ${response.status()} ${await response.text()}`);
  }

  const data = await response.json();
  await apiContext.dispose();
  return { id: data.id, title: data.title, slug: data.slug };
}

export async function deleteTestPage(authToken: string, pageId: string): Promise<void> {
  const apiContext = await request.newContext({
    baseURL: 'http://127.0.0.1:8090',
    extraHTTPHeaders: { Authorization: authToken },
  });
  await apiContext.delete(`/api/collections/pages/records/${pageId}`);
  await apiContext.dispose();
}

export async function getAdminToken(): Promise<string> {
  return loginViaAPI(TEST_USERS.admin.email, TEST_USERS.admin.password);
}
