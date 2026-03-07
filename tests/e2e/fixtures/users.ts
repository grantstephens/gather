export const TEST_USERS = {
  admin: {
    email: 'test-admin@test.local',
    password: 'testpass123',
    role: 'admin',
    display_name: 'Test Admin'
  },
  editor: {
    email: 'test-editor@test.local',
    password: 'testpass123',
    role: 'editor',
    display_name: 'Test Editor'
  },
  user: {
    email: 'test-user@test.local',
    password: 'testpass123',
    role: 'user',
    display_name: 'Test User'
  }
} as const;

export type UserRole = keyof typeof TEST_USERS;

export function getTestUser(role: UserRole) {
  return TEST_USERS[role];
}

export function generateUniqueEmail(prefix: string = 'test'): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.local`;
}
