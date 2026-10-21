package seo

import (
	"bytes"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var blurbPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("h2", "h3", "h4", "p", "strong", "em", "ul", "ol", "li", "br")
	p.AllowAttrs("href").OnElements("a")
	p.AllowURLSchemes("http", "https", "mailto")
	p.AllowRelativeURLs(false)
	p.RequireNoFollowOnLinks(false)
	return p
}()

// renderMarkdownHTML renders Markdown to sanitised HTML for crawlers. Site-relative
// links ("/event/x") are made absolute with baseURL; protocol-relative, javascript:
// and any other non-http(s)/mailto links lose their href. Raw HTML is dropped.
func renderMarkdownHTML(md, baseURL string) string {
	if strings.TrimSpace(md) == "" {
		return ""
	}
	gm := goldmark.New()
	src := []byte(md)
	doc := gm.Parser().Parse(text.NewReader(src))
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if l, ok := n.(*ast.Link); ok && entering {
			dest := string(l.Destination)
			if strings.HasPrefix(dest, "/") && !strings.HasPrefix(dest, "//") {
				l.Destination = []byte(strings.TrimRight(baseURL, "/") + dest)
			}
		}
		return ast.WalkContinue, nil
	})
	var buf bytes.Buffer
	if err := gm.Renderer().Render(&buf, src, doc); err != nil {
		return ""
	}
	return strings.TrimSpace(blurbPolicy.Sanitize(buf.String()))
}
