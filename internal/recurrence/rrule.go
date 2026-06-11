package recurrence

import (
	"fmt"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// MaxHorizon is the furthest we'll expand a never-ending rule
const MaxHorizon = 365 * 24 * time.Hour

// Expand returns all occurrence start times for the given RRULE within [rangeStart, rangeEnd).
// For rules without UNTIL or COUNT, expansion is capped at 1 year from dtstart.
func Expand(rule string, dtstart time.Time, rangeStart time.Time, rangeEnd time.Time) ([]time.Time, error) {
	ruleStr := fmt.Sprintf("DTSTART:%s\nRRULE:%s", dtstart.UTC().Format("20060102T150405Z"), rule)

	r, err := rrule.StrToRRule(ruleStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RRULE %q: %w", rule, err)
	}

	// Cap range for open-ended rules
	hasEnd := strings.Contains(strings.ToUpper(rule), "UNTIL") || strings.Contains(strings.ToUpper(rule), "COUNT")
	effectiveEnd := rangeEnd
	if !hasEnd {
		maxEnd := dtstart.Add(MaxHorizon)
		if effectiveEnd.After(maxEnd) {
			effectiveEnd = maxEnd
		}
	}

	return r.Between(rangeStart, effectiveEnd, true), nil
}

// FilterExceptions removes dates listed in the comma-separated exceptions string.
// Exceptions are ISO date strings (YYYY-MM-DD) compared against occurrence dates.
func FilterExceptions(times []time.Time, exceptions string) []time.Time {
	if exceptions == "" {
		return times
	}

	exSet := make(map[string]bool)
	for _, ex := range strings.Split(exceptions, ",") {
		exSet[strings.TrimSpace(ex)] = true
	}

	filtered := make([]time.Time, 0, len(times))
	for _, t := range times {
		dateStr := t.Format("2006-01-02")
		if !exSet[dateStr] {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// HumanLabel returns a human-readable description of an RRULE string.
func HumanLabel(rule string) string {
	upper := strings.ToUpper(rule)
	parts := make(map[string]string)
	for _, part := range strings.Split(upper, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			parts[kv[0]] = kv[1]
		}
	}

	freq := parts["FREQ"]
	interval := parts["INTERVAL"]
	byDay := parts["BYDAY"]
	bySetPos := parts["BYSETPOS"]

	switch freq {
	case "DAILY":
		return "Repeats daily"
	case "WEEKLY":
		if interval == "2" {
			return "Repeats fortnightly"
		}
		return "Repeats weekly"
	case "MONTHLY":
		if byDay != "" && bySetPos != "" {
			ordinal := ordinalString(bySetPos)
			dayName := dayAbbrevToFull(byDay)
			return fmt.Sprintf("Repeats monthly on the %s %s", ordinal, dayName)
		}
		return "Repeats monthly"
	case "YEARLY":
		return "Repeats yearly"
	}

	return "Repeats"
}

// BuildRule constructs an RRULE string from structured UI parameters.
func BuildRule(freq string, interval int, byDay string, bySetPos int, until *time.Time, count int) string {
	parts := []string{"FREQ=" + strings.ToUpper(freq)}

	if interval > 1 {
		parts = append(parts, fmt.Sprintf("INTERVAL=%d", interval))
	}
	if byDay != "" {
		parts = append(parts, "BYDAY="+strings.ToUpper(byDay))
	}
	if bySetPos != 0 {
		parts = append(parts, fmt.Sprintf("BYSETPOS=%d", bySetPos))
	}
	if until != nil {
		parts = append(parts, "UNTIL="+until.UTC().Format("20060102T150405Z"))
	}
	if count > 0 {
		parts = append(parts, fmt.Sprintf("COUNT=%d", count))
	}

	return strings.Join(parts, ";")
}

func ordinalString(s string) string {
	switch s {
	case "1":
		return "1st"
	case "2":
		return "2nd"
	case "3":
		return "3rd"
	case "4":
		return "4th"
	case "5":
		return "5th"
	case "-1":
		return "last"
	default:
		return s + "th"
	}
}

func dayAbbrevToFull(abbrev string) string {
	switch abbrev {
	case "MO":
		return "Monday"
	case "TU":
		return "Tuesday"
	case "WE":
		return "Wednesday"
	case "TH":
		return "Thursday"
	case "FR":
		return "Friday"
	case "SA":
		return "Saturday"
	case "SU":
		return "Sunday"
	default:
		return abbrev
	}
}
