package activitypub

import (
	"encoding/json"
	"fmt"
	"html"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type OrderedCollection struct {
	Context      string `json:"@context"`
	Type         string `json:"type"`
	ID           string `json:"id"`
	TotalItems   int    `json:"totalItems"`
	OrderedItems []any  `json:"orderedItems"`
}

type Activity struct {
	Context any    `json:"@context"`
	Type    string `json:"type"`
	ID      string `json:"id"`
	Actor   string `json:"actor"`
	Object  any    `json:"object"`
}

type Note struct {
	Context      any       `json:"@context,omitempty"`
	Type         string    `json:"type"`
	ID           string    `json:"id"`
	AttributedTo string    `json:"attributedTo"`
	Content      string    `json:"content"`
	Published    time.Time `json:"published"`
	URL          string    `json:"url"`
	To           []string  `json:"to"`
	Cc           []string  `json:"cc,omitempty"`
	Attachment   []any     `json:"attachment,omitempty"`
	Tag          []any     `json:"tag,omitempty"`
}

func GetOutbox(app core.App, baseURL string) ([]byte, error) {
	events, err := app.FindRecordsByFilter(
		"events",
		"status = 'published'",
		"-start_datetime",
		20,
		0,
	)
	if err != nil {
		return nil, err
	}

	items := make([]any, 0, len(events))
	for _, event := range events {
		note := eventToNote(event, baseURL)
		activity := Activity{
			Type:   "Create",
			ID:     fmt.Sprintf("%s/ap/activities/%s", baseURL, event.Id),
			Actor:  baseURL + "/ap/actor",
			Object: note,
		}
		items = append(items, activity)
	}

	collection := OrderedCollection{
		Context:      "https://www.w3.org/ns/activitystreams",
		Type:         "OrderedCollection",
		ID:           baseURL + "/ap/outbox",
		TotalItems:   len(items),
		OrderedItems: items,
	}

	return json.MarshalIndent(collection, "", "  ")
}

func eventToNote(event *core.Record, baseURL string) Note {
	startTime := event.GetDateTime("start_datetime").Time()

	title := html.EscapeString(event.GetString("title"))
	desc := event.GetString("description")

	content := fmt.Sprintf(
		"<p><strong>%s</strong></p><p>%s</p><p>%s</p>",
		title,
		startTime.Format("Monday, January 2, 2006 at 3:04 PM"),
		desc,
	)

	note := Note{
		Type:         "Note",
		ID:           fmt.Sprintf("%s/ap/events/%s", baseURL, event.Id),
		AttributedTo: baseURL + "/ap/actor",
		Content:      content,
		Published:    event.GetDateTime("created").Time(),
		URL:          fmt.Sprintf("%s/event/%s", baseURL, event.Id),
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:           []string{baseURL + "/ap/actor/followers"},
	}

	return note
}

func CreateActivityForEvent(event *core.Record, baseURL string, activityType string) Activity {
	note := eventToNote(event, baseURL)

	return Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    activityType,
		ID:      fmt.Sprintf("%s/ap/activities/%s/%d", baseURL, event.Id, time.Now().Unix()),
		Actor:   baseURL + "/ap/actor",
		Object:  note,
	}
}
