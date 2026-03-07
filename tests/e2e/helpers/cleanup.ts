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
