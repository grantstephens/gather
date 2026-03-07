package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"
)

func TestCreateEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	user := CreateTestUser(t, app, "test@example.com", "testpass123", "user")
	defer CleanupRecords(t, app, "users", user.ID)

	eventData := map[string]interface{}{
		"title":          "Test Event",
		"description":    "Test Description",
		"start_datetime": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"end_datetime":   time.Now().Add(26 * time.Hour).Format(time.RFC3339),
		"status":         "draft",
	}

	body, _ := json.Marshal(eventData)

	scenario := tests.ApiScenario{
		Name:   "create event as user",
		Method: http.MethodPost,
		URL:    "/api/collections/events/records",
		Body:   strings.NewReader(string(body)),
		Headers: map[string]string{
			"Authorization": user.Token,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        200,
		ExpectedContent: []string{
			`"title":"Test Event"`,
			`"status":"draft"`,
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			// Cleanup created event
			var result map[string]interface{}
			json.NewDecoder(res.Body).Decode(&result)
			if id, ok := result["id"].(string); ok {
				CleanupRecords(t.(*testing.T), app, "events", id)
			}
		},
	}

	scenario.Test(t)
}

func TestUserCannotPublishEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	user := CreateTestUser(t, app, "test@example.com", "testpass123", "user")
	defer CleanupRecords(t, app, "users", user.ID)

	eventData := map[string]interface{}{
		"title":          "Test Event",
		"description":    "Test Description",
		"start_datetime": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"status":         "published", // Regular user trying to publish
	}

	body, _ := json.Marshal(eventData)

	scenario := tests.ApiScenario{
		Name:   "user cannot publish event directly",
		Method: http.MethodPost,
		URL:    "/api/collections/events/records",
		Body:   strings.NewReader(string(body)),
		Headers: map[string]string{
			"Authorization": user.Token,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        200,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			var result map[string]interface{}
			json.NewDecoder(res.Body).Decode(&result)

			// Should be forced to draft or pending by hooks
			if result["status"] != "draft" && result["status"] != "pending" {
				t.Errorf("Regular user should not be able to publish directly, got status: %v", result["status"])
			}

			if id, ok := result["id"].(string); ok {
				CleanupRecords(t.(*testing.T), app, "events", id)
			}
		},
	}

	scenario.Test(t)
}

func TestAdminCanPublishEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	admin := CreateTestUser(t, app, "admin@example.com", "testpass123", "admin")
	defer CleanupRecords(t, app, "users", admin.ID)

	eventData := map[string]interface{}{
		"title":          "Admin Event",
		"description":    "Test Description",
		"start_datetime": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"status":         "published",
	}

	body, _ := json.Marshal(eventData)

	scenario := tests.ApiScenario{
		Name:   "admin can publish event",
		Method: http.MethodPost,
		URL:    "/api/collections/events/records",
		Body:   strings.NewReader(string(body)),
		Headers: map[string]string{
			"Authorization": admin.Token,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        200,
		ExpectedContent: []string{
			`"status":"published"`,
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			var result map[string]interface{}
			json.NewDecoder(res.Body).Decode(&result)
			if id, ok := result["id"].(string); ok {
				CleanupRecords(t.(*testing.T), app, "events", id)
			}
		},
	}

	scenario.Test(t)
}

func TestDeleteEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	user := CreateTestUser(t, app, "test@example.com", "testpass123", "user")
	defer CleanupRecords(t, app, "users", user.ID)

	// Create event first
	eventData := map[string]interface{}{
		"title":          "Event to Delete",
		"description":    "Will be deleted",
		"start_datetime": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"status":         "draft",
	}

	body, _ := json.Marshal(eventData)

	createScenario := tests.ApiScenario{
		Name:   "create event for deletion test",
		Method: http.MethodPost,
		URL:    "/api/collections/events/records",
		Body:   strings.NewReader(string(body)),
		Headers: map[string]string{
			"Authorization": user.Token,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			return app
		},
		DisableTestAppCleanup: true,
		ExpectedStatus:        200,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			var created map[string]interface{}
			json.NewDecoder(res.Body).Decode(&created)
			eventID := created["id"].(string)

			// Now delete it
			deleteScenario := tests.ApiScenario{
				Name:   "delete event",
				Method: http.MethodDelete,
				URL:    "/api/collections/events/records/" + eventID,
				Headers: map[string]string{
					"Authorization": user.Token,
				},
				TestAppFactory: func(t testing.TB) *tests.TestApp {
					return app
				},
				DisableTestAppCleanup: true,
				ExpectedStatus:        204,
			}
			deleteScenario.Test(t.(*testing.T))
		},
	}

	createScenario.Test(t)
}
