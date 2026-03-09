import { request } from '@playwright/test';
import { TEST_USERS } from './fixtures/users';

async function globalSetup() {
  const baseURL = 'http://127.0.0.1:8090';
  const context = await request.newContext({ baseURL });

  // Create test users if they don't exist
  for (const [role, userData] of Object.entries(TEST_USERS)) {
    try {
      // Try to authenticate - if it succeeds, user exists
      const authResponse = await context.post('/api/collections/users/auth-with-password', {
        data: {
          identity: userData.email,
          password: userData.password,
        },
      });

      if (authResponse.ok()) {
        console.log(`✓ Test ${role} user already exists: ${userData.email}`);
        continue;
      }
    } catch (e) {
      // User doesn't exist, will create below
    }

    // Create the user
    try {
      const createResponse = await context.post('/api/collections/users/records', {
        data: {
          email: userData.email,
          password: userData.password,
          passwordConfirm: userData.password,
          role: userData.role,
          display_name: userData.display_name,
        },
      });

      if (createResponse.ok()) {
        console.log(`✓ Created test ${role} user: ${userData.email}`);
      } else {
        const error = await createResponse.text();
        console.log(`✗ Failed to create ${role} user: ${error}`);
      }
    } catch (e) {
      console.log(`✗ Error creating ${role} user:`, e);
    }
  }

  await context.dispose();
}

export default globalSetup;
