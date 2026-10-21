package activitypub

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"gather/internal/slug"

	"github.com/pocketbase/pocketbase/core"
)

// Subject actors: a Group per approved place and per town (city), so people can follow
// @venue@host or @town-name@host instead of the whole instance. Nothing is pre-created;
// the keypair is generated on first request for a subject that resolves.

var ErrSubjectNotFound = errors.New("subject not found")

// Subject is what a Group actor stands for.
type Subject struct {
	Kind     string // "place" or "town"
	Ref      string // place record id, or city slug
	Name     string
	Summary  string
	Username string // WebFinger handle
}

// Key is how subjects are stored: "place:{id}" or "town:{citySlug}".
func (s Subject) Key() string { return s.Kind + ":" + s.Ref }

func subjectActorID(baseURL, kind, ref string) string {
	return fmt.Sprintf("%s/ap/actor/%s/%s", baseURL, kind, ref)
}

// FindSubject resolves an approved place or an existing town. Pending places never resolve.
func FindSubject(app core.App, kind, ref string) (*Subject, error) {
	switch kind {
	case "place":
		p, err := app.FindRecordById("places", ref)
		if err != nil || p.GetString("status") != "approved" {
			return nil, ErrSubjectNotFound
		}
		return placeSubject(p.Id, p.GetString("name"), p.GetString("description"), p.GetString("slug")), nil
	case "town":
		places, err := app.FindRecordsByFilter("places", "status = 'approved' && city != ''", "", 0, 0)
		if err != nil {
			return nil, err
		}
		for _, p := range places {
			if city := p.GetString("city"); slug.Slugify(city) == ref && ref != "" {
				return &Subject{Kind: "town", Ref: ref, Name: city, Summary: "Events in " + city, Username: "town-" + ref}, nil
			}
		}
	}
	return nil, ErrSubjectNotFound
}

func placeSubject(id, name, description, placeSlug string) *Subject {
	username := placeSlug
	if username == "" {
		username = id
	}
	return &Subject{Kind: "place", Ref: id, Name: name, Summary: description, Username: username}
}

// FindSubjectByUsername resolves a WebFinger handle: a place slug, or "town-{citySlug}".
func FindSubjectByUsername(app core.App, username string) (*Subject, error) {
	if ref, ok := strings.CutPrefix(username, "town-"); ok {
		if s, err := FindSubject(app, "town", ref); err == nil {
			return s, nil
		}
	}
	p, err := app.FindFirstRecordByFilter("places", "slug = {:s} && status = 'approved'", map[string]any{"s": username})
	if err != nil {
		return nil, ErrSubjectNotFound
	}
	return FindSubject(app, "place", p.Id)
}

var subjectKeyMu sync.Mutex

// subjectKeys returns the subject's keypair, generating it on the first call.
func subjectKeys(app core.App, s *Subject) (privatePEM, publicPEM string, err error) {
	subjectKeyMu.Lock()
	defer subjectKeyMu.Unlock()

	if rec, err := app.FindFirstRecordByFilter("ap_actors", "subject = {:s}", map[string]any{"s": s.Key()}); err == nil {
		return rec.GetString("private_key"), rec.GetString("public_key"), nil
	}
	privatePEM, publicPEM, err = GenerateKeyPair()
	if err != nil {
		return "", "", err
	}
	col, err := app.FindCollectionByNameOrId("ap_actors")
	if err != nil {
		return "", "", err
	}
	rec := core.NewRecord(col)
	rec.Set("subject", s.Key())
	rec.Set("private_key", privatePEM)
	rec.Set("public_key", publicPEM)
	if err := app.Save(rec); err != nil {
		return "", "", err
	}
	return privatePEM, publicPEM, nil
}

func SubjectActorJSON(app core.App, baseURL, kind, ref string) ([]byte, error) {
	s, err := FindSubject(app, kind, ref)
	if err != nil {
		return nil, err
	}
	_, pub, err := subjectKeys(app, s)
	if err != nil {
		return nil, err
	}
	id := subjectActorID(baseURL, s.Kind, s.Ref)
	actor := &Actor{
		Context:           []any{"https://www.w3.org/ns/activitystreams", "https://w3id.org/security/v1"},
		Type:              "Group",
		ID:                id,
		PreferredUsername: s.Username,
		Name:              s.Name,
		Summary:           s.Summary,
		URL:               baseURL,
		Inbox:             id + "/inbox",
		Outbox:            id + "/outbox",
		Followers:         id + "/followers",
		Discoverable:      true,
		PublicKey:         &PublicKey{ID: id + "#main-key", Owner: id, PublicKeyPem: pub},
	}
	return actor.ToJSON()
}

// HandleWebfingerFor resolves the instance and relay actors, then place and town actors.
func HandleWebfingerFor(app core.App, resource, baseURL string) ([]byte, error) {
	if b, err := HandleWebfinger(resource, baseURL); err == nil {
		return b, nil
	}
	name, _, ok := strings.Cut(strings.TrimPrefix(resource, "acct:"), "@")
	if !ok || !strings.HasPrefix(resource, "acct:") {
		return nil, fmt.Errorf("invalid resource format")
	}
	s, err := FindSubjectByUsername(app, name)
	if err != nil {
		return nil, err
	}
	return marshalWebfinger(resource, subjectActorID(baseURL, s.Kind, s.Ref))
}
