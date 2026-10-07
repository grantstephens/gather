package seo

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Link is a labelled URL used for venue, tag, series and related-event links.
type Link struct{ Text, URL string }

// EventPage is everything the bot-facing event page shows. It is assembled by
// BuildEventPage (database) and turned into HTML by RenderEventHTML (pure).
type EventPage struct {
	Title, InstanceName, BaseURL string
	URL                          string
	Description                  string // plain text, paragraphs separated by \n
	MetaDescription              string
	State                        State
	StartISO, DateText           string
	Venue                        *Link // URL is "" when the place page would 404
	Address                      string
	Tags                         []Link // only tags that have a live page
	Cost, BookingURL             string
	OtherDates                   []Link
	MoreAtVenue                  []Link
	Alternatives                 []Link // shown when the event has ended
	FeaturedIn                   *Link
	SeriesLabel                  string // "Sat 10 Oct" when upcoming siblings exist
	MetaTags, JSONLD             string
}

// NoIndex reports whether crawlers should be told to drop this page.
func (p *EventPage) NoIndex() bool { return p.State == Retired }

func (p *EventPage) pageTitle() string {
	t := p.Title
	if p.SeriesLabel != "" {
		t += " – " + p.SeriesLabel
	}
	return t
}

// RenderEventHTML renders the server-side page crawlers receive.
func RenderEventHTML(p *EventPage) string {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }

	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("  <meta charset=\"UTF-8\">\n")
	b.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	w("  <title>%s - %s</title>\n", htmlEscape(p.pageTitle()), htmlEscape(p.InstanceName))
	w("  <link rel=\"canonical\" href=\"%s\">\n", htmlEscape(p.URL))
	if p.MetaDescription != "" {
		w("  <meta name=\"description\" content=\"%s\">\n", htmlEscape(p.MetaDescription))
	}
	if p.NoIndex() {
		b.WriteString("  <meta name=\"robots\" content=\"noindex,follow\">\n")
	}
	if strings.TrimSpace(p.MetaTags) != "" {
		b.WriteString("  " + strings.ReplaceAll(strings.TrimSpace(p.MetaTags), "\n", "\n  ") + "\n")
	}
	b.WriteString("  <script type=\"application/ld+json\">\n" + p.JSONLD + "\n  </script>\n")
	b.WriteString("</head>\n<body>\n")

	w("  <nav><a href=\"%s/\">Home</a> <a href=\"%s/picks\">Weekend picks</a></nav>\n", htmlEscape(p.BaseURL), htmlEscape(p.BaseURL))
	b.WriteString("  <main>\n  <article>\n")
	w("    <h1>%s</h1>\n", htmlEscape(p.Title))

	if p.State != Upcoming {
		b.WriteString("    <p><strong>This event has ended.</strong> It is kept here for reference.</p>\n")
		writeLinks(&b, "Coming up", p.Alternatives)
	}
	if p.DateText != "" {
		if p.StartISO != "" {
			w("    <p><time datetime=\"%s\">%s</time></p>\n", htmlEscape(p.StartISO), htmlEscape(p.DateText))
		} else {
			w("    <p>%s</p>\n", htmlEscape(p.DateText))
		}
	}
	if p.Venue != nil {
		venue := htmlEscape(p.Venue.Text)
		if p.Venue.URL != "" {
			venue = fmt.Sprintf("<a href=\"%s\">%s</a>", htmlEscape(p.Venue.URL), venue)
		}
		if p.Address != "" {
			w("    <p>Venue: %s, %s</p>\n", venue, htmlEscape(p.Address))
		} else {
			w("    <p>Venue: %s</p>\n", venue)
		}
	}
	if len(p.Tags) > 0 {
		var parts []string
		for _, t := range p.Tags {
			parts = append(parts, fmt.Sprintf("<a href=\"%s\">%s</a>", htmlEscape(t.URL), htmlEscape(t.Text)))
		}
		w("    <p>Categories: %s</p>\n", strings.Join(parts, ", "))
	}
	for _, para := range strings.Split(p.Description, "\n") {
		if para = strings.TrimSpace(para); para != "" {
			w("    <p>%s</p>\n", htmlEscape(para))
		}
	}
	if p.Cost != "" {
		w("    <p>Cost: %s</p>\n", htmlEscape(p.Cost))
	}
	if p.BookingURL != "" {
		w("    <p><a href=\"%s\" rel=\"noopener\">Booking and more information</a></p>\n", htmlEscape(p.BookingURL))
	}
	writeLinks(&b, "Other dates", p.OtherDates)
	if p.Venue != nil {
		writeLinks(&b, "More at "+p.Venue.Text, p.MoreAtVenue)
	}
	if p.FeaturedIn != nil {
		w("    <p>Featured in <a href=\"%s\">%s</a></p>\n", htmlEscape(p.FeaturedIn.URL), htmlEscape(p.FeaturedIn.Text))
	}
	b.WriteString("  </article>\n  </main>\n</body>\n</html>\n")
	return b.String()
}

func writeLinks(b *strings.Builder, heading string, links []Link) {
	if len(links) == 0 {
		return
	}
	fmt.Fprintf(b, "    <h2>%s</h2>\n    <ul>\n", htmlEscape(heading))
	for _, l := range links {
		fmt.Fprintf(b, "      <li><a href=\"%s\">%s</a></li>\n", htmlEscape(l.URL), htmlEscape(l.Text))
	}
	b.WriteString("    </ul>\n")
}

// dateRangeText renders "Saturday 10 October 2026, 7:30 PM – 9:00 PM" (London time).
func dateRangeText(start, end time.Time) string {
	s := longDateTime(start)
	if end.IsZero() || !end.After(start) {
		return s
	}
	ls, le := londonTime(start), londonTime(end)
	if ls.Format("2006-01-02") == le.Format("2006-01-02") {
		return s + " – " + le.Format("3:04 PM")
	}
	return s + " – " + longDateTime(end)
}

// BuildEventPage gathers everything the event page shows. Links to a place or
// tag are only emitted when that page exists (has upcoming events); otherwise
// the name is plain text, so crawlers are never sent to a 404.
func BuildEventPage(app core.App, event *core.Record, baseURL string, now time.Time) (*EventPage, error) {
	instanceName := "Gather"
	if settings, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if name := settings.GetString("instance_name"); name != "" {
			instanceName = name
		}
	}
	start := event.GetDateTime("start_datetime").Time()
	end := event.GetDateTime("end_datetime").Time()
	title := event.GetString("title")

	p := &EventPage{
		Title: title, InstanceName: instanceName, BaseURL: baseURL,
		URL:         eventPageURL(baseURL, event),
		Description: stripMarkdown(event.GetString("description")),
		State:       EventLifecycle(event, now),
		Cost:        event.GetString("cost"), BookingURL: event.GetString("source_url"),
	}
	if !start.IsZero() {
		p.StartISO = formatDateTime(event.GetDateTime("start_datetime"))
		p.DateText = dateRangeText(start, end)
	}

	nowStr := now.UTC().Format("2006-01-02 15:04:05")
	upcoming := func(extra string, limit int, params dbx.Params) []*core.Record {
		if params == nil {
			params = dbx.Params{}
		}
		params["now"] = nowStr
		filter := "status = 'published' && start_datetime >= {:now}"
		if extra != "" {
			filter += " && " + extra
		}
		recs, _ := app.FindRecordsByFilter("events", filter, "start_datetime", limit, 0, params)
		return recs
	}
	eventLink := func(rec *core.Record, text string) Link {
		return Link{Text: text, URL: eventPageURL(baseURL, rec)}
	}

	// venue
	var place *core.Record
	placeName, placeCity := "", ""
	if pid := event.GetString("place"); pid != "" {
		if rec, err := app.FindRecordById("places", pid); err == nil {
			place, placeName, placeCity = rec, rec.GetString("name"), rec.GetString("city")
		}
	}
	if place != nil {
		v := &Link{Text: placeName}
		if place.GetString("status") == "approved" && len(upcoming("place = {:p}", 1, dbx.Params{"p": place.Id})) > 0 {
			v.URL = baseURL + "/place/" + PlaceRef(place)
		}
		p.Venue, p.Address = v, place.GetString("address")
	}

	// tags
	for _, id := range event.GetStringSlice("tags") {
		tag, err := app.FindRecordById("tags", id)
		if err != nil || tag.GetString("status") != "approved" {
			continue
		}
		if len(upcoming("tags ?~ {:t}", 1, dbx.Params{"t": tag.Id})) == 0 {
			continue
		}
		p.Tags = append(p.Tags, Link{Text: tag.GetString("name"), URL: tagURL(baseURL, tag.GetString("name"))})
	}

	// series: same title (and place) on other upcoming dates
	sibFilter, sibParams := "title = {:t} && id != {:id}", dbx.Params{"t": title, "id": event.Id}
	if place != nil {
		sibFilter += " && place = {:p}"
		sibParams["p"] = place.Id
	}
	siblings := upcoming(sibFilter, 12, sibParams)
	skip := map[string]bool{event.Id: true}
	if len(siblings) > 0 && !start.IsZero() {
		p.SeriesLabel = shortDate(start)
	}
	for _, sib := range siblings {
		skip[sib.Id] = true
		p.OtherDates = append(p.OtherDates, eventLink(sib, shortDate(sib.GetDateTime("start_datetime").Time())))
	}

	if place != nil {
		for _, rec := range upcoming("place = {:p}", 30, dbx.Params{"p": place.Id}) {
			if len(p.MoreAtVenue) >= 5 {
				break
			}
			if !skip[rec.Id] {
				p.MoreAtVenue = append(p.MoreAtVenue, eventLink(rec, rec.GetString("title")))
			}
		}
	}

	// ended events point readers at what is on next
	if p.State != Upcoming {
		seen := map[string]bool{event.Id: true}
		add := func(recs []*core.Record) {
			for _, rec := range recs {
				if len(p.Alternatives) >= 5 {
					return
				}
				if seen[rec.Id] {
					continue
				}
				seen[rec.Id] = true
				p.Alternatives = append(p.Alternatives, eventLink(rec,
					rec.GetString("title")+" – "+shortDate(rec.GetDateTime("start_datetime").Time())))
			}
		}
		if place != nil {
			add(upcoming("place = {:p}", 10, dbx.Params{"p": place.Id}))
		}
		if tags := event.GetStringSlice("tags"); len(p.Alternatives) < 5 && len(tags) > 0 {
			add(upcoming("tags ?~ {:t}", 10, dbx.Params{"t": tags[0]}))
		}
		if len(p.Alternatives) < 5 {
			add(upcoming("", 10, nil))
		}
	}

	// featured in a weekend-picks post
	picks, _ := app.FindRecordsByFilter("picks", "hidden = false && events ?~ {:id}", "-start_date", 5, 0, dbx.Params{"id": event.Id})
	for _, pk := range picks {
		if PickLifecycle(app, pk, now) != Retired {
			p.FeaturedIn = &Link{Text: pk.GetString("title"), URL: baseURL + "/picks/" + pk.GetString("slug")}
			break
		}
	}

	p.MetaDescription = metaDescription(event.GetString("description"), start, placeName, placeCity)

	imageURL := ""
	if image := event.GetString("image"); image != "" {
		imageURL = fmt.Sprintf("%s/api/files/events/%s/%s", baseURL, event.Id, image)
	}
	p.MetaTags = buildMetaTags(p.pageTitle(), p.MetaDescription, p.URL, imageURL, instanceName)

	jsonLD, err := GenerateEventJSONLD(app, event, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JSON-LD: %w", err)
	}
	p.JSONLD = string(jsonLD)
	return p, nil
}
