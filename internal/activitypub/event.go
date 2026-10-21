package activitypub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pocketbase/pocketbase/core"

	"gather/internal/recurrence"
	"gather/internal/textutil"
)

const defaultTimezone = "Europe/London"

// maxHashtagLen is Mobilizon's tag limit; longer tags are dropped on receipt.
const maxHashtagLen = 40

// Coord marshals as a JSON float even when whole. Mobilizon ignores
// coordinates that arrive as JSON integers.
type Coord float64

func (c Coord) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(c), 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return []byte(s), nil
}

type APPostalAddress struct {
	Type            string `json:"type"`
	StreetAddress   string `json:"streetAddress,omitempty"`
	AddressLocality string `json:"addressLocality,omitempty"`
	AddressCountry  string `json:"addressCountry,omitempty"`
}

type APPlace struct {
	Type      string           `json:"type"`
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Latitude  *Coord           `json:"latitude,omitempty"`
	Longitude *Coord           `json:"longitude,omitempty"`
	Address   *APPostalAddress `json:"address,omitempty"`
}

type APHashtag struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// APAttachment is a media attachment. Mobilizon requires Name.
type APAttachment struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	MediaType string `json:"mediaType,omitempty"`
	URL       string `json:"url"`
}

// APEvent is the ActivityStreams Event shape Mobilizon ingests.
type APEvent struct {
	Context                  any            `json:"@context"`
	Type                     string         `json:"type"`
	ID                       string         `json:"id"`
	URL                      string         `json:"url"`
	AttributedTo             string         `json:"attributedTo"`
	To                       []string       `json:"to"`
	Cc                       []string       `json:"cc,omitempty"`
	Name                     string         `json:"name"`
	Content                  string         `json:"content"`
	MediaType                string         `json:"mediaType"`
	StartTime                string         `json:"startTime"`
	EndTime                  string         `json:"endTime,omitempty"`
	Timezone                 string         `json:"timezone"`
	Published                time.Time      `json:"published"`
	Updated                  time.Time      `json:"updated"`
	Status                   string         `json:"status"`
	JoinMode                 string         `json:"joinMode"`
	ExternalParticipationURL string         `json:"externalParticipationUrl"`
	IsOnline                 bool           `json:"isOnline"`
	InLanguage               string         `json:"inLanguage"`
	Location                 *APPlace       `json:"location,omitempty"`
	Tag                      []APHashtag    `json:"tag,omitempty"`
	Attachment               []APAttachment `json:"attachment,omitempty"`
}

// eventContext defines the extension terms Mobilizon's own Events use, copied
// from a real Mobilizon 5.2.2 Event (testdata/mobilizon_event.json). Note the
// mz namespace is mobilizon.org, not the joinmobilizon.org the docs mention.
var eventContext = []any{
	"https://www.w3.org/ns/activitystreams",
	map[string]any{
		"sc":                       "http://schema.org#",
		"mz":                       "https://mobilizon.org/ns#",
		"ical":                     "http://www.w3.org/2002/12/cal/ical#",
		"Hashtag":                  "as:Hashtag",
		"PostalAddress":            "sc:PostalAddress",
		"streetAddress":            "sc:streetAddress",
		"addressLocality":          "sc:addressLocality",
		"addressCountry":           "sc:addressCountry",
		"inLanguage":               "sc:inLanguage",
		"address":                  map[string]string{"@id": "sc:address", "@type": "sc:PostalAddress"},
		"location":                 map[string]string{"@id": "sc:location", "@type": "sc:Place"},
		"status":                   map[string]string{"@id": "ical:status", "@type": "ical:status"},
		"timezone":                 map[string]string{"@id": "mz:timezone", "@type": "sc:Text"},
		"isOnline":                 map[string]string{"@id": "mz:isOnline", "@type": "sc:Boolean"},
		"joinModeType":             map[string]string{"@id": "mz:joinModeType", "@type": "rdfs:Class"},
		"joinMode":                 map[string]string{"@id": "mz:joinMode", "@type": "mz:joinModeType"},
		"externalParticipationUrl": map[string]string{"@id": "mz:externalParticipationUrl", "@type": "sc:URL"},
	},
}

func instanceTimezone(app core.App) *time.Location {
	name := defaultTimezone
	if s, err := app.FindFirstRecordByFilter("settings", "id != ''"); err == nil {
		if tz := s.GetString("timezone"); tz != "" {
			name = tz
		}
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc, _ = time.LoadLocation(defaultTimezone)
	}
	return loc
}

func eventURL(event *core.Record, baseURL string) string {
	path := event.GetString("slug")
	if path == "" {
		path = event.Id
	}
	return fmt.Sprintf("%s/event/%s", baseURL, path)
}

func eventToAPEvent(app core.App, event *core.Record, baseURL string) (APEvent, error) {
	loc := instanceTimezone(app)
	pageURL := eventURL(event, baseURL)

	content := textutil.StripBoilerplate(event.GetString("description"))
	if rule := event.GetString("recurrence_rule"); rule != "" {
		content += fmt.Sprintf("<p><em>%s</em></p>", recurrence.HumanLabel(rule))
	}
	content += fmt.Sprintf(`<p><a href="%s">%s</a></p>`, pageURL, pageURL)

	published := event.GetDateTime("created").Time()
	if published.IsZero() {
		published = time.Now()
	}
	updated := event.GetDateTime("updated").Time()
	if updated.IsZero() {
		updated = published
	}

	status := "CONFIRMED"
	if event.GetString("status") == "cancelled" {
		status = "CANCELLED"
	}

	out := APEvent{
		Context:                  eventContext,
		Type:                     "Event",
		ID:                       fmt.Sprintf("%s/ap/events/%s", baseURL, event.Id),
		URL:                      pageURL,
		AttributedTo:             baseURL + "/ap/actor",
		To:                       []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:                       []string{baseURL + "/ap/actor/followers"},
		Name:                     event.GetString("title"),
		Content:                  content,
		MediaType:                "text/html",
		StartTime:                event.GetDateTime("start_datetime").Time().In(loc).Format(time.RFC3339),
		Timezone:                 loc.String(),
		Published:                published,
		Updated:                  updated,
		Status:                   status,
		JoinMode:                 "external",
		ExternalParticipationURL: pageURL,
		InLanguage:               "en",
	}
	if end := event.GetDateTime("end_datetime").Time(); !end.IsZero() {
		out.EndTime = end.In(loc).Format(time.RFC3339)
	}

	if placeID := event.GetString("place"); placeID != "" {
		if place, err := app.FindRecordById("places", placeID); err == nil && place.GetString("status") == "approved" {
			out.Location = placeToAP(place, baseURL)
		}
	}

	for _, tagID := range event.GetStringSlice("tags") {
		tag, err := app.FindRecordById("tags", tagID)
		if err != nil || tag.GetString("status") != "approved" {
			continue
		}
		name := tag.GetString("name")
		if utf8.RuneCountInString(name) > maxHashtagLen {
			continue
		}
		out.Tag = append(out.Tag, APHashtag{Type: "Hashtag", Name: "#" + strings.ReplaceAll(name, " ", "")})
	}

	// Uploads are converted to WebP, so always reference the .webp file.
	if image := event.GetString("image"); image != "" {
		if ext := strings.ToLower(filepath.Ext(image)); ext != ".webp" {
			image = strings.TrimSuffix(image, filepath.Ext(image)) + ".webp"
		}
		out.Attachment = []APAttachment{{
			Type:      "Document",
			Name:      "Banner",
			MediaType: "image/webp",
			URL:       fmt.Sprintf("%s/api/files/%s/%s", baseURL, event.BaseFilesPath(), image),
		}}
	}

	return out, nil
}

// EventObject renders the dereferenceable object at /ap/events/{id}: the Event
// for published and cancelled events, a 410 Tombstone otherwise. Remote servers
// fetch this URL to confirm a Delete, so a gone event must answer 410 rather
// than falling through to the SPA's HTML. Tombstones carry no event details.
func EventObject(app core.App, baseURL, id string) ([]byte, int) {
	if event, err := app.FindRecordById("events", id); err == nil {
		if s := event.GetString("status"); s == "published" || s == "cancelled" {
			if ap, err := eventToAPEvent(app, event, baseURL); err == nil {
				if body, err := json.Marshal(ap); err == nil {
					return body, http.StatusOK
				}
			}
			return []byte(`{"error":"failed to render event"}`), http.StatusInternalServerError
		}
	}
	body, _ := json.Marshal(map[string]string{
		"@context":   "https://www.w3.org/ns/activitystreams",
		"type":       "Tombstone",
		"id":         fmt.Sprintf("%s/ap/events/%s", baseURL, id),
		"formerType": "Event",
	})
	return body, http.StatusGone
}

func placeToAP(place *core.Record, baseURL string) *APPlace {
	p := &APPlace{
		Type: "Place",
		ID:   fmt.Sprintf("%s/ap/places/%s", baseURL, place.Id),
		Name: place.GetString("name"),
		Address: &APPostalAddress{
			Type:            "PostalAddress",
			StreetAddress:   place.GetString("address"),
			AddressLocality: place.GetString("city"),
			AddressCountry:  place.GetString("country_code"),
		},
	}
	if gp := place.GetGeoPoint("location"); gp.Lat != 0 || gp.Lon != 0 {
		lat, lon := Coord(gp.Lat), Coord(gp.Lon)
		p.Latitude, p.Longitude = &lat, &lon
	}
	return p
}
