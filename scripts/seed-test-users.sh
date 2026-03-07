#!/bin/bash
# Seed test users for E2E testing

set -e

BASE_URL="${BASE_URL:-http://127.0.0.1:8090}"
ADMIN_EMAIL="${PB_ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${PB_ADMIN_PASSWORD:-adminpassword123}"

echo "Seeding test users to $BASE_URL..."

# Authenticate as admin
AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/collections/_superusers/auth-with-password" \
  -H "Content-Type: application/json" \
  -d "{\"identity\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")

TOKEN=$(echo "$AUTH_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Error: Failed to authenticate as superuser"
  exit 1
fi

echo "Authenticated successfully"

# Create test users
create_user() {
  local email=$1
  local password=$2
  local role=$3

  echo "Creating $role: $email"

  curl -s -X POST "$BASE_URL/api/collections/users/records" \
    -H "Authorization: $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\":\"$email\",
      \"password\":\"$password\",
      \"passwordConfirm\":\"$password\",
      \"role\":\"$role\",
      \"display_name\":\"Test $role\"
    }" > /dev/null || echo "  (user may already exist)"
}

create_user "test-admin@test.local" "testpass123" "admin"
create_user "test-editor@test.local" "testpass123" "editor"
create_user "test-user@test.local" "testpass123" "user"

echo ""
echo "Test users created:"
echo "  test-admin@test.local  / testpass123 (admin)"
echo "  test-editor@test.local / testpass123 (editor)"
echo "  test-user@test.local   / testpass123 (user)"
