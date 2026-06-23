package activitypub

import (
	"encoding/json"
	"fmt"
	"html"
	"path/filepath"
	"strings"
	"time"

	dbx "github.com/pocketbase/dbx"
	"gather/internal/recurrence"
	"github.com/pocketbase/pocketbase/core"
)

type OrderedCollection struct {
	Context    string `json:"@context"`
	Type       string `json:"type"`
	ID         string `json:"id"`
	TotalItems int    `json:"totalItems"`
	First      string `json:"first,omitempty"`
}

type OrderedCollectionPage struct {
	Context      string `json:"@context"`
	Type         string `json:"type"`
	ID           string `json:"id"`
	PartOf       string `json:"partOf"`
	Next         string `json:"next,omitempty"`
	Prev         string `json:"prev,omitempty"`
	OrderedItems []any  `json:"orderedItems"`
}

type Activity struct {
	Context any      `json:"@context"`
	Type    string   `json:"type"`
	ID      string   `json:"id"`
	Actor   string   `json:"actor"`
	To      []string `json:"to,omitempty"`
	Cc      []string `json:"cc,omitempty"`
	Object  any      `json:"object"`
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

const outboxPageSize = 20

func GetOutbox(app core.App, baseURL string, page int) ([]byte, error) {
	outboxURL := baseURL + "/ap/outbox"

	if page == 0 {
		total, err := app.CountRecords("events", dbx.HashExp{"status": "published"})
		if err != nil {
			return nil, err
		}
		collection := OrderedCollection{
			Context:    "https://www.w3.org/ns/activitystreams",
			Type:       "OrderedCollection",
			ID:         outboxURL,
			TotalItems: int(total),
			First:      outboxURL + "?page=1",
		}
		return json.MarshalIndent(collection, "", "  ")
	}

	offset := (page - 1) * outboxPageSize
	events, err := app.FindRecordsByFilter(
		"events",
		"status = 'published'",
		"-id",
		outboxPageSize,
		offset,
	)
	if err != nil {
		return nil, err
	}

	items := make([]any, 0, len(events))
	for _, event := range events {
		note := eventToNote(event, baseURL)
		activity := Activity{
			Context: "https://www.w3.org/ns/activitystreams",
			Type:    "Create",
			ID:      fmt.Sprintf("%s/ap/activities/%s", baseURL, event.Id),
			Actor:   baseURL + "/ap/actor",
			Object:  note,
		}
		items = append(items, activity)
	}

	pageURL := fmt.Sprintf("%s?page=%d", outboxURL, page)
	result := OrderedCollectionPage{
		Context:      "https://www.w3.org/ns/activitystreams",
		Type:         "OrderedCollectionPage",
		ID:           pageURL,
		PartOf:       outboxURL,
		OrderedItems: items,
	}
	if len(events) == outboxPageSize {
		result.Next = fmt.Sprintf("%s?page=%d", outboxURL, page+1)
	}
	if page > 1 {
		result.Prev = fmt.Sprintf("%s?page=%d", outboxURL, page-1)
	}

	return json.MarshalIndent(result, "", "  ")
}

func eventToNote(event *core.Record, baseURL string) Note {
	startTime := event.GetDateTime("start_datetime").Time()

	title := html.EscapeString(event.GetString("title"))
	desc := event.GetString("description")

	recurrenceInfo := ""
	if rule := event.GetString("recurrence_rule"); rule != "" {
		recurrenceInfo = fmt.Sprintf("<p><em>%s</em></p>", html.EscapeString(recurrence.HumanLabel(rule)))
	}

	dateStr := ""
	if !startTime.IsZero() {
		dateStr = startTime.Local().Format("Monday, January 2, 2006 at 3:04 PM MST")
	}

	content := fmt.Sprintf(
		"<p><strong>%s</strong></p><p>%s</p>%s<p>%s</p>",
		title,
		dateStr,
		recurrenceInfo,
		desc,
	)

	// Published is when the Note was posted to the fediverse, not when the event occurs.
	// Mastodon sorts by this field, so using start_datetime would push future events
	// to the wrong position in followers' feeds.
	published := event.GetDateTime("created").Time()
	if published.IsZero() {
		published = time.Now()
	}

	note := Note{
		Type:         "Note",
		ID:           fmt.Sprintf("%s/ap/events/%s", baseURL, event.Id),
		AttributedTo: baseURL + "/ap/actor",
		Content:      content,
		Published:    published,
		URL:          fmt.Sprintf("%s/event/%s", baseURL, event.Id),
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:           []string{baseURL + "/ap/actor/followers"},
	}

	// Attach event image if present. All uploaded images are converted to WebP,
	// so we always reference the .webp file regardless of the stored filename.
	if image := event.GetString("image"); image != "" {
		ext := strings.ToLower(filepath.Ext(image))
		if ext != ".webp" && ext != ".svg" {
			image = strings.TrimSuffix(image, filepath.Ext(image)) + ".webp"
		}
		imageURL := fmt.Sprintf("%s/api/files/%s/%s", baseURL, event.BaseFilesPath(), image)
		mediaType := "image/webp"
		if ext == ".svg" {
			mediaType = "image/svg+xml"
		}
		note.Attachment = []any{APImage{
			Type:      "Document",
			MediaType: mediaType,
			URL:       imageURL,
		}}
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
		To:      []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:      []string{baseURL + "/ap/actor/followers"},
		Object:  note,
	}
}
