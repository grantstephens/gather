package seo

import (
	"encoding/json"
	"testing"

	"github.com/pocketbase/pocketbase/tools/types"
)

func TestFormatDateTime(t *testing.T) {
	tests := []struct {
		name     string
		datetime types.DateTime
		want     string
	}{
		{
			name:     "Valid time",
			datetime: mustParseDateTime("2026-03-15T14:30:00Z"),
			want:     "2026-03-15T14:30:00Z",
		},
		{
			name:     "Zero time",
			datetime: types.DateTime{},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDateTime(tt.datetime); got != tt.want {
				t.Errorf("formatDateTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper to parse datetime in tests
func mustParseDateTime(value string) types.DateTime {
	dt, err := types.ParseDateTime(value)
	if err != nil {
		panic(err)
	}
	return dt
}

func TestMapEventStatus(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"published", "https://schema.org/EventScheduled"},
		{"cancelled", "https://schema.org/EventCancelled"},
		{"draft", "https://schema.org/EventScheduled"},
		{"pending", "https://schema.org/EventScheduled"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := mapEventStatus(tt.status); got != tt.want {
				t.Errorf("mapEventStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Bold text",
			input: "This is **bold** text",
			want:  "This is bold text",
		},
		{
			name:  "Italic text",
			input: "This is *italic* text",
			want:  "This is italic text",
		},
		{
			name:  "Links",
			input: "Check out [this link](https://example.com)",
			want:  "Check out this link",
		},
		{
			name:  "Headings",
			input: "## Heading\nContent here",
			want:  "Heading\nContent here",
		},
		{
			name:  "Mixed formatting",
			input: "**Bold** and *italic* with [link](url)",
			want:  "Bold and italic with link",
		},
		{
			name:  "Plain text",
			input: "Just plain text",
			want:  "Just plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripMarkdown(tt.input); got != tt.want {
				t.Errorf("stripMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventJSONLDStructure(t *testing.T) {
	// Test that EventJSONLD marshals correctly
	jsonLD := &EventJSONLD{
		Context:             "https://schema.org",
		Type:                "Event",
		Name:                "Test Event",
		Description:         "A test event",
		StartDate:           "2026-03-15T14:30:00Z",
		EventStatus:         "https://schema.org/EventScheduled",
		EventAttendanceMode: "https://schema.org/OfflineEventAttendanceMode",
		URL:                 "https://example.com/event/123",
	}

	data, err := json.Marshal(jsonLD)
	if err != nil {
		t.Fatalf("Failed to marshal JSON-LD: %v", err)
	}

	// Unmarshal to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON-LD: %v", err)
	}

	// Verify required fields
	if result["@context"] != "https://schema.org" {
		t.Errorf("@context = %v, want https://schema.org", result["@context"])
	}
	if result["@type"] != "Event" {
		t.Errorf("@type = %v, want Event", result["@type"])
	}
	if result["name"] != "Test Event" {
		t.Errorf("name = %v, want Test Event", result["name"])
	}
}
