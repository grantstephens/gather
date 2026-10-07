package seo

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// AllRecords pages through FindRecordsByFilter so callers are never silently
// capped (the old sitemap stopped at 500 events).
func AllRecords(app core.App, collection, filter, sort string, params ...dbx.Params) ([]*core.Record, error) {
	const page = 200
	var out []*core.Record
	for offset := 0; ; offset += page {
		recs, err := app.FindRecordsByFilter(collection, filter, sort, page, offset, params...)
		if err != nil {
			return nil, err
		}
		out = append(out, recs...)
		if len(recs) < page {
			return out, nil
		}
	}
}

func upcomingIDs(app core.App, now time.Time) (places, tags map[string]bool) {
	places, tags = map[string]bool{}, map[string]bool{}
	evs, _ := AllRecords(app, "events",
		"status = 'published' && start_datetime >= {:now}", "",
		dbx.Params{"now": now.UTC().Format("2006-01-02 15:04:05")})
	for _, ev := range evs {
		if p := ev.GetString("place"); p != "" {
			places[p] = true
		}
		for _, t := range ev.GetStringSlice("tags") {
			tags[t] = true
		}
	}
	return places, tags
}

// ActivePlaces are approved places with at least one upcoming published event —
// exactly the places whose page exists (others 404).
func ActivePlaces(app core.App, now time.Time) []*core.Record {
	active, _ := upcomingIDs(app, now)
	places, _ := AllRecords(app, "places", "status = 'approved'", "name")
	var out []*core.Record
	for _, p := range places {
		if active[p.Id] {
			out = append(out, p)
		}
	}
	return out
}

// ActiveTags are approved tags with at least one upcoming published event.
func ActiveTags(app core.App, now time.Time) []*core.Record {
	_, active := upcomingIDs(app, now)
	tags, _ := AllRecords(app, "tags", "status = 'approved'", "name")
	var out []*core.Record
	for _, t := range tags {
		if active[t.Id] {
			out = append(out, t)
		}
	}
	return out
}

// PlaceRef is what appears in a place URL: the slug when set, else the record id.
func PlaceRef(place *core.Record) string {
	if s := place.GetString("slug"); s != "" {
		return s
	}
	return place.Id
}
