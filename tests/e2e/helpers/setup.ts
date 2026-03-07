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

  if (!response.ok()) {
    const text = await response.text();
    throw new Error(`Failed to create test place: ${response.status()} ${text}`);
  }

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

  if (!response.ok()) {
    const text = await response.text();
    throw new Error(`Failed to create test tag: ${response.status()} ${text}`);
  }

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

  if (!response.ok()) {
    const text = await response.text();
    throw new Error(`Failed to create test user: ${response.status()} ${text}`);
  }

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
