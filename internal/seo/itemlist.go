package seo

import "encoding/json"

// itemListJSONLD renders a schema.org ItemList for a page that lists events,
// or "" when there is nothing to list.
func itemListJSONLD(items []Link) string {
	if len(items) == 0 {
		return ""
	}
	type element struct {
		Type     string `json:"@type"`
		Position int    `json:"position"`
		URL      string `json:"url"`
		Name     string `json:"name"`
	}
	type list struct {
		Context  string    `json:"@context"`
		Type     string    `json:"@type"`
		Count    int       `json:"numberOfItems"`
		Elements []element `json:"itemListElement"`
	}
	l := list{Context: "https://schema.org", Type: "ItemList", Count: len(items)}
	for i, it := range items {
		l.Elements = append(l.Elements, element{Type: "ListItem", Position: i + 1, URL: it.URL, Name: it.Text})
	}
	b, err := json.Marshal(l)
	if err != nil {
		return ""
	}
	return string(b)
}
