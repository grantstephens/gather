package activitypub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gather/internal/slug"

	"github.com/pocketbase/pocketbase/core"
)

// QueueSubjectEvent sends an event lifecycle activity ("Create", "Update" or "Delete") to
// followers of the event's place and town actors. Each group owns its own copy of the event
// (attributedTo the group, id under the group), and sends the activity itself, which is
// what Mobilizon needs to show it in a follower's group feed. An inbox that follows both
// gets one copy, from the first subject (place before town) it follows.
func QueueSubjectEvent(app core.App, baseURL string, event *core.Record, activityType string) error {
	seen := map[string]bool{}
	now := time.Now()
	for _, s := range eventSubjects(app, event) {
		followers, err := app.FindRecordsByFilter("ap_subject_followers", "subject = {:s}", "", 0, 0, map[string]any{"s": s.Key()})
		if err != nil {
			return err
		}
		var inboxes []string
		for _, f := range followers {
			inbox := f.GetString("shared_inbox_url")
			if inbox == "" {
				inbox = f.GetString("inbox_url")
			}
			if !seen[inbox] {
				seen[inbox] = true
				inboxes = append(inboxes, inbox)
			}
		}
		if len(inboxes) == 0 {
			continue
		}

		groupID := subjectActorID(baseURL, s.Kind, s.Ref)
		var object any = groupID + "/events/" + event.Id
		if activityType != "Delete" {
			ev, err := eventToAPEventAs(app, event, baseURL, s)
			if err != nil {
				return err
			}
			object = ev
		}
		activity := Activity{
			Context: "https://www.w3.org/ns/activitystreams",
			Type:    activityType,
			ID:      fmt.Sprintf("%s/activities/%s/%d", groupID, event.Id, now.UnixNano()),
			Actor:   baseURL + "/ap/actor",
			To:      []string{publicAudience},
			Cc:      []string{groupID + "/followers"},
			Object:  object,
		}
		for _, inbox := range inboxes {
			if err := queueDeliveryAt(app, activity, inbox, now); err != nil {
				return err
			}
		}
	}
	return nil
}

// SubjectEventObject serves a group's copy of an event: 200 with the Event while the event is
// published or cancelled and belongs to the subject, otherwise 410 with a Tombstone.
func SubjectEventObject(app core.App, baseURL, kind, ref, eventID string) ([]byte, int) {
	objectID := subjectActorID(baseURL, kind, ref) + "/events/" + eventID
	if event, err := app.FindRecordById("events", eventID); err == nil {
		if st := event.GetString("status"); st == "published" || st == "cancelled" {
			for _, s := range eventSubjects(app, event) {
				if s.Kind == kind && s.Ref == ref {
					ap, err := eventToAPEventAs(app, event, baseURL, s)
					if err == nil {
						if body, err := json.Marshal(ap); err == nil {
							return body, http.StatusOK
						}
					}
					return []byte(`{"error":"failed to render event"}`), http.StatusInternalServerError
				}
			}
		}
	}
	body, _ := json.Marshal(map[string]string{
		"@context":   "https://www.w3.org/ns/activitystreams",
		"type":       "Tombstone",
		"id":         objectID,
		"formerType": "Event",
	})
	return body, http.StatusGone
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
