package recurrence

import (
	"testing"
	"time"
)

func TestExpandWeekly(t *testing.T) {
	start := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC) // Monday
	rangeStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	times, err := Expand("FREQ=WEEKLY", start, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(times) != 4 {
		t.Fatalf("Expected 4 occurrences, got %d: %v", len(times), times)
	}

	expected := []int{6, 13, 20, 27}
	for i, exp := range expected {
		if times[i].Day() != exp {
			t.Errorf("Occurrence %d: expected day %d, got %d", i, exp, times[i].Day())
		}
	}
}

func TestExpandFortnightly(t *testing.T) {
	start := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	times, err := Expand("FREQ=WEEKLY;INTERVAL=2", start, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(times) != 2 {
		t.Fatalf("Expected 2 occurrences, got %d: %v", len(times), times)
	}

	if times[0].Day() != 6 || times[1].Day() != 20 {
		t.Errorf("Expected days 6 and 20, got %d and %d", times[0].Day(), times[1].Day())
	}
}

func TestExpandMonthlyBySetPos(t *testing.T) {
	start := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC) // 3rd Saturday
	rangeStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	times, err := Expand("FREQ=MONTHLY;BYDAY=SA;BYSETPOS=3", start, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(times) != 3 {
		t.Fatalf("Expected 3 occurrences, got %d: %v", len(times), times)
	}

	expectedDays := []int{18, 16, 20}
	for i, exp := range expectedDays {
		if times[i].Day() != exp {
			t.Errorf("Occurrence %d: expected day %d, got %d", i, exp, times[i].Day())
		}
	}
}

func TestExpandWithCount(t *testing.T) {
	start := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	times, err := Expand("FREQ=WEEKLY;COUNT=3", start, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(times) != 3 {
		t.Fatalf("Expected 3 occurrences, got %d", len(times))
	}
}

func TestExpandWithUntil(t *testing.T) {
	start := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	times, err := Expand("FREQ=WEEKLY;UNTIL=20260427T180000Z", start, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(times) != 4 {
		t.Fatalf("Expected 4 occurrences, got %d: %v", len(times), times)
	}
}

func TestExpandRespectsMaxHorizon(t *testing.T) {
	start := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	times, err := Expand("FREQ=DAILY", start, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	maxDate := start.AddDate(1, 0, 1)
	for _, ts := range times {
		if ts.After(maxDate) {
			t.Errorf("Occurrence %v exceeds max horizon %v", ts, maxDate)
		}
	}
}

func TestExpandBeforeStartIsEmpty(t *testing.T) {
	start := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	times, err := Expand("FREQ=WEEKLY", start, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(times) != 0 {
		t.Fatalf("Expected 0 occurrences before start, got %d", len(times))
	}
}

func TestHumanLabel(t *testing.T) {
	tests := []struct {
		rule     string
		expected string
	}{
		{"FREQ=DAILY", "Repeats daily"},
		{"FREQ=WEEKLY", "Repeats weekly"},
		{"FREQ=WEEKLY;INTERVAL=2", "Repeats fortnightly"},
		{"FREQ=MONTHLY", "Repeats monthly"},
		{"FREQ=MONTHLY;BYDAY=SA;BYSETPOS=3", "Repeats monthly on the 3rd Saturday"},
		{"FREQ=YEARLY", "Repeats yearly"},
	}

	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			label := HumanLabel(tt.rule)
			if label != tt.expected {
				t.Errorf("HumanLabel(%q) = %q, want %q", tt.rule, label, tt.expected)
			}
		})
	}
}

func TestFilterExceptions_RemovesDates(t *testing.T) {
	times := []time.Time{
		time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 13, 18, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 20, 18, 0, 0, 0, time.UTC),
	}

	result := FilterExceptions(times, "2026-04-13")
	if len(result) != 2 {
		t.Fatalf("Expected 2 times after filtering, got %d", len(result))
	}
	if result[0].Day() != 6 || result[1].Day() != 20 {
		t.Errorf("Wrong dates after filtering: %v", result)
	}
}

func TestFilterExceptions_MultipleExceptions(t *testing.T) {
	times := []time.Time{
		time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 13, 18, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 20, 18, 0, 0, 0, time.UTC),
	}

	result := FilterExceptions(times, "2026-04-06, 2026-04-20")
	if len(result) != 1 {
		t.Fatalf("Expected 1 time after filtering, got %d", len(result))
	}
	if result[0].Day() != 13 {
		t.Errorf("Expected day 13, got %d", result[0].Day())
	}
}

func TestFilterExceptions_EmptyExceptions(t *testing.T) {
	times := []time.Time{
		time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 13, 18, 0, 0, 0, time.UTC),
	}

	result := FilterExceptions(times, "")
	if len(result) != 2 {
		t.Errorf("Expected 2 times with no exceptions, got %d", len(result))
	}
}
