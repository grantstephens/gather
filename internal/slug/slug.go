package slug

import (
	"regexp"
	"strings"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// Generate creates a URL-friendly slug from a title and record ID.
// Format: {slugified-title}-{first8charsOfID}
// The ID suffix guarantees uniqueness.
func Generate(title, id string) string {
	s := strings.ToLower(title)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "event"
	}
	suffix := id
	if len(id) > 8 {
		suffix = id[:8]
	}
	return s + "-" + suffix
}
