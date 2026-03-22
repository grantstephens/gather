package seo

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var srcAttrPattern = regexp.MustCompile(`(?i)src\s*=\s*["'](https?://[^"']+)["']`)

// ExtractExternalOrigins returns the unique HTTPS origins found in src="..."
// attributes within the given HTML snippet (e.g. custom_head).
func ExtractExternalOrigins(html string) []string {
	seen := make(map[string]bool)
	var origins []string
	for _, m := range srcAttrPattern.FindAllStringSubmatch(html, -1) {
		u, err := url.Parse(m[1])
		if err != nil || u.Host == "" {
			continue
		}
		origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		if !seen[origin] {
			seen[origin] = true
			origins = append(origins, origin)
		}
	}
	return origins
}

// BuildCSP returns the full Content-Security-Policy header value.
// extraOrigins (derived from custom_head) are added to script-src and connect-src.
func BuildCSP(extraOrigins []string) string {
	scriptSrc := "'self' 'sha256-tBGiZqQH9MMf5f20bIZnnQz1p4lvVBHKyBmzgeAIBcE=' 'unsafe-hashes' 'sha256-MhtPZXr7+LpJUY5qtMutB+qWfQtMaPccfe7QXtCcEYc='"
	connectSrc := "'self'"
	if len(extraOrigins) > 0 {
		extra := strings.Join(extraOrigins, " ")
		scriptSrc += " " + extra
		connectSrc += " " + extra
	}
	return fmt.Sprintf(
		"default-src 'self'; "+
			"img-src 'self' https://*.tile.openstreetmap.org data: blob:; "+
			"style-src 'self' 'unsafe-inline'; "+
			"font-src 'self'; "+
			"script-src %s; "+
			"connect-src %s; "+
			"frame-ancestors 'none'; "+
			"object-src 'none'; "+
			"base-uri 'self'",
		scriptSrc, connectSrc,
	)
}
