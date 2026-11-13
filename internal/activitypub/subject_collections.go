package activitypub

import (
	"encoding/json"

	dbx "github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// SubjectCollection serves a subject actor's "followers" or "outbox" collection. The outbox
// is an empty collection: events reach followers by delivery (Create + Announce), and
// the canonical objects live at /ap/events/{id}.
func SubjectCollection(app core.App, baseURL, kind, ref, which string) ([]byte, error) {
	s, err := FindSubject(app, kind, ref)
	if err != nil {
		return nil, err
	}
	total := 0
	if which == "followers" {
		n, err := app.CountRecords("ap_subject_followers", dbx.HashExp{"subject": s.Key()})
		if err != nil {
			return nil, err
		}
		total = int(n)
	}
	return json.Marshal(OrderedCollection{
		Context:    "https://www.w3.org/ns/activitystreams",
		Type:       "OrderedCollection",
		ID:         subjectActorID(baseURL, s.Kind, s.Ref) + "/" + which,
		TotalItems: total,
	})
}
