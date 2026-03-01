package ical

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func formatICSTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func GenerateFeed(app core.App, baseURL string, filter string) ([]byte, error) {
	settings, _ := app.FindFirstRecordByFilter("settings", "id != ''")

	instanceName := "Gather"
	if settings != nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	fullFilter := "status = 'published'"
	if filter != "" {
		fullFilter += " && " + filter
	}

	events, err := app.FindRecordsByFilter(
		"events",
		fullFilter,
		"-start_datetime",
		100,
		0,
	)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString(fmt.Sprintf("PRODID:-//%s//Gather//EN\r\n", instanceName))
	sb.WriteString(fmt.Sprintf("X-WR-CALNAME:%s\r\n", instanceName))

	for _, event := range events {
		startTime := event.GetDateTime("start_datetime").Time()
		endTime := event.GetDateTime("end_datetime").Time()
		if endTime.IsZero() {
			endTime = startTime.Add(time.Hour)
		}

		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s@%s\r\n", event.Id, baseURL))
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICSTime(event.GetDateTime("created").Time())))
		sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICSTime(startTime)))
		sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICSTime(endTime)))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(event.GetString("title"))))

		if desc := event.GetString("description"); desc != "" {
			desc = stripHTML(desc)
			sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(desc)))
		}

		sb.WriteString(fmt.Sprintf("URL:%s/event/%s\r\n", baseURL, event.Id))
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")
	return []byte(sb.String()), nil
}

func GenerateSingleEvent(app core.App, baseURL string, eventID string) ([]byte, error) {
	event, err := app.FindRecordById("events", eventID)
	if err != nil {
		return nil, err
	}

	settings, _ := app.FindFirstRecordByFilter("settings", "id != ''")
	instanceName := "Gather"
	if settings != nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}

	startTime := event.GetDateTime("start_datetime").Time()
	endTime := event.GetDateTime("end_datetime").Time()
	if endTime.IsZero() {
		endTime = startTime.Add(time.Hour)
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString(fmt.Sprintf("PRODID:-//%s//Gather//EN\r\n", instanceName))
	sb.WriteString("BEGIN:VEVENT\r\n")
	sb.WriteString(fmt.Sprintf("UID:%s@%s\r\n", event.Id, baseURL))
	sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICSTime(event.GetDateTime("created").Time())))
	sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICSTime(startTime)))
	sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICSTime(endTime)))
	sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(event.GetString("title"))))

	if desc := event.GetString("description"); desc != "" {
		desc = stripHTML(desc)
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(desc)))
	}

	sb.WriteString(fmt.Sprintf("URL:%s/event/%s\r\n", baseURL, event.Id))
	sb.WriteString("END:VEVENT\r\n")
	sb.WriteString("END:VCALENDAR\r\n")

	return []byte(sb.String()), nil
}

func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
