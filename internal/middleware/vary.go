package middleware

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// VaryUserAgent marks every route that serves different HTML to bots and humans
// (see seo.IsBot) as varying by User-Agent. Without it a shared cache keyed by
// URL serves the human SPA shell to Googlebot after any human visit.
func VaryUserAgent(e *core.RequestEvent) error {
	applyVary(e.Response.Header(), e.Request.Method, e.Request.URL.Path)
	return e.Next()
}

// Paths that never branch on the user agent. Matched on a path-segment boundary
// (the prefix, then end / "/" / "."), so a custom page slug that merely starts
// the same way (/feedback, /embedded-events, /apple) still gets the header.
var noVaryPrefixes = []string{
	"/api", "/_", "/assets", "/feed", "/ics", "/ap", "/federation", "/embed", "/.well-known",
}

func underPrefix(path, prefix string) bool {
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	if len(path) == len(prefix) {
		return true
	}
	c := path[len(prefix)]
	return c == '/' || c == '.'
}

var noVaryExact = map[string]bool{
	"/sitemap.xml": true, "/robots.txt": true, "/llms.txt": true,
	"/favicon.ico": true, "/ai-catalog.json": true,
}

func applyVary(h http.Header, method, path string) {
	if method != http.MethodGet && method != http.MethodHead {
		return
	}
	if noVaryExact[path] {
		return
	}
	for _, p := range noVaryPrefixes {
		if underPrefix(path, p) {
			return
		}
	}
	h.Add("Vary", "User-Agent")
}
