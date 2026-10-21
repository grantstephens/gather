package textutil

import (
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

var (
	costLine   = regexp.MustCompile(`(?i)(?:^|<br\s*/?>|\n)[ \t]*Cost[ \t]*:[ \t]*([^<\n]+)`)
	sourceLine = regexp.MustCompile(`(?i)(?:^|<br\s*/?>|\n)[ \t]*Source[ \t]*:[ \t]*(https?://[^\s<]+)`)
)

// ParseBoilerplate extracts the "Cost:" and "Source:" lines the scrapers append
// to descriptions, so they can live in real fields.
func ParseBoilerplate(description string) (cost, sourceURL string) {
	if m := costLine.FindStringSubmatch(description); m != nil {
		cost = strings.TrimSpace(m[1])
	}
	if m := sourceLine.FindStringSubmatch(description); m != nil {
		sourceURL = strings.TrimSpace(m[1])
	}
	return cost, sourceURL
}

var trailingBreaks = regexp.MustCompile(`(?i)(?:<br\s*/?>|\s)+$`)

// StripBoilerplate removes the "Cost:" and "Source:" lines ParseBoilerplate
// recognises, for outputs (e.g. federated events) that carry them elsewhere.
func StripBoilerplate(description string) string {
	s := costLine.ReplaceAllString(description, "")
	s = sourceLine.ReplaceAllString(s, "")
	return trailingBreaks.ReplaceAllString(s, "")
}

// maxCostRunes matches the events.cost field's Max. A longer "Cost:" line must
// be cut, not rejected: validation would otherwise refuse the whole event.
const maxCostRunes = 200

func capRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n]))
}

// FillCostSource sets an event's empty cost/source_url fields from its
// description boilerplate and reports whether anything changed. Explicit
// values are never overwritten.
func FillCostSource(rec *core.Record) bool {
	if rec.Collection().Name != "events" {
		return false
	}
	needCost, needSource := rec.GetString("cost") == "", rec.GetString("source_url") == ""
	if !needCost && !needSource {
		return false
	}
	cost, source := ParseBoilerplate(rec.GetString("description"))
	changed := false
	if needCost && cost != "" {
		rec.Set("cost", capRunes(cost, maxCostRunes))
		changed = true
	}
	if needSource && source != "" {
		rec.Set("source_url", source)
		changed = true
	}
	return changed
}

// BackfillCostSource applies FillCostSource to every existing event. Idempotent.
func BackfillCostSource(app core.App) error {
	for offset := 0; ; offset += 200 {
		recs, err := app.FindRecordsByFilter("events", "", "created", 200, offset)
		if err != nil {
			return err
		}
		for _, rec := range recs {
			if FillCostSource(rec) {
				if err := app.SaveNoValidate(rec); err != nil {
					return err
				}
			}
		}
		if len(recs) < 200 {
			return nil
		}
	}
}
