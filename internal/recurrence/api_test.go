package recurrence

import (
	"testing"
	"time"
)

func TestBuildVirtualEvent(t *testing.T) {
	baseID := "abc123"
	occurrenceDate := time.Date(2026, 4, 13, 18, 0, 0, 0, time.UTC)
	originalStart := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)
	originalEnd := time.Date(2026, 4, 6, 20, 0, 0, 0, time.UTC)

	ve := BuildVirtualEvent(baseID, originalStart, originalEnd, occurrenceDate)

	expectedID := "abc123__20260413"
	if ve.ID != expectedID {
		t.Errorf("Expected ID %q, got %q", expectedID, ve.ID)
	}
	if !ve.Start.Equal(occurrenceDate) {
		t.Errorf("Expected start %v, got %v", occurrenceDate, ve.Start)
	}
	expectedEnd := time.Date(2026, 4, 13, 20, 0, 0, 0, time.UTC)
	if !ve.End.Equal(expectedEnd) {
		t.Errorf("Expected end %v, got %v", expectedEnd, ve.End)
	}
	if ve.BaseEventID != baseID {
		t.Errorf("Expected base event ID %q, got %q", baseID, ve.BaseEventID)
	}
}

func TestBuildVirtualEventNoEndTime(t *testing.T) {
	baseID := "abc123"
	occurrenceDate := time.Date(2026, 4, 13, 18, 0, 0, 0, time.UTC)
	originalStart := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)

	ve := BuildVirtualEvent(baseID, originalStart, time.Time{}, occurrenceDate)

	if !ve.End.IsZero() {
		t.Errorf("Expected zero end time, got %v", ve.End)
	}
}

func TestParseVirtualID(t *testing.T) {
	baseID, dateStr, ok := ParseVirtualID("abc123__20260413")
	if !ok {
		t.Fatal("Expected ParseVirtualID to return true")
	}
	if baseID != "abc123" {
		t.Errorf("Expected base ID 'abc123', got %q", baseID)
	}
	if dateStr != "20260413" {
		t.Errorf("Expected date '20260413', got %q", dateStr)
	}
}

func TestParseVirtualIDRegular(t *testing.T) {
	_, _, ok := ParseVirtualID("abc123")
	if ok {
		t.Error("Expected ParseVirtualID to return false for regular ID")
	}
}
