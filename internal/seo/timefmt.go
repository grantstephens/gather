package seo

import (
	"time"
	_ "time/tzdata" // the container image may lack zoneinfo
)

var london = func() *time.Location {
	if l, err := time.LoadLocation("Europe/London"); err == nil {
		return l
	}
	return time.UTC
}()

func londonTime(t time.Time) time.Time { return t.In(london) }

// shortDate renders "Sat 10 Oct".
func shortDate(t time.Time) string { return londonTime(t).Format("Mon 2 Jan") }

// longDateTime renders "Saturday 10 October 2026, 7:30 PM".
func longDateTime(t time.Time) string { return londonTime(t).Format("Monday 2 January 2006, 3:04 PM") }
