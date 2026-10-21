package activitypub

// PHASE0 SPIKE — throwaway code to test Mobilizon's relay follow + Announce flow.
// No signature verification, in-memory state. Not for merge.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

var (
	spikeMu        sync.Mutex
	spikeFollowers = map[string]string{}          // follower actor URL -> inbox
	spikeEvents    = map[string]json.RawMessage{} // last path segment of event id -> Event JSON
)

func RelayActorJSON(app core.App, baseURL string) ([]byte, error) {
	settings, err := app.FindFirstRecordByFilter("settings", "id != ''")
	if err != nil {
		return nil, err
	}
	id := baseURL + "/ap/relay"
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
	var a IncomingActivity
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	app.Logger().Info("relay-spike: inbox", "type", a.Type, "actor", a.Actor)
	if a.Type != "Follow" {
		return nil
	}
	go func() {
		info, err := fetchActor(app, baseURL, a.Actor)
		if err != nil {
			app.Logger().Error("relay-spike: fetchActor", "error", err)
			return
		}
		spikeMu.Lock()
		spikeFollowers[a.Actor] = info.Inbox
		spikeMu.Unlock()

		priv, err := loadPrivateKey(app)
		if err != nil {
			app.Logger().Error("relay-spike: key", "error", err)
			return
		}
		relayID := baseURL + "/ap/relay"
		accept := Activity{
			Context: "https://www.w3.org/ns/activitystreams",
			Type:    "Accept",
			ID:      fmt.Sprintf("%s/ap/relay/accept/%d", baseURL, time.Now().UnixNano()),
			Actor:   relayID,
			To:      []string{a.Actor},
			Object:  json.RawMessage(data),
		}
		if err := deliverActivityWithKey(priv, relayID+"#main-key", accept, info.Inbox); err != nil {
			app.Logger().Error("relay-spike: Accept delivery", "error", err)
			return
		}
		app.Logger().Info("relay-spike: Accept sent", "to", info.Inbox)
	}()
	return nil
}

// SpikeEvent returns a stored Event by the last path segment of its id.
func SpikeEvent(id string) (json.RawMessage, bool) {
	spikeMu.Lock()
	defer spikeMu.Unlock()
	e, ok := spikeEvents[id]
	return e, ok
}

// SpikeAnnounce stores the Event JSON (so /ap/events/{id} can serve it) and Announces its id
// from the relay actor to every relay follower.
func SpikeAnnounce(app core.App, baseURL string, event json.RawMessage) error {
	var e struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(event, &e); err != nil || e.ID == "" {
		return fmt.Errorf("event needs an id")
	}
	seg := e.ID[strings.LastIndex(e.ID, "/")+1:]

	spikeMu.Lock()
	spikeEvents[seg] = event
	followers := make(map[string]string, len(spikeFollowers))
	for k, v := range spikeFollowers {
		followers[k] = v
	}
	spikeMu.Unlock()

	priv, err := loadPrivateKey(app)
	if err != nil {
		return err
	}
	relayID := baseURL + "/ap/relay"
	for actor, inbox := range followers {
		announce := Activity{
			Context: "https://www.w3.org/ns/activitystreams",
			Type:    "Announce",
			ID:      fmt.Sprintf("%s/ap/relay/announce/%d", baseURL, time.Now().UnixNano()),
			Actor:   relayID,
			To:      []string{"https://www.w3.org/ns/activitystreams#Public"},
			Cc:      []string{relayID + "/followers", actor},
			Object:  e.ID,
		}
		if err := deliverActivityWithKey(priv, relayID+"#main-key", announce, inbox); err != nil {
			return fmt.Errorf("announce to %s: %w", inbox, err)
		}
	}
	return nil
}
