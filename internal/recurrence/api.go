package recurrence

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// VirtualEvent holds the shifted dates for a recurring event occurrence.
type VirtualEvent struct {
	ID          string
	BaseEventID string
	Start       time.Time
	End         time.Time
}

// BuildVirtualEvent creates a VirtualEvent by shifting dates to an occurrence date.
func BuildVirtualEvent(baseID string, originalStart, originalEnd time.Time, occurrenceStart time.Time) VirtualEvent {
	ve := VirtualEvent{
		ID:          fmt.Sprintf("%s__%s", baseID, occurrenceStart.Format("20060102")),
		BaseEventID: baseID,
		Start:       occurrenceStart,
	}
	if !originalEnd.IsZero() {
		ve.End = occurrenceStart.Add(originalEnd.Sub(originalStart))
	}
	return ve
}

// ParseVirtualID splits a synthetic ID into base event ID and date string.
func ParseVirtualID(id string) (baseID string, dateStr string, ok bool) {
	parts := strings.SplitN(id, "__", 2)
	if len(parts) != 2 || len(parts[1]) != 8 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ExpandedEvent is the JSON shape returned by the events listing API.
type ExpandedEvent struct {
	ID                   string `json:"id"`
	BaseEventID          string `json:"base_event_id,omitempty"`
	Slug                 string `json:"slug,omitempty"`
	Title                string `json:"title"`
	Description          string `json:"description"`
	StartDatetime        string `json:"start_datetime"`
	EndDatetime          string `json:"end_datetime,omitempty"`
	Place                string `json:"place,omitempty"`
	Tags                 any    `json:"tags,omitempty"`
	Image                string `json:"image,omitempty"`
	Author               string `json:"author,omitempty"`
	Status               string `json:"status"`
	RecurrenceRule       string `json:"recurrence_rule,omitempty"`
	RecurrenceExceptions string `json:"recurrence_exceptions,omitempty"`
	RecurrenceLabel      string `json:"recurrence_label,omitempty"`
	Expand               any    `json:"expand,omitempty"`
}

// ListExpandedEvents fetches published events in a date range, expanding recurring events.
func ListExpandedEvents(app core.App, rangeStart, rangeEnd time.Time, town string, tagIDs []string, page, pageSize int) ([]ExpandedEvent, int, error) {
	// Build safe extra filters from discrete params
	var extraFilters []string
	if town != "" {
		// Sanitize: only allow alphanumeric, spaces, hyphens, apostrophes
		safe := true
		for _, r := range town {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '\'' || r == '.') {
				safe = false
				break
			}
		}
		if safe && town != "" {
			extraFilters = append(extraFilters, fmt.Sprintf("place.city = '%s'", town))
		}
	}
	for _, tagID := range tagIDs {
		// Tag IDs are PocketBase record IDs - only allow alphanumeric
		safe := true
		for _, r := range tagID {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				safe = false
				break
			}
		}
		if safe {
			extraFilters = append(extraFilters, fmt.Sprintf("tags.id ?= '%s'", tagID))
		}
	}
	extraFilter := strings.Join(extraFilters, " && ")

	filters := []string{
		"status = 'published'",
		"recurrence_rule = ''",
		fmt.Sprintf("start_datetime >= '%s'", rangeStart.UTC().Format("2006-01-02 15:04:05")),
		fmt.Sprintf("start_datetime <= '%s'", rangeEnd.UTC().Format("2006-01-02 15:04:05")),
	}
	if extraFilter != "" {
		filters = append(filters, extraFilter)
	}

	regularEvents, err := app.FindRecordsByFilter("events", strings.Join(filters, " && "), "start_datetime", 0, 0)
	if err != nil {
		regularEvents = []*core.Record{}
	}

	recurFilters := []string{
		"status = 'published'",
		"recurrence_rule != ''",
		fmt.Sprintf("start_datetime <= '%s'", rangeEnd.UTC().Format("2006-01-02 15:04:05")),
	}
	if extraFilter != "" {
		recurFilters = append(recurFilters, extraFilter)
	}

	recurringEvents, err := app.FindRecordsByFilter("events", strings.Join(recurFilters, " && "), "start_datetime", 0, 0)
	if err != nil {
		recurringEvents = []*core.Record{}
	}

	var allEvents []ExpandedEvent

	for _, rec := range regularEvents {
		allEvents = append(allEvents, RecordToExpandedPublic(app, rec, "", nil))
	}

	for _, rec := range recurringEvents {
		rule := rec.GetString("recurrence_rule")
		exceptions := rec.GetString("recurrence_exceptions")
		start := rec.GetDateTime("start_datetime").Time()

		occurrences, err := Expand(rule, start, rangeStart, rangeEnd)
		if err != nil {
			continue
		}
		occurrences = FilterExceptions(occurrences, exceptions)

		end := rec.GetDateTime("end_datetime").Time()
		for _, occ := range occurrences {
			ve := BuildVirtualEvent(rec.Id, start, end, occ)
			allEvents = append(allEvents, RecordToExpandedPublic(app, rec, ve.ID, &ve))
		}
	}

	sort.Slice(allEvents, func(i, j int) bool {
		ti, _ := time.Parse("2006-01-02 15:04:05.000Z", allEvents[i].StartDatetime)
		tj, _ := time.Parse("2006-01-02 15:04:05.000Z", allEvents[j].StartDatetime)
		return ti.Before(tj)
	})

	total := len(allEvents)
	offset := (page - 1) * pageSize
	if offset >= total {
		return []ExpandedEvent{}, total, nil
	}
	end := offset + pageSize
	if end > total {
		end = total
	}

	return allEvents[offset:end], total, nil
}

// RecordToExpandedPublic converts a PocketBase record to ExpandedEvent JSON shape,
// optionally overriding dates from a VirtualEvent. Expands place and tag relations.
func RecordToExpandedPublic(app core.App, rec *core.Record, virtualID string, ve *VirtualEvent) ExpandedEvent {
	e := ExpandedEvent{
		ID:                   rec.Id,
		Slug:                 rec.GetString("slug"),
		Title:                rec.GetString("title"),
		Description:          rec.GetString("description"),
		StartDatetime:        rec.GetDateTime("start_datetime").Time().UTC().Format("2006-01-02 15:04:05.000Z"),
		Place:                rec.GetString("place"),
		Tags:                 rec.Get("tags"),
		Image:                rec.GetString("image"),
		Author:               rec.GetString("author"),
		Status:               rec.GetString("status"),
		RecurrenceRule:       rec.GetString("recurrence_rule"),
		RecurrenceExceptions: rec.GetString("recurrence_exceptions"),
	}

	if endTime := rec.GetDateTime("end_datetime").Time(); !endTime.IsZero() {
		e.EndDatetime = endTime.UTC().Format("2006-01-02 15:04:05.000Z")
	}

	if e.RecurrenceRule != "" {
		e.RecurrenceLabel = HumanLabel(e.RecurrenceRule)
	}

	if ve != nil {
		e.ID = virtualID
		e.BaseEventID = ve.BaseEventID
		e.StartDatetime = ve.Start.UTC().Format("2006-01-02 15:04:05.000Z")
		if !ve.End.IsZero() {
			e.EndDatetime = ve.End.UTC().Format("2006-01-02 15:04:05.000Z")
		}
	}

	// Expand relations
	expand := make(map[string]any)
	if placeID := rec.GetString("place"); placeID != "" {
		if place, err := app.FindRecordById("places", placeID); err == nil {
			expand["place"] = map[string]any{
				"id":        place.Id,
				"name":      place.GetString("name"),
				"address":   place.GetString("address"),
				"city":      place.GetString("city"),
				"latitude":  place.GetFloat("latitude"),
				"longitude": place.GetFloat("longitude"),
				"status":    place.GetString("status"),
			}
		}
	}
	if tagIDs := rec.GetStringSlice("tags"); len(tagIDs) > 0 {
		var tagList []map[string]any
		for _, tagID := range tagIDs {
			if tag, err := app.FindRecordById("tags", tagID); err == nil {
				tagList = append(tagList, map[string]any{
					"id":     tag.Id,
					"name":   tag.GetString("name"),
					"color":  tag.GetString("color"),
					"status": tag.GetString("status"),
				})
			}
		}
		if len(tagList) > 0 {
			expand["tags"] = tagList
		}
	}
	if len(expand) > 0 {
		e.Expand = expand
	}

	return e
}
