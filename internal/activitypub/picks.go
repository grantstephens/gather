package activitypub

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"gather/internal/seo"

	"github.com/pocketbase/pocketbase/core"
)

const picksIntroMaxLen = 500

var paragraphBreak = regexp.MustCompile(`\n\s*\n`)

// picksIntro returns the first paragraph of a Markdown blurb as plain text, at
// most maxLen runes, cut at a sentence end if one falls in the back two thirds,
// otherwise at a word boundary with an ellipsis.
func picksIntro(blurb string, maxLen int) string {
	first := paragraphBreak.Split(strings.TrimSpace(blurb), 2)[0]
	text := []rune(seo.StripMarkdownInline(first))
	if len(text) <= maxLen {
		return string(text)
	}
	cut := text[:maxLen]
	if i := strings.LastIndexAny(string(cut), ".!?"); i > maxLen/3 {
		return strings.TrimSpace(string(cut)[:i+1])
	}
	if i := strings.LastIndex(string(cut), " "); i > 0 {
		return strings.TrimSpace(string(cut)[:i]) + "…"
	}
	return string(cut) + "…"
}

func picksToNote(picks *core.Record, baseURL string) Note {
	title := html.EscapeString(picks.GetString("title"))
	blurb := html.EscapeString(picksIntro(picks.GetString("blurb"), picksIntroMaxLen))
	picksURL := fmt.Sprintf("%s/picks/%s", baseURL, picks.GetString("slug"))

	content := fmt.Sprintf(
		"<p><strong>%s</strong></p><p>%s</p><p><a href=%q>Read more</a></p>",
		title,
		blurb,
		picksURL,
	)

	published := picks.GetDateTime("created").Time()
	if published.IsZero() {
		published = time.Now()
	}

	return Note{
		Type:         "Note",
		ID:           fmt.Sprintf("%s/ap/picks/%s", baseURL, picks.Id),
		AttributedTo: baseURL + "/ap/actor",
		Content:      content,
		Published:    published,
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
