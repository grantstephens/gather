package activitypub

import (
	"gather/internal/slug"

	"github.com/pocketbase/pocketbase/core"
)

// QueueSubjectEvent sends an event lifecycle activity to followers of the event's place and town
// actors. An inbox that follows both gets it once; the Announce comes from the first subject
// (place before town) that inbox follows.
func QueueSubjectEvent(app core.App, baseURL string, event *core.Record, activityType string) error {
	subjects := eventSubjects(app, event)
	if len(subjects) == 0 {
		return nil
	}

	seen := map[string]bool{}
	var targets []eventTarget
	for _, s := range subjects {
		followers, err := app.FindRecordsByFilter("ap_subject_followers", "subject = {:s}", "", 0, 0, map[string]any{"s": s.Key()})
		if err != nil {
			return err
		}
		for _, f := range followers {
			inbox := f.GetString("shared_inbox_url")
			if inbox == "" {
				inbox = f.GetString("inbox_url")
			}
			if seen[inbox] {
				continue
			}
			seen[inbox] = true
			targets = append(targets, eventTarget{inbox: inbox, announcer: subjectActorID(baseURL, s.Kind, s.Ref)})
		}
	}
	return queueEventTargets(app, baseURL, event, activityType, targets)
}

// eventSubjects returns the place and town actors an event belongs to, if its place is approved.
func eventSubjects(app core.App, event *core.Record) []*Subject {
	placeID := event.GetString("place")
	if placeID == "" {
		return nil
	}
	place, err := FindSubject(app, "place", placeID)
	if err != nil {
		return nil
	}
	subjects := []*Subject{place}
	if rec, err := app.FindRecordById("places", placeID); err == nil {
		if town, err := FindSubject(app, "town", slug.Slugify(rec.GetString("city"))); err == nil {
			subjects = append(subjects, town)
		}
	}
	return subjects
}
