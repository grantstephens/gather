package newsletter

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"gather/internal/recurrence"
	"gather/internal/seo"

	"github.com/pocketbase/pocketbase/core"
)

// Campaign is everything needed to render one picks email. PlaceName and
// ThumbURL are injected so rendering stays a pure function.
type Campaign struct {
	Pick             *core.Record
	Events           []*core.Record
	SiteName         string
	BaseURL          string
	PrivacyURL       string // absolute; empty hides the link
	Footer           string
	UnsubscribeToken string
	Now              time.Time
	PlaceName        func(*core.Record) string
	ThumbURL         func(*core.Record) string
}

// NewCampaign wires the real place and image lookups.
func NewCampaign(app core.App, pick *core.Record, events []*core.Record, baseURL string, now time.Time, unsubToken string) Campaign {
	places := map[string]string{}
	return Campaign{
		Pick:             pick,
		Events:           events,
		SiteName:         siteName(app),
		BaseURL:          baseURL,
		PrivacyURL:       privacyAbsolute(app, baseURL),
		Footer:           LoadConfig(app).Footer,
		UnsubscribeToken: unsubToken,
		Now:              now,
		PlaceName: func(ev *core.Record) string {
			id := ev.GetString("place")
			if id == "" {
				return ""
			}
			if name, ok := places[id]; ok {
				return name
			}
			name := ""
			if p, err := app.FindRecordById("places", id); err == nil {
				name = p.GetString("name")
			}
			places[id] = name
			return name
		},
		ThumbURL: func(ev *core.Record) string {
			img := ev.GetString("image")
			if img == "" {
				return ""
			}
			return fmt.Sprintf("%s/api/files/%s/%s/%s?thumb=400x300", baseOf(baseURL), ev.Collection().Id, ev.Id, img)
		},
	}
}

// addUTM tags a link to this Gather instance with the newsletter campaign.
// Links to any other host (compared by scheme and host, never by prefix) and
// anything unparsable are returned untouched.
func addUTM(rawURL, baseURL, campaign string) string {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return rawURL
	}
	b, err := url.Parse(baseURL)
	if err != nil || u.Scheme != b.Scheme || !strings.EqualFold(u.Host, b.Host) {
		return rawURL
	}
	q := u.Query()
	q.Set("utm_source", "newsletter")
	q.Set("utm_medium", "email")
	q.Set("utm_campaign", campaign)
	u.RawQuery = q.Encode()
	return u.String()
}

// oneLine collapses control characters (incl. CR/LF, which would allow header
// injection) and runs of whitespace into single spaces.
func oneLine(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

func eventLink(c Campaign, ev *core.Record, campaign string) string {
	ref := ev.GetString("slug")
	if ref == "" {
		ref = ev.Id
	}
	return addUTM(baseOf(c.BaseURL)+"/event/"+ref, c.BaseURL, campaign)
}

// eventWhen shows the next occurrence for recurring events, not the series' first date.
func eventWhen(ev *core.Record, now time.Time) string {
	start := ev.GetDateTime("start_datetime").Time()
	if rule := ev.GetString("recurrence_rule"); rule != "" && !start.IsZero() {
		if times, err := recurrence.Expand(rule, start, now, now.AddDate(0, 0, 90)); err == nil {
			times = recurrence.FilterExceptions(times, ev.GetString("recurrence_exceptions"))
			if len(times) > 0 {
				return seo.LongDateTime(times[0])
			}
		}
	}
	if start.IsZero() {
		return ""
	}
	return seo.LongDateTime(start)
}

// RenderCampaign builds the subject, HTML and plain-text bodies and the
// bulk-mail headers (RFC 8058 one-click unsubscribe).
func RenderCampaign(c Campaign) (subject, htmlBody, textBody string, headers map[string]string) {
	slug := c.Pick.GetString("slug")
	title := oneLine(c.Pick.GetString("title"))
	if title == "" {
		title = "What's on this weekend"
	}
	subject = title
	site := oneLine(c.SiteName)
	if site == "" {
		site = "Gather"
	}
	base := baseOf(c.BaseURL)
	rewrite := func(dest string) string { return addUTM(dest, c.BaseURL, slug) }
	blurbHTML := styleBlurb(seo.RenderMarkdownHTMLWith(c.Pick.GetString("blurb"), c.BaseURL, rewrite))
	webURL := addUTM(base+"/picks/"+slug, c.BaseURL, slug)
	unsubPage := UnsubscribePageURL(c.BaseURL, c.UnsubscribeToken)

	cardRows := map[string]string{} // event id -> <tr> card
	var textEvents strings.Builder
	for _, ev := range c.Events {
		link := eventLink(c, ev, slug)
		when := eventWhen(ev, c.Now)
		place := ""
		if c.PlaceName != nil {
			place = oneLine(c.PlaceName(ev))
		}
		thumb := ""
		if c.ThumbURL != nil {
			thumb = c.ThumbURL(ev)
		}
		evTitle := oneLine(ev.GetString("title"))

		var cards strings.Builder
		cards.WriteString(`<tr><td style="padding:8px 0 16px;"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border:1px solid #e2e8f0;border-radius:12px;border-collapse:separate;overflow:hidden;background:#ffffff;"><tr>`)
		if thumb != "" {
			fmt.Fprintf(&cards, `<td class="ec-img" width="170" valign="top" style="width:170px;background:#f1f5f9;"><a href="%s"><img src="%s" width="170" alt="" style="display:block;width:170px;max-width:100%%;height:auto;border:0;"></a></td>`, html.EscapeString(link), html.EscapeString(thumb))
		}
		cards.WriteString(`<td class="ec-body" valign="middle" style="padding:16px 20px;">`)
		if when != "" {
			fmt.Fprintf(&cards, `<div style="font-size:12px;font-weight:600;color:#0d9488;text-transform:uppercase;letter-spacing:0.05em;margin-bottom:4px;">%s</div>`, html.EscapeString(when))
		}
		fmt.Fprintf(&cards, `<a href="%s" %s">%s</a>`, html.EscapeString(link), cardTitleStyle, html.EscapeString(evTitle))
		if place != "" {
			fmt.Fprintf(&cards, `<div style="font-size:12px;color:#5c6b7f;margin-top:4px;">%s</div>`, html.EscapeString(place))
		}
		cards.WriteString(`</td></tr></table></td></tr>`)
		cardRows[ev.Id] = cards.String()

		fmt.Fprintf(&textEvents, "- %s", evTitle)
		if when != "" {
			fmt.Fprintf(&textEvents, "\n  %s", when)
		}
		if place != "" {
			fmt.Fprintf(&textEvents, "\n  %s", place)
		}
		fmt.Fprintf(&textEvents, "\n  %s\n\n", link)
	}

	blurbHTML, leftover := weaveCards(blurbHTML, c, cardRows)
	var cards strings.Builder
	for _, ev := range c.Events {
		if row, ok := leftover[ev.Id]; ok {
			cards.WriteString(row)
		}
	}

	footerHTML := fmt.Sprintf(`You are receiving this because you subscribed to What&#39;s on this weekend from %s. <a href="%s" style="color:#64748b;">Unsubscribe</a>`, html.EscapeString(site), html.EscapeString(unsubPage))
	if c.PrivacyURL != "" {
		footerHTML += fmt.Sprintf(` &middot; <a href="%s" style="color:#64748b;">Privacy policy</a>`, html.EscapeString(c.PrivacyURL))
	}
	if f := strings.TrimSpace(c.Footer); f != "" {
		footerHTML += `<br>` + strings.ReplaceAll(html.EscapeString(f), "\n", "<br>")
	}

	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"><style>@media only screen and (max-width:480px){.ec-img,.ec-body{display:block!important;width:100%%!important;box-sizing:border-box}.ec-img img{width:100%%!important}}</style></head>
<body style="margin:0;padding:0;background:#f1f5f9;font-family:Inter,system-ui,-apple-system,sans-serif;color:#0f172a;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f1f5f9;padding:24px 12px;"><tr><td align="center">
<table width="100%%" cellpadding="0" cellspacing="0" style="max-width:600px;">
<tr><td style="background:#ffffff;border:1px solid #e2e8f0;border-bottom:1px solid #e2e8f0;border-radius:12px 12px 0 0;padding:20px 28px;"><a href="%s" style="font-size:22px;font-weight:800;letter-spacing:-0.04em;line-height:1;color:#0f172a;text-decoration:none;">%s</a></td></tr>
<tr><td style="background:#ffffff;padding:28px;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;">
<h1 style="margin:0 0 16px;font-size:28px;font-weight:700;letter-spacing:-0.025em;line-height:1.25;color:#0f172a;">%s</h1>
<div style="font-size:16px;line-height:1.7;color:#475569;">%s</div>
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="margin-top:8px;">%s</table>
<p style="margin:24px 0 0;font-size:14px;"><a href="%s" style="color:#0d9488;text-decoration:underline;">View on the web</a></p>
</td></tr>
<tr><td style="background:#f8fafc;border:1px solid #e2e8f0;border-top:none;border-radius:0 0 12px 12px;padding:16px 28px;font-size:12px;line-height:1.6;color:#64748b;">%s</td></tr>
</table></td></tr></table>
<img src="%s/newsletter/open/%s.gif" width="1" height="1" alt="" style="display:block;border:0;">
</body></html>`,
		html.EscapeString(rewrite(base+"/")),
		wordmarkHTML(site),
		html.EscapeString(title),
		blurbHTML,
		cards.String(),
		html.EscapeString(webURL),
		footerHTML,
		html.EscapeString(base), html.EscapeString(c.Pick.Id),
	)

	var tb strings.Builder
	fmt.Fprintf(&tb, "%s\n%s\n\n", site, title)
	if blurb := strings.TrimSpace(strings.ReplaceAll(c.Pick.GetString("blurb"), "\r", "")); blurb != "" {
		tb.WriteString(blurb + "\n\n")
	}
	tb.WriteString(textEvents.String())
	fmt.Fprintf(&tb, "View on the web: %s\n\n--\nYou are receiving this because you subscribed to What's on this weekend from %s.\nUnsubscribe: %s\n", webURL, site, unsubPage)
	if c.PrivacyURL != "" {
		fmt.Fprintf(&tb, "Privacy policy: %s\n", c.PrivacyURL)
	}
	if f := strings.TrimSpace(c.Footer); f != "" {
		tb.WriteString(strings.ReplaceAll(f, "\r", "") + "\n")
	}
	textBody = tb.String()

	headers = map[string]string{
		"List-Unsubscribe":      "<" + UnsubscribePostURL(c.BaseURL, c.UnsubscribeToken) + ">",
		"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
		"Precedence":            "bulk",
	}
	return subject, htmlBody, textBody, headers
}

var (
	blockEndRe = regexp.MustCompile(`(?is)</(?:p|h[1-6])>`)
	hrefAttrRe = regexp.MustCompile(`(?i)href="([^"]+)"`)
)

// weaveCards drops each event's card straight after the first paragraph or
// heading of the blurb that links to that event, so the email reads as one
// piece. It returns the new blurb HTML and the cards that were not placed
// (events the blurb never links to), keyed by event id.
func weaveCards(blurbHTML string, c Campaign, cardRows map[string]string) (string, map[string]string) {
	leftover := make(map[string]string, len(cardRows))
	for id, row := range cardRows {
		leftover[id] = row
	}
	byRef := map[string]*core.Record{}
	for _, ev := range c.Events {
		byRef[ev.Id] = ev
		if slug := ev.GetString("slug"); slug != "" {
			byRef[slug] = ev
		}
	}
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return blurbHTML, leftover
	}

	var out strings.Builder
	prev := 0
	for _, loc := range blockEndRe.FindAllStringIndex(blurbHTML, -1) {
		block := blurbHTML[prev:loc[1]]
		out.WriteString(block)
		prev = loc[1]
		for _, m := range hrefAttrRe.FindAllStringSubmatch(block, -1) {
			u, err := url.Parse(html.UnescapeString(m[1]))
			if err != nil || !strings.EqualFold(u.Host, base.Host) || !strings.HasPrefix(u.Path, "/event/") {
				continue
			}
			ev := byRef[strings.TrimPrefix(u.Path, "/event/")]
			if ev == nil {
				continue
			}
			if row, ok := leftover[ev.Id]; ok {
				fmt.Fprintf(&out, `<table width="100%%" cellpadding="0" cellspacing="0" style="margin:0;">%s</table>`, row)
				delete(leftover, ev.Id)
			}
		}
	}
	out.WriteString(blurbHTML[prev:])
	return out.String(), leftover
}

// cardTitleStyle is the inline style for an event card's title link.
const cardTitleStyle = `style="font-size:20px;font-weight:700;letter-spacing:-0.02em;line-height:1.35;color:#0f172a;text-decoration:none;`

var (
	blurbTagRe = regexp.MustCompile(`(?i)<(p|h[1-6]|a|strong|ul|ol|li|blockquote)(\s[^>]*)?>`)
)

// styleBlurb inlines the site's typography into the blurb's HTML (email
// clients ignore our stylesheet): larger lead paragraph, bold headings with
// room above, accent-coloured underlined links.
func styleBlurb(h string) string {
	firstP := true
	return blurbTagRe.ReplaceAllStringFunc(h, func(tag string) string {
		m := blurbTagRe.FindStringSubmatch(tag)
		name, attrs := strings.ToLower(m[1]), m[2]
		var st string
		switch name {
		case "p":
			st = "margin:0 0 16px;"
			if firstP {
				st += "font-size:18px;"
				firstP = false
			}
		case "h1", "h2", "h3", "h4", "h5", "h6":
			st = "margin:32px 0 8px;font-size:20px;font-weight:700;line-height:1.3;color:#0f172a;"
		case "a":
			st = "color:#0d9488;text-decoration:underline;"
		case "strong":
			st = "color:#0f172a;font-weight:600;"
		case "ul", "ol":
			st = "margin:0 0 16px;padding-left:24px;"
		case "li":
			st = "margin:0 0 4px;"
		case "blockquote":
			st = "margin:0 0 16px;padding-left:16px;border-left:3px solid #e2e8f0;"
		}
		return "<" + name + attrs + ` style="` + st + `">`
	})
}

// wordmarkHTML renders the site name like the site header does: a teal dot
// after the first word ("Perthshire.Events").
func wordmarkHTML(name string) string {
	dot := `<span style="color:#0d9488;">.</span>`
	if i := strings.Index(name, " "); i > 0 {
		return html.EscapeString(name[:i]) + dot + html.EscapeString(name[i:])
	}
	return html.EscapeString(name) + dot
}
