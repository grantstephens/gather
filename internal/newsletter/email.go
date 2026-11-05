package newsletter

import (
	"fmt"
	"html"
	"net/url"
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
	blurbHTML := seo.RenderMarkdownHTMLWith(c.Pick.GetString("blurb"), c.BaseURL, rewrite)
	webURL := addUTM(base+"/picks/"+slug, c.BaseURL, slug)
	unsubPage := UnsubscribePageURL(c.BaseURL, c.UnsubscribeToken)

	var cards, textEvents strings.Builder
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

		cards.WriteString(`<tr><td style="padding:12px 0;border-top:1px solid #e2e8f0;"><table width="100%" cellpadding="0" cellspacing="0"><tr>`)
		if thumb != "" {
			fmt.Fprintf(&cards, `<td width="120" valign="top" style="padding-right:12px;"><a href="%s"><img src="%s" width="120" alt="" style="display:block;border-radius:6px;max-width:120px;height:auto;"></a></td>`, html.EscapeString(link), html.EscapeString(thumb))
		}
		fmt.Fprintf(&cards, `<td valign="top"><a href="%s" style="font-size:16px;font-weight:600;color:#0f766e;text-decoration:none;">%s</a>`, html.EscapeString(link), html.EscapeString(evTitle))
		if when != "" {
			fmt.Fprintf(&cards, `<div style="font-size:14px;color:#334155;margin-top:2px;">%s</div>`, html.EscapeString(when))
		}
		if place != "" {
			fmt.Fprintf(&cards, `<div style="font-size:14px;color:#64748b;">%s</div>`, html.EscapeString(place))
		}
		cards.WriteString(`</td></tr></table></td></tr>`)

		fmt.Fprintf(&textEvents, "- %s", evTitle)
		if when != "" {
			fmt.Fprintf(&textEvents, "\n  %s", when)
		}
		if place != "" {
			fmt.Fprintf(&textEvents, "\n  %s", place)
		}
		fmt.Fprintf(&textEvents, "\n  %s\n\n", link)
	}

	footerHTML := fmt.Sprintf(`You are receiving this because you subscribed to What&#39;s on this weekend from %s. <a href="%s" style="color:#64748b;">Unsubscribe</a>`, html.EscapeString(site), html.EscapeString(unsubPage))
	if c.PrivacyURL != "" {
		footerHTML += fmt.Sprintf(` &middot; <a href="%s" style="color:#64748b;">Privacy policy</a>`, html.EscapeString(c.PrivacyURL))
	}
	if f := strings.TrimSpace(c.Footer); f != "" {
		footerHTML += `<br>` + strings.ReplaceAll(html.EscapeString(f), "\n", "<br>")
	}

	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light dark"></head>
<body style="margin:0;padding:0;background:#f1f5f9;font-family:Inter,system-ui,-apple-system,sans-serif;color:#0f172a;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f1f5f9;padding:24px 12px;"><tr><td align="center">
<table width="100%%" cellpadding="0" cellspacing="0" style="max-width:600px;">
<tr><td style="background:#0d9488;border-radius:8px 8px 0 0;padding:20px 28px;"><span style="font-size:20px;font-weight:700;color:#ffffff;">%s</span></td></tr>
<tr><td style="background:#ffffff;padding:28px;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;">
<h1 style="margin:0 0 16px;font-size:22px;line-height:1.3;">%s</h1>
<div style="font-size:15px;line-height:1.6;">%s</div>
<table width="100%%" cellpadding="0" cellspacing="0" style="margin-top:20px;">%s</table>
<p style="margin:24px 0 0;font-size:14px;"><a href="%s" style="color:#0f766e;">View on the web</a></p>
</td></tr>
<tr><td style="background:#f8fafc;border:1px solid #e2e8f0;border-top:none;border-radius:0 0 8px 8px;padding:16px 28px;font-size:12px;line-height:1.6;color:#64748b;">%s</td></tr>
</table></td></tr></table>
<img src="%s/newsletter/open/%s.gif" width="1" height="1" alt="" style="display:block;border:0;">
</body></html>`,
		html.EscapeString(site),
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
