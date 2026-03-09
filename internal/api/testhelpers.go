package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// NewTestApp creates a test PocketBase app instance with migrations
func NewTestApp(t *testing.T) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	// Run migrations to set up collections
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return app
}

// AuthRecord represents an authenticated user for testing
type AuthRecord struct {
	ID    string
	Email string
	Token string
}

// CreateTestUser creates a user and returns auth token
func CreateTestUser(t *testing.T, app *tests.TestApp, email, password, role string) AuthRecord {
	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("Failed to find users collection: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("email", email)
	record.Set("password", password)
	record.Set("passwordConfirm", password)
	record.Set("role", role)
	record.Set("display_name", "Test "+role)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Authenticate to get token
	token, err := authenticateUser(app, email, password)
	if err != nil {
		t.Fatalf("Failed to authenticate test user: %v", err)
	}

	return AuthRecord{
		ID:    record.Id,
		Email: email,
		Token: token,
	}
}

// authenticateUser logs in and returns auth token
func authenticateUser(app *tests.TestApp, email, password string) (string, error) {
	// Use PocketBase's internal auth
	record, err := app.FindAuthRecordByEmail("users", email)
	if err != nil {
		return "", err
	}

	if !record.ValidatePassword(password) {
		return "", err
	}

	// Generate static auth token
	token, err := record.NewStaticAuthToken(0)
	if err != nil {
		return "", err
	}

	return token, nil
}

// NewAuthRequest creates an HTTP request with auth header
func NewAuthRequest(method, path string, body string, token string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	return req
}

// ParseJSONResponse parses JSON response into target
func ParseJSONResponse(t *testing.T, resp *httptest.ResponseRecorder, target interface{}) {
	if err := json.Unmarshal(resp.Body.Bytes(), target); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
}

// CleanupRecords deletes test records
func CleanupRecords(t *testing.T, app *tests.TestApp, collection string, ids ...string) {
	for _, id := range ids {
		record, err := app.FindRecordById(collection, id)
		if err != nil {
			continue // Record might already be deleted
		}
		if err := app.Delete(record); err != nil {
			t.Logf("Warning: failed to cleanup record %s: %v", id, err)
		}
	}
}
