package newsletter

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"gather/internal/api"
	"gather/internal/recurrence"
	"gather/internal/seo"

	"github.com/pocketbase/pocketbase/core"
)

func TestAddUTM(t *testing.T) {
	const base = "https://gather.test"
	got := addUTM("https://gather.test/event/a", base, "weekend")
	for _, want := range []string{"utm_source=newsletter", "utm_medium=email", "utm_campaign=weekend"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
	if got := addUTM("https://gather.test/event/a?a=1", base, "weekend"); !strings.Contains(got, "a=1") || !strings.Contains(got, "utm_campaign=weekend") {
		t.Errorf("existing query must be preserved: %s", got)
	}
	for _, untouched := range []string{
		"https://gather.test.evil.com/x",
		"https://gather.testevil.com/x",
		"https://other.example/x",
		"/relative",
		"mailto:a@b.c",
		"javascript:alert(1)",
		"::not a url",
	} {
		if got := addUTM(untouched, base, "weekend"); got != untouched {
			t.Errorf("addUTM(%q) = %q, want unchanged", untouched, got)
		}
	}
}

// campaignFixture builds a realistic campaign with two events.
func campaignFixture(t *testing.T, app core.App, title, blurb string) Campaign {
	t.Helper()
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	e1 := mkEvent(t, app, now.AddDate(0, 0, 2), time.Time{}, "published", "")
	e2 := mkEvent(t, app, now.AddDate(0, 0, 3), time.Time{}, "published", "")
	pick := mkPick(t, app, pickSpec{slug: "weekend-picks", start: "2026-11-06", events: []string{e1, e2}})
	pick.Set("title", title)
	pick.Set("blurb", blurb)
	if err := app.SaveNoValidate(pick); err != nil {
		t.Fatal(err)
	}
	var evs []*core.Record
	for _, id := range []string{e1, e2} {
		ev, _ := app.FindRecordById("events", id)
		evs = append(evs, ev)
	}
	return Campaign{
		Pick:             pick,
		Events:           evs,
		SiteName:         "Test Gather",
		BaseURL:          testBase,
		UnsubscribeToken: "unsub123",
		Now:              now,
		PlaceName: func(ev *core.Record) string {
			if ev.Id == e1 {
				return "The Hall"
			}
			return ""
		},
		ThumbURL: func(ev *core.Record) string {
			if ev.Id == e1 {
				return testBase + "/api/files/events/" + ev.Id + "/pic.jpg?thumb=400x300"
			}
			return ""
		},
	}
}

var hrefRe = regexp.MustCompile(`href="([^"]+)"`)

func TestRenderCampaignLinksAndFooter(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	c := campaignFixture(t, app, "Weekend Picks: 6-8 Nov", "Intro with [a link](/event/abc) here.\n\nSecond paragraph.")
	c.Footer = "Gather Ltd, 1 High Street"

	subject, htmlBody, textBody, headers := RenderCampaign(c)

	if subject != "Weekend Picks: 6-8 Nov" {
		t.Errorf("subject = %q", subject)
	}
	links := 0
	for _, m := range hrefRe.FindAllStringSubmatch(htmlBody, -1) {
		href := strings.ReplaceAll(m[1], "&amp;", "&")
		if !strings.HasPrefix(href, testBase) {
			continue
		}
		if strings.Contains(href, "/newsletter/unsubscribe") || strings.Contains(href, "/privacy") {
			continue
		}
		links++
		if !strings.Contains(href, "utm_campaign=weekend-picks") || !strings.Contains(href, "utm_source=newsletter") {
			t.Errorf("Gather link without UTM: %s", href)
		}
	}
	if links < 4 { // blurb link + 2 event cards + view-on-web
		t.Errorf("expected at least 4 tracked links, found %d", links)
	}
	if !strings.Contains(htmlBody, "pic.jpg?thumb=400x300") {
		t.Error("event thumbnail missing")
	}
	if !strings.Contains(htmlBody, "The Hall") {
		t.Error("place name missing")
	}
	if !strings.Contains(htmlBody, "Gather Ltd, 1 High Street") {
		t.Error("operator footer text missing")
	}
	if !strings.Contains(htmlBody, UnsubscribePageURL(testBase, "unsub123")) {
		t.Error("unsubscribe page link missing from html")
	}
	if !strings.Contains(textBody, UnsubscribePageURL(testBase, "unsub123")) {
		t.Error("unsubscribe page link missing from text")
	}
	if want := "/newsletter/open/" + c.Pick.Id + ".gif"; !strings.Contains(htmlBody, want) {
		t.Errorf("open pixel %s missing", want)
	}
	if want := "<" + UnsubscribePostURL(testBase, "unsub123") + ">"; headers["List-Unsubscribe"] != want {
		t.Errorf("List-Unsubscribe = %q, want %q", headers["List-Unsubscribe"], want)
	}
	if headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" {
		t.Errorf("List-Unsubscribe-Post = %q", headers["List-Unsubscribe-Post"])
	}
	if headers["Precedence"] != "bulk" {
		t.Errorf("Precedence = %q", headers["Precedence"])
	}

	// privacy link only when a privacy URL is supplied
	if strings.Contains(htmlBody, "/privacy") {
		t.Error("no privacy link expected without PrivacyURL")
	}
	c.PrivacyURL = testBase + "/privacy"
	_, withPrivacy, _, _ := RenderCampaign(c)
	if !strings.Contains(withPrivacy, testBase+"/privacy") {
		t.Error("privacy link expected when PrivacyURL is set")
	}
}

func TestRenderCampaignSanitisesBlurb(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	c := campaignFixture(t, app, "Picks", "<script>alert(1)</script> [x](javascript:alert(1)) <img src=x onerror=alert(2)>")
	_, htmlBody, _, _ := RenderCampaign(c)
	for _, bad := range []string{"<script", "javascript:", "onerror"} {
		if strings.Contains(htmlBody, bad) {
			t.Errorf("html must not contain %q", bad)
		}
	}
}

func TestRenderCampaignEscapesAndStripsHeaderInjection(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	c := campaignFixture(t, app, "A\r\nBcc: evil@x.com <b>", "x")
	c.SiteName = "<i>Site</i>"
	subject, htmlBody, textBody, _ := RenderCampaign(c)

	if strings.ContainsAny(subject, "\r\n") {
		t.Errorf("subject must not contain CR/LF: %q", subject)
	}
	if strings.Contains(htmlBody, "evil@x.com <b>") || !strings.Contains(htmlBody, "&lt;b&gt;") {
		t.Error("title must be HTML-escaped")
	}
	if strings.Contains(htmlBody, "<i>Site</i>") {
		t.Error("site name must be HTML-escaped")
	}
	if strings.Contains(textBody, "\r") {
		t.Error("text body must not contain CR from titles")
	}
}

func TestRenderCampaignRecurringShowsNextOccurrence(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	c := campaignFixture(t, app, "Picks", "x")
	start := c.Now.AddDate(0, 0, -70)
	id := mkEvent(t, app, start, time.Time{}, "published", "FREQ=WEEKLY;COUNT=500")
	ev, _ := app.FindRecordById("events", id)
	c.Events = []*core.Record{ev}

	next, err := recurrence.Expand("FREQ=WEEKLY;COUNT=500", start, c.Now, c.Now.AddDate(0, 0, 14))
	if err != nil || len(next) == 0 {
		t.Fatal("test setup: no next occurrence", err)
	}
	_, htmlBody, _, _ := RenderCampaign(c)
	if !strings.Contains(htmlBody, seo.LongDateTime(next[0])) {
		t.Errorf("expected next occurrence %q in email", seo.LongDateTime(next[0]))
	}
	if strings.Contains(htmlBody, seo.LongDateTime(start)) {
		t.Error("must not show the series' first date")
	}
}

func TestRenderCampaignSize(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	c := campaignFixture(t, app, "Picks", strings.Repeat("A fairly long sentence about local events. ", 200))
	base := c.Events[0]
	c.Events = nil
	for i := 0; i < 30; i++ {
		c.Events = append(c.Events, base)
	}
	_, htmlBody, _, _ := RenderCampaign(c)
	if len(htmlBody) > 100*1024 {
		t.Errorf("html is %d bytes; Gmail clips at ~102KB", len(htmlBody))
	}
}
