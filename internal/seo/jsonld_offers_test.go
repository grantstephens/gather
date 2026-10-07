package seo

import (
	"encoding/json"
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"

	_ "gather/migrations"
)

func ldFor(t *testing.T, app core.App, cost, description string) map[string]any {
	t.Helper()
	events, _ := app.FindCollectionByNameOrId("events")
	ev := core.NewRecord(events)
	ev.Set("title", "Concert")
	ev.Set("description", description)
	ev.Set("start_datetime", time.Now().Add(48*time.Hour))
	ev.Set("status", "published")
	ev.Set("cost", cost)
	if err := app.Save(ev); err != nil {
		t.Fatal(err)
	}
	b, err := GenerateEventJSONLD(app, ev, "https://site.test")
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestJSONLDOffersOnlyWhenCostIsKnown(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()

	with := ldFor(t, app, "£5", "A fine concert that you will enjoy a great deal, with plenty of detail about it all.")
	offers, ok := with["offers"].(map[string]any)
	if !ok || offers["price"] != "5" || offers["priceCurrency"] != "GBP" {
		t.Fatalf("offers = %#v", with["offers"])
	}

	without := ldFor(t, app, "", "A fine concert that you will enjoy a great deal, with plenty of detail about it all.")
	if _, has := without["offers"]; has {
		t.Errorf("no cost must mean no offers key, got %#v", without["offers"])
	}

	unparseable := ldFor(t, app, "Pay what you can", "A fine concert that you will enjoy a great deal, with plenty of detail about it all.")
	if _, has := unparseable["offers"]; has {
		t.Errorf("unparseable cost must mean no offers key")
	}
}

func TestJSONLDDescriptionFallsBackWhenEmpty(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	ld := ldFor(t, app, "", "")
	if d, _ := ld["description"].(string); d == "" {
		t.Errorf("description must never be empty when the date is known, got %#v", ld["description"])
	}
}
