package activitypub

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

var subjectActorPath = regexp.MustCompile(`/ap/actor/(place|town)/([^/#?]+)$`)

// subjectPrivateKey returns the signing key when actorID is a subject actor.
func subjectPrivateKey(app core.App, actorID string) (*rsa.PrivateKey, bool) {
	m := subjectActorPath.FindStringSubmatch(actorID)
	if m == nil {
		return nil, false
	}
	s, err := FindSubject(app, m[1], m[2])
	if err != nil {
		return nil, false
	}
	priv, _, err := subjectKeys(app, s)
	if err != nil {
		return nil, false
	}
	key, err := ParsePrivateKey(priv)
	if err != nil {
		return nil, false
	}
	return key, true
}

var subjectFollowMu sync.Mutex

func HandleSubjectInbox(app core.App, baseURL, kind, ref string, r *http.Request) error {
	s, err := FindSubject(app, kind, ref)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if err := signatureVerifier(app, baseURL, r, data); err != nil {
		return fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}
	var a IncomingActivity
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	switch a.Type {
	case "Follow":
		go func() {
			if err := handleSubjectFollow(app, baseURL, s, a, data); err != nil {
				app.Logger().Error("ap-subject: failed to handle Follow", "subject", s.Key(), "actor", a.Actor, "error", err)
			}
		}()
	case "Undo":
		go func() {
			if err := handleSubjectUndo(app, s, a); err != nil {
				app.Logger().Error("ap-subject: failed to handle Undo", "subject", s.Key(), "actor", a.Actor, "error", err)
			}
		}()
	}
	return nil
}

func handleSubjectFollow(app core.App, baseURL string, s *Subject, a IncomingActivity, raw []byte) error {
	info, err := fetchActor(app, baseURL, a.Actor)
	if err != nil {
		return fmt.Errorf("fetchActor %s: %w", a.Actor, err)
	}
	if err := saveSubjectFollower(app, s, a.Actor, info); err != nil {
		return err
	}

	id := subjectActorID(baseURL, s.Kind, s.Ref)
	accept := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Accept",
		ID:      fmt.Sprintf("%s/accept/%d", id, time.Now().UnixNano()),
		Actor:   id,
		To:      []string{a.Actor},
		Object:  json.RawMessage(raw),
	}
	return QueueDelivery(app, accept, info.Inbox)
}

func saveSubjectFollower(app core.App, s *Subject, actorURL string, info *ActorInfo) error {
	subjectFollowMu.Lock()
	defer subjectFollowMu.Unlock()

	params := map[string]any{"s": s.Key(), "u": actorURL}
	if existing, _ := app.FindFirstRecordByFilter("ap_subject_followers", "subject = {:s} && actor_url = {:u}", params); existing != nil {
		return nil
	}
	col, err := app.FindCollectionByNameOrId("ap_subject_followers")
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	rec.Set("subject", s.Key())
	rec.Set("actor_url", actorURL)
	rec.Set("inbox_url", info.Inbox)
	rec.Set("shared_inbox_url", info.SharedInbox)
	if err := app.Save(rec); err != nil {
		return fmt.Errorf("save follower: %w", err)
	}
	return nil
}

func handleSubjectUndo(app core.App, s *Subject, a IncomingActivity) error {
	var obj struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(a.Object, &obj); err != nil || obj.Type != "Follow" {
		return nil
	}
	rec, err := app.FindFirstRecordByFilter("ap_subject_followers", "subject = {:s} && actor_url = {:u}", map[string]any{"s": s.Key(), "u": a.Actor})
	if err != nil {
		return nil
	}
	return app.Delete(rec)
}
