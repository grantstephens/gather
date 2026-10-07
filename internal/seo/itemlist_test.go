package seo

import (
	"encoding/json"
	"testing"
)

func TestItemListJSONLD(t *testing.T) {
	out := itemListJSONLD([]Link{{Text: "A", URL: "https://x/a"}, {Text: "B & C", URL: "https://x/b"}})
	var got struct {
		Context  string `json:"@context"`
		Type     string `json:"@type"`
		Count    int    `json:"numberOfItems"`
		Elements []struct {
			Type     string `json:"@type"`
			Position int    `json:"position"`
			URL      string `json:"url"`
			Name     string `json:"name"`
		} `json:"itemListElement"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	if got.Type != "ItemList" || got.Context != "https://schema.org" || got.Count != 2 || len(got.Elements) != 2 {
		t.Fatalf("unexpected list: %+v", got)
	}
	if e := got.Elements[1]; e.Type != "ListItem" || e.Position != 2 || e.URL != "https://x/b" || e.Name != "B & C" {
		t.Errorf("element 2 = %+v", e)
	}
}

func TestItemListJSONLDEmpty(t *testing.T) {
	if got := itemListJSONLD(nil); got != "" {
		t.Errorf("empty list should produce nothing, got %q", got)
	}
}
