package slug

import (
	"fmt"
	"regexp"
	"strings"

	"gather/internal/textutil"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify lowercases s, decodes stray HTML entities, and joins words with hyphens.
// It returns "" if nothing usable remains.
func Slugify(s string) string {
	s = strings.ToLower(textutil.DecodeEntities(s))
	return strings.Trim(nonAlphanumeric.ReplaceAllString(s, "-"), "-")
}

// Generate creates a URL-friendly slug from a title and record ID.
// Format: {slugified-title}-{first8charsOfID}; the ID suffix guarantees uniqueness.
func Generate(title, id string) string {
	s := Slugify(title)
	if s == "" {
		s = "event"
	}
	suffix := id
	if len(id) > 8 {
		suffix = id[:8]
	}
	return s + "-" + suffix
}

// Unique returns base, or base-2, base-3, … — the first value taken() reports free.
func Unique(base string, taken func(string) bool) string {
	if base == "" {
		base = "place"
	}
	if !taken(base) {
		return base
	}
	for i := 2; ; i++ {
		if c := fmt.Sprintf("%s-%d", base, i); !taken(c) {
			return c
		}
	}
}
