package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

// generateUniqueEmail creates a unique email address using timestamp
func generateUniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UnixNano())
}

func TestCreateEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	email := generateUniqueEmail("test")
	user := CreateTestUser(t, app, email, "testpass123", "user")
	defer CleanupRecords(t, app, "users", user.ID)

	// Get events collection
	collection, err := app.FindCollectionByNameOrId("events")
	if err != nil {
		t.Fatalf("Failed to find events collection: %v", err)
	}

	// Create event record
	record := core.NewRecord(collection)
	record.Set("title", "Test Event")
	record.Set("description", "Test Description")
	record.Set("start_datetime", time.Now().Add(24*time.Hour))
	record.Set("end_datetime", time.Now().Add(26*time.Hour))
	record.Set("status", "draft")
	record.Set("author", user.ID)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}
	defer CleanupRecords(t, app, "events", record.Id)

	// Verify event was created with correct data
	if record.GetString("title") != "Test Event" {
		t.Errorf("Expected title 'Test Event', got %s", record.GetString("title"))
	}

	if record.GetString("status") != "draft" {
		t.Errorf("Expected status 'draft', got %s", record.GetString("status"))
	}
}

func TestEventStatusValidation(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	email := generateUniqueEmail("test")
	user := CreateTestUser(t, app, email, "testpass123", "user")
	defer CleanupRecords(t, app, "users", user.ID)

	// Get events collection
	collection, err := app.FindCollectionByNameOrId("events")
	if err != nil {
		t.Fatalf("Failed to find events collection: %v", err)
	}

	// Create event with pending status
	record := core.NewRecord(collection)
	record.Set("title", "Test Event")
	record.Set("description", "Test Description")
	record.Set("start_datetime", time.Now().Add(24*time.Hour))
	record.Set("status", "pending")
	record.Set("author", user.ID)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}
	defer CleanupRecords(t, app, "events", record.Id)

	// Verify status was saved correctly
	if record.GetString("status") != "pending" {
		t.Errorf("Expected status 'pending', got %s", record.GetString("status"))
	}
}

func TestAdminCanPublishEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	email := generateUniqueEmail("admin")
	admin := CreateTestUser(t, app, email, "testpass123", "admin")
	defer CleanupRecords(t, app, "users", admin.ID)

	// Get events collection
	collection, err := app.FindCollectionByNameOrId("events")
	if err != nil {
		t.Fatalf("Failed to find events collection: %v", err)
	}

	// Create published event as admin
	record := core.NewRecord(collection)
	record.Set("title", "Admin Event")
	record.Set("description", "Test Description")
	record.Set("start_datetime", time.Now().Add(24*time.Hour))
	record.Set("status", "published")
	record.Set("author", admin.ID)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}
	defer CleanupRecords(t, app, "events", record.Id)

	// Admin should be able to publish
	if record.GetString("status") != "published" {
		t.Errorf("Admin should be able to publish, got status: %s", record.GetString("status"))
	}
}

func TestDeleteEvent(t *testing.T) {
	app := NewTestApp(t)
	defer app.Cleanup()

	email := generateUniqueEmail("test")
	user := CreateTestUser(t, app, email, "testpass123", "user")
	defer CleanupRecords(t, app, "users", user.ID)

	// Get events collection
	collection, err := app.FindCollectionByNameOrId("events")
	if err != nil {
		t.Fatalf("Failed to find events collection: %v", err)
	}

	// Create event
	record := core.NewRecord(collection)
	record.Set("title", "Event to Delete")
	record.Set("description", "Will be deleted")
	record.Set("start_datetime", time.Now().Add(24*time.Hour))
	record.Set("status", "draft")
	record.Set("author", user.ID)

	if err := app.Save(record); err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	eventID := record.Id

	// Delete it
	if err := app.Delete(record); err != nil {
		t.Fatalf("Failed to delete event: %v", err)
	}

	// Verify it's deleted
	_, err = app.FindRecordById("events", eventID)
	if err == nil {
		t.Error("Event should be deleted")
	}
}
