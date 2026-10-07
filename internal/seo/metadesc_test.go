package seo

import (
	"strings"
	"testing"
	"time"
)

func TestMetaDescription(t *testing.T) {
	start := time.Date(2026, 10, 10, 19, 30, 0, 0, time.UTC)
	long := strings.Repeat("A long and useful description sentence. ", 6)
	if got := metaDescription(long, start, "Pitlochry Festival Theatre", "Pitlochry"); strings.Contains(got, "Pitlochry") || len(got) > 163 {
		t.Errorf("long description must pass through truncated, got %q", got)
	}
	got := metaDescription("Join Alan.", start, "Perth Concert Hall", "Perth")
	if !strings.Contains(got, "Join Alan.") || !strings.Contains(got, "Sat 10 Oct") || !strings.Contains(got, "Perth Concert Hall") {
		t.Errorf("short description needs date+venue suffix, got %q", got)
	}
	if got := metaDescription("", start, "", ""); !strings.Contains(got, "Sat 10 Oct") {
		t.Errorf("empty description with no venue still gets the date, got %q", got)
	}
	if got := metaDescription("", time.Time{}, "", ""); got != "" {
		t.Errorf("nothing known => empty, got %q", got)
	}
}

func TestShortAndLongDateTimeUseLondonTime(t *testing.T) {
	// 23:30 UTC on 10 Oct is 00:30 BST on 11 Oct
	ts := time.Date(2026, 10, 10, 23, 30, 0, 0, time.UTC)
	if got := shortDate(ts); got != "Sun 11 Oct" {
		t.Errorf("shortDate = %q", got)
	}
	if got := longDateTime(ts); got != "Sunday 11 October 2026, 12:30 AM" {
		t.Errorf("longDateTime = %q", got)
	}
}
