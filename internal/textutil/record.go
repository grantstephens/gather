package textutil

import "github.com/pocketbase/pocketbase/core"

var decodedFields = map[string][]string{
	"events": {"title", "description"},
	"places": {"name", "address"},
	"tags":   {"name"},
}

// DecodeRecordText decodes stray HTML entities in the record's text fields and
// reports whether anything changed. Shared by the save hooks and the migration.
func DecodeRecordText(rec *core.Record) bool {
	changed := false
	for _, f := range decodedFields[rec.Collection().Name] {
		old := rec.GetString(f)
		if neu := DecodeEntities(old); neu != old {
			rec.Set(f, neu)
			changed = true
		}
	}
	return changed
}

// DecodeAllExisting rewrites every stored event, place and tag whose text still
// contains entities. Idempotent. Slugs are deliberately left alone so
// bookmarked URLs keep working.
func DecodeAllExisting(app core.App) error {
	for _, name := range []string{"events", "places", "tags"} {
		for offset := 0; ; offset += 200 {
			recs, err := app.FindRecordsByFilter(name, "", "created", 200, offset)
			if err != nil {
				return err
			}
			for _, rec := range recs {
				if DecodeRecordText(rec) {
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
	return nil
}
