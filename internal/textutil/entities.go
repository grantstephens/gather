// Package textutil holds small text helpers shared by hooks, migrations and renderers.
package textutil

import (
	"regexp"
	"strconv"
	"strings"
)

var numericEntity = regexp.MustCompile(`&#(?:[xX]([0-9a-fA-F]{1,6})|([0-9]{1,7}));`)

var namedEntities = strings.NewReplacer("&amp;", "&", "&quot;", `"`, "&apos;", "'", "&nbsp;", " ")

// DecodeEntities undoes HTML-escaping that scrapers applied to plain text,
// including double-encoding ("&amp;#039;"), up to three passes. It decodes
// numeric entities, &amp; &quot; &apos; &nbsp;, and deliberately never
// produces '<' or '>' so decoding can never introduce markup.
func DecodeEntities(s string) string {
	for i := 0; i < 3 && strings.Contains(s, "&"); i++ {
		next := decodeOnce(s)
		if next == s {
			break
		}
		s = next
	}
	return s
}

func decodeOnce(s string) string {
	s = numericEntity.ReplaceAllStringFunc(s, func(m string) string {
		sub := numericEntity.FindStringSubmatch(m)
		var n int64
		var err error
		if sub[1] != "" {
			n, err = strconv.ParseInt(sub[1], 16, 32)
		} else {
			n, err = strconv.ParseInt(sub[2], 10, 32)
		}
		r := rune(n)
		if err != nil || r == 0 || r > 0x10FFFF || (r >= 0xD800 && r <= 0xDFFF) || r == '<' || r == '>' {
			return m
		}
		return string(r)
	})
	return namedEntities.Replace(s)
}
