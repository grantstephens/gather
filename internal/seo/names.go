package seo

import (
	"net/url"
	"strings"
	"unicode"
)

// regionFromInstance: "Perthshire Events" -> "Perthshire"; anything else unchanged.
func regionFromInstance(name string) string {
	n := strings.TrimSpace(name)
	if strings.HasSuffix(n, " Events") {
		return strings.TrimSpace(strings.TrimSuffix(n, " Events"))
	}
	return n
}

// humanizeTag: "strathearn-arts" -> "Strathearn Arts".
func humanizeTag(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' || unicode.IsSpace(r) })
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// tagURL is a tag page's URL. The name is a path segment, so it must be
// escaped: "C#" would otherwise lose its fragment and "a/b" gain a segment.
func tagURL(baseURL, name string) string {
	return baseURL + "/tag/" + url.PathEscape(name)
}
