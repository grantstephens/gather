package activitypub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type IncomingActivity struct {
	Type   string          `json:"type"`
	ID     string          `json:"id"`
	Actor  string          `json:"actor"`
	Object json.RawMessage `json:"object"`
}

func HandleInbox(app core.App, baseURL string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	var activity IncomingActivity
	if err := json.Unmarshal(data, &activity); err != nil {
		app.Logger().Error("ap-inbox: failed to parse activity", "error", err, "body", string(data[:min(len(data), 512)]))
		return err
	}

	app.Logger().Info("ap-inbox: received activity", "type", activity.Type, "actor", activity.Actor, "id", activity.ID)

	switch activity.Type {
	case "Follow":
		// Process asynchronously — accept immediately, handle errors in background
		go func() {
			if err := handleFollow(app, baseURL, activity); err != nil {
				app.Logger().Error("ap-inbox: failed to handle Follow", "actor", activity.Actor, "error", err)
			}
		}()
	case "Undo":
		go func() {
			if err := handleUndo(app, activity); err != nil {
				app.Logger().Error("ap-inbox: failed to handle Undo", "actor", activity.Actor, "error", err)
			}
		}()
	default:
		app.Logger().Info("ap-inbox: ignoring activity type", "type", activity.Type, "actor", activity.Actor)
	}

	return nil
}

func handleFollow(app core.App, baseURL string, activity IncomingActivity) error {
	actorInfo, err := fetchActor(app, baseURL, activity.Actor)
	if err != nil {
		return fmt.Errorf("fetchActor %s: %w", activity.Actor, err)
	}

	collection, err := app.FindCollectionByNameOrId("ap_followers")
	if err != nil {
		return err
	}

	existing, _ := app.FindFirstRecordByFilter("ap_followers", "actor_url = {:url}", map[string]any{"url": activity.Actor})
	if existing != nil {
		app.Logger().Info("ap-inbox: already following, re-sending Accept", "actor", activity.Actor)
	} else {
		record := core.NewRecord(collection)
		record.Set("actor_url", activity.Actor)
		record.Set("inbox_url", actorInfo.Inbox)
		record.Set("shared_inbox_url", actorInfo.SharedInbox)

		if err := app.Save(record); err != nil {
			return fmt.Errorf("save follower: %w", err)
		}
		app.Logger().Info("ap-inbox: saved new follower", "actor", activity.Actor, "inbox", actorInfo.Inbox)
	}

	accept := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Accept",
		ID:      fmt.Sprintf("%s/ap/activities/accept/%d", baseURL, time.Now().UnixNano()),
		Actor:   baseURL + "/ap/actor",
		To:      []string{activity.Actor},
		Object:  activity,
	}

	var lastErr error
	for attempt, delay := range []time.Duration{0, 5 * time.Second, 30 * time.Second} {
		if delay > 0 {
			time.Sleep(delay)
		}
		if lastErr = DeliverActivity(app, accept, actorInfo.Inbox); lastErr == nil {
			app.Logger().Info("ap-inbox: Accept delivered", "actor", activity.Actor, "inbox", actorInfo.Inbox)
			return nil
		}
		app.Logger().Error("ap-inbox: Accept delivery failed", "attempt", attempt+1, "inbox", actorInfo.Inbox, "error", lastErr)
	}
	return fmt.Errorf("Accept delivery exhausted retries: %w", lastErr)
}

func handleUndo(app core.App, activity IncomingActivity) error {
	var undoObject struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(activity.Object, &undoObject); err != nil {
		return err
	}

	if undoObject.Type == "Follow" {
		follower, err := app.FindFirstRecordByFilter("ap_followers", "actor_url = {:url}", map[string]any{"url": activity.Actor})
		if err != nil {
			return nil // already gone
		}
		if err := app.Delete(follower); err != nil {
			return err
		}
		app.Logger().Info("ap-inbox: removed follower", "actor", activity.Actor)
	}

	return nil
}

type ActorInfo struct {
	Inbox       string
	SharedInbox string
}

func fetchActor(app core.App, baseURL, actorURL string) (*ActorInfo, error) {
	req, err := http.NewRequest("GET", actorURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json")

	// Sign the GET request — GoToSocial and others with "authorized fetch" enabled
	// return 401 for unsigned actor lookups.
	if err := signGetRequest(app, req, baseURL+"/ap/actor#main-key"); err != nil {
		return nil, fmt.Errorf("sign actor fetch: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("actor fetch returned %d", resp.StatusCode)
	}

	var actor struct {
		Inbox     string `json:"inbox"`
		Endpoints struct {
			SharedInbox string `json:"sharedInbox"`
		} `json:"endpoints"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		return nil, fmt.Errorf("decode actor JSON: %w", err)
	}

	if actor.Inbox == "" {
		return nil, fmt.Errorf("actor has no inbox URL")
	}

	return &ActorInfo{
		Inbox:       actor.Inbox,
		SharedInbox: actor.Endpoints.SharedInbox,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
