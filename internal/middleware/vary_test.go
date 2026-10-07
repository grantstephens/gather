package middleware

import (
	"net/http"
	"testing"
)

func TestApplyVary(t *testing.T) {
	cases := []struct {
		method, path string
		want         bool
	}{
		{"GET", "/", true},
		{"GET", "/event/film-saipan-15-qwo49pf3", true},
		{"HEAD", "/tag/theatre", true},
		{"GET", "/place/abc", true},
		{"GET", "/picks", true},
		{"GET", "/about", true},
		{"POST", "/", false},
		{"GET", "/api/collections/events/records", false},
		{"GET", "/_/", false},
		{"GET", "/assets/index-abc.js", false},
		{"GET", "/feed.rss", false},
		{"GET", "/feed.ics", false},
		{"GET", "/sitemap.xml", false},
		{"GET", "/robots.txt", false},
		{"GET", "/favicon.ico", false},
		{"GET", "/.well-known/ard.json", false},
	}
	for _, c := range cases {
		h := http.Header{}
		applyVary(h, c.method, c.path)
		got := h.Get("Vary") == "User-Agent"
		if got != c.want {
			t.Errorf("%s %s: Vary=%q, want applied=%v", c.method, c.path, h.Get("Vary"), c.want)
		}
	}
}

func TestApplyVaryKeepsExistingValues(t *testing.T) {
	h := http.Header{}
	h.Add("Vary", "Accept-Encoding")
	applyVary(h, "GET", "/")
	if got := h.Values("Vary"); len(got) != 2 {
		t.Fatalf("want both Vary values kept, got %v", got)
	}
}
