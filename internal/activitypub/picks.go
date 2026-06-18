package activitypub

import (
	"fmt"
	"html"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func picksToNote(picks *core.Record, baseURL string) Note {
	title := html.EscapeString(picks.GetString("title"))
	blurb := html.EscapeString(picks.GetString("blurb"))
	picksURL := fmt.Sprintf("%s/picks/%s", baseURL, picks.GetString("slug"))

	content := fmt.Sprintf(
		"<p><strong>%s</strong></p><p>%s</p><p><a href=%q>Read more</a></p>",
		title,
		blurb,
		picksURL,
	)

	return Note{
		Type:         "Note",
		ID:           fmt.Sprintf("%s/ap/picks/%s", baseURL, picks.Id),
		AttributedTo: baseURL + "/ap/actor",
		Content:      content,
		Published:    time.Now(),
		URL:          picksURL,
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:           []string{baseURL + "/ap/actor/followers"},
	}
}

func CreateActivityForPicks(picks *core.Record, baseURL string) Activity {
	note := picksToNote(picks, baseURL)
	return Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Create",
		ID:      fmt.Sprintf("%s/ap/activities/picks/%s/%d", baseURL, picks.Id, time.Now().Unix()),
		Actor:   baseURL + "/ap/actor",
		To:      []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:      []string{baseURL + "/ap/actor/followers"},
		Object:  note,
	}
}
