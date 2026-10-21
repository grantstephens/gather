package activitypub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// The relay actor (relay@host, type Application) exists because Mobilizon only lists
// remote events that were Announced by an actor it follows. Mobilizon admins follow
// this actor; every event we publish is then Announced through it.

const publicAudience = "https://www.w3.org/ns/activitystreams#Public"

// announceDelay lets the Create land before the Announce that references it.
const announceDelay = 10 * time.Second

func relayID(baseURL string) string { return baseURL + "/ap/relay" }

func RelayActorJSON(app core.App, baseURL string) ([]byte, error) {
	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		return nil, err
	}
	id := relayID(baseURL)
	actor := map[string]any{
		"@context":          []string{"https://www.w3.org/ns/activitystreams", "https://w3id.org/security/v1"},
		"type":              "Application",
		"id":                id,
		"preferredUsername": "relay",
		"name":              "Gather relay",
		"url":               baseURL,
		"inbox":             id + "/inbox",
		"outbox":            id + "/outbox",
		"followers":         id + "/followers",
		"publicKey": map[string]any{
			"id":           id + "#main-key",
			"owner":        id,
			"publicKeyPem": settings.GetString("ap_public_key"),
		},
	}
	return json.MarshalIndent(actor, "", "  ")
}

func HandleRelayInbox(app core.App, baseURL string, r *http.Request) error {
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
			if err := handleRelayFollow(app, baseURL, a, data); err != nil {
				app.Logger().Error("ap-relay: failed to handle Follow", "actor", a.Actor, "error", err)
			}
		}()
	case "Undo":
		go func() {
			if err := handleRelayUndo(app, a); err != nil {
				app.Logger().Error("ap-relay: failed to handle Undo", "actor", a.Actor, "error", err)
			}
		}()
	default:
		app.Logger().Info("ap-relay: ignoring activity type", "type", a.Type, "actor", a.Actor)
	}
	return nil
}

func handleRelayFollow(app core.App, baseURL string, a IncomingActivity, raw []byte) error {
	info, err := fetchActor(app, baseURL, a.Actor)
	if err != nil {
		return fmt.Errorf("fetchActor %s: %w", a.Actor, err)
	}

	existing, _ := app.FindFirstRecordByFilter("ap_relay_followers", "actor_url = {:u}", map[string]any{"u": a.Actor})
	if existing == nil {
		col, err := app.FindCollectionByNameOrId("ap_relay_followers")
		if err != nil {
			return err
		}
		rec := core.NewRecord(col)
		rec.Set("actor_url", a.Actor)
		rec.Set("inbox_url", info.Inbox)
		rec.Set("shared_inbox_url", info.SharedInbox)
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("save relay follower: %w", err)
		}
	}

	accept := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Accept",
		ID:      fmt.Sprintf("%s/ap/relay/accept/%d", baseURL, time.Now().UnixNano()),
		Actor:   relayID(baseURL),
		To:      []string{a.Actor},
		Object:  json.RawMessage(raw),
	}
	return QueueDelivery(app, accept, info.Inbox)
}

func handleRelayUndo(app core.App, a IncomingActivity) error {
	var obj struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(a.Object, &obj); err != nil || obj.Type != "Follow" {
		return nil
	}
	rec, err := app.FindFirstRecordByFilter("ap_relay_followers", "actor_url = {:u}", map[string]any{"u": a.Actor})
	if err != nil {
		return nil // already gone
	}
	return app.Delete(rec)
}

// QueueRelayEvent sends an event lifecycle activity ("Create", "Update" or "Delete") to every
// relay follower. Create/Update carry the Event, authored by the instance actor, which owns it.
// Create is followed by an Announce from the relay actor, which is what makes Mobilizon list it.
// Delete carries just the event id; Mobilizon re-fetches it and sees the 410.
func QueueRelayEvent(app core.App, baseURL string, event *core.Record, activityType string) error {
	followers, err := app.FindRecordsByFilter("ap_relay_followers", "id != ''", "", 0, 0)
	if err != nil {
		return err
	}
	inboxes := map[string]bool{}
	for _, f := range followers {
		inbox := f.GetString("shared_inbox_url")
		if inbox == "" {
			inbox = f.GetString("inbox_url")
		}
		inboxes[inbox] = true
	}
	if len(inboxes) == 0 {
		return nil
	}

	eventID := fmt.Sprintf("%s/ap/events/%s", baseURL, event.Id)
	var object any = eventID
	if activityType != "Delete" {
		ev, err := eventToAPEvent(app, event, baseURL)
		if err != nil {
			return err
		}
		object = ev
	}
	now := time.Now()
	activity := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    activityType,
		ID:      fmt.Sprintf("%s/ap/activities/%s/%d", baseURL, event.Id, now.UnixNano()),
		Actor:   baseURL + "/ap/actor",
		To:      []string{publicAudience},
		Object:  object,
	}
	announce := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Announce",
		ID:      fmt.Sprintf("%s/ap/relay/announce/%d", baseURL, now.UnixNano()),
		Actor:   relayID(baseURL),
		To:      []string{publicAudience},
		Cc:      []string{relayID(baseURL) + "/followers"},
		Object:  eventID,
	}

	for inbox := range inboxes {
		if err := queueDeliveryAt(app, activity, inbox, now); err != nil {
			return err
		}
		if activityType == "Create" {
			if err := queueDeliveryAt(app, announce, inbox, now.Add(announceDelay)); err != nil {
				return err
			}
		}
	}
	return nil
}
