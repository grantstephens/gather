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
		return err
	}

	switch activity.Type {
	case "Follow":
		return handleFollow(app, baseURL, activity)
	case "Undo":
		return handleUndo(app, activity)
	default:
		return nil
	}
}

func handleFollow(app core.App, baseURL string, activity IncomingActivity) error {
	actorInfo, err := fetchActor(activity.Actor)
	if err != nil {
		return err
	}

	collection, err := app.FindCollectionByNameOrId("ap_followers")
	if err != nil {
		return err
	}

	existing, _ := app.FindFirstRecordByFilter("ap_followers", "actor_url = {:url}", map[string]any{"url": activity.Actor})
	if existing != nil {
		return nil
	}

	record := core.NewRecord(collection)
	record.Set("actor_url", activity.Actor)
	record.Set("inbox_url", actorInfo.Inbox)
	record.Set("shared_inbox_url", actorInfo.SharedInbox)

	if err := app.Save(record); err != nil {
		return err
	}

	accept := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Accept",
		ID:      fmt.Sprintf("%s/ap/activities/accept/%s", baseURL, record.Id),
		Actor:   baseURL + "/ap/actor",
		Object:  activity,
	}

	go func() {
		var lastErr error
		for attempt, delay := range []time.Duration{0, 5 * time.Second, 30 * time.Second} {
			if delay > 0 {
				time.Sleep(delay)
			}
			if lastErr = DeliverActivity(app, accept, actorInfo.Inbox); lastErr == nil {
				return
			}
			app.Logger().Error("failed to deliver Accept", "attempt", attempt+1, "inbox", actorInfo.Inbox, "error", lastErr)
		}
	}()

	return nil
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
			return nil
		}
		return app.Delete(follower)
	}

	return nil
}

type ActorInfo struct {
	Inbox       string
	SharedInbox string
}

func fetchActor(actorURL string) (*ActorInfo, error) {
	req, err := http.NewRequest("GET", actorURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var actor struct {
		Inbox     string `json:"inbox"`
		Endpoints struct {
			SharedInbox string `json:"sharedInbox"`
		} `json:"endpoints"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		return nil, err
	}

	return &ActorInfo{
		Inbox:       actor.Inbox,
		SharedInbox: actor.Endpoints.SharedInbox,
	}, nil
}
