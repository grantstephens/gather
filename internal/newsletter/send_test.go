package newsletter

import (
	"errors"
	"strings"
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type tickFixture struct {
	app    *tests.TestApp
	now    time.Time
	loc    *time.Location
	pickID string
}

// newTickFixture: a Wednesday 08:00 schedule, one eligible pick, two
// confirmed subscribers and one pending one.
func newTickFixture(t *testing.T) *tickFixture {
	t.Helper()
	app := api.NewTestApp(t)
	loc := time.FixedZone("SAST", 2*3600)
	now := time.Date(2026, 11, 4, 8, 0, 0, 0, loc) // Wednesday
	setNewsletterSettings(t, app, true, map[string]any{
		"newsletter_send_day":  3,
		"newsletter_send_time": "08:00",
	})
	setMail(app, true, "news@gather.test")
	ev := mkEvent(t, app, now.AddDate(0, 0, 2), time.Time{}, "published", "")
	pick := mkPick(t, app, pickSpec{slug: "weekend-picks", start: "2026-11-06", events: []string{ev}})
	seedSubscriber(t, app, "one@example.com", "confirmed", time.Now())
	seedSubscriber(t, app, "two@example.com", "confirmed", time.Now())
	seedSubscriber(t, app, "pending@example.com", "pending", time.Now())
	return &tickFixture{app: app, now: now, loc: loc, pickID: pick.Id}
}

func (f *tickFixture) pick(t *testing.T) *core.Record {
	t.Helper()
	p, err := f.app.FindRecordById("picks", f.pickID)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestTickSendsOnceAtScheduledMinute(t *testing.T) {
	f := newTickFixture(t)
	defer f.app.Cleanup()

	sent, err := Tick(f.app, f.now, f.loc, testBase)
	if err != nil || sent != 2 {
		t.Fatalf("Tick = %d, %v; want 2, nil", sent, err)
	}
	if n := f.app.TestMailer.TotalSend(); n != 2 {
		t.Errorf("mails sent = %d, want 2 (pending subscriber must not get one)", n)
	}
	p := f.pick(t)
	if p.GetDateTime("newsletter_sent_at").IsZero() {
		t.Error("pick must be stamped as sent")
	}
	if p.GetInt("newsletter_sent_count") != 2 {
		t.Errorf("newsletter_sent_count = %d, want 2", p.GetInt("newsletter_sent_count"))
	}
	msg := f.app.TestMailer.LastMessage()
	if msg.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" {
		t.Errorf("campaign mail missing one-click headers: %v", msg.Headers)
	}

	// Same matching minute again: nothing more goes out.
	sent, err = Tick(f.app, f.now, f.loc, testBase)
	if err != nil || sent != 0 {
		t.Errorf("second Tick = %d, %v; want 0, nil", sent, err)
	}
	if n := f.app.TestMailer.TotalSend(); n != 2 {
		t.Errorf("mails after second tick = %d, want still 2", n)
	}
}

func TestTickDoesNothingOffSchedule(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(f *tickFixture)
		nowAdd  time.Duration
		baseURL string
	}{
		{name: "one minute late", nowAdd: time.Minute},
		{name: "wrong weekday", nowAdd: 24 * time.Hour},
		{name: "toggle off", mutate: func(f *tickFixture) { setNewsletterSettings(t, f.app, false, nil) }},
		{name: "smtp off", mutate: func(f *tickFixture) { setMail(f.app, false, "news@gather.test") }},
		{name: "no sender address", mutate: func(f *tickFixture) { setMail(f.app, true, "") }},
		{name: "localhost base url", baseURL: "http://127.0.0.1:8090"},
		{name: "empty base url", baseURL: "-"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newTickFixture(t)
			defer f.app.Cleanup()
			if c.mutate != nil {
				c.mutate(f)
			}
			base := testBase
			if c.baseURL == "-" {
				base = ""
			} else if c.baseURL != "" {
				base = c.baseURL
			}
			sent, err := Tick(f.app, f.now.Add(c.nowAdd), f.loc, base)
			if err != nil || sent != 0 {
				t.Errorf("Tick = %d, %v; want 0, nil", sent, err)
			}
			if n := f.app.TestMailer.TotalSend(); n != 0 {
				t.Errorf("mails sent = %d, want 0", n)
			}
			if !f.pick(t).GetDateTime("newsletter_sent_at").IsZero() {
				t.Error("pick must stay unstamped")
			}
		})
	}
}

func TestTickNoEligiblePick(t *testing.T) {
	f := newTickFixture(t)
	defer f.app.Cleanup()
	p := f.pick(t)
	p.Set("hidden", true)
	if err := f.app.SaveNoValidate(p); err != nil {
		t.Fatal(err)
	}
	sent, err := Tick(f.app, f.now, f.loc, testBase)
	if err != nil || sent != 0 || f.app.TestMailer.TotalSend() != 0 {
		t.Errorf("Tick = %d, %v, mails %d; want nothing", sent, err, f.app.TestMailer.TotalSend())
	}
}

func TestStampSentIsAtomic(t *testing.T) {
	f := newTickFixture(t)
	defer f.app.Cleanup()
	before := f.pick(t).GetString("updated")

	ok, err := stampSent(f.app, f.pickID)
	if err != nil || !ok {
		t.Fatalf("first stamp = %v, %v; want true", ok, err)
	}
	ok, err = stampSent(f.app, f.pickID)
	if err != nil || ok {
		t.Errorf("second stamp = %v, %v; want false", ok, err)
	}
	p := f.pick(t)
	if p.GetDateTime("newsletter_sent_at").IsZero() {
		t.Error("newsletter_sent_at not set")
	}
	if p.GetString("updated") != before {
		t.Error("stamping must not bump `updated` (sitemap lastmod / JSON-LD dateModified)")
	}
}

func TestSendPickSkipsFailedRecipient(t *testing.T) {
	f := newTickFixture(t)
	defer f.app.Cleanup()
	seedSubscriber(t, f.app, "bad@x.test", "confirmed", time.Now())
	f.app.OnMailerSend().BindFunc(func(e *core.MailerEvent) error {
		if e.Message.To[0].Address == "bad@x.test" {
			return errors.New("boom")
		}
		return e.Next()
	})

	n, err := SendPick(f.app, f.pick(t), testBase, 0)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || f.app.TestMailer.TotalSend() != 2 {
		t.Errorf("sent %d (mailer %d); the two good recipients must still get mail", n, f.app.TestMailer.TotalSend())
	}
}

func TestSendTestOnlyToRecipient(t *testing.T) {
	f := newTickFixture(t)
	defer f.app.Cleanup()
	endedID := mkEvent(t, f.app, f.now.AddDate(0, 0, -30), time.Time{}, "published", "")
	ended, _ := f.app.FindRecordById("events", endedID)
	ended.Set("title", "Long Gone Gig")
	if err := f.app.SaveNoValidate(ended); err != nil {
		t.Fatal(err)
	}
	p := f.pick(t)
	p.Set("events", append(p.GetStringSlice("events"), endedID))
	if err := f.app.SaveNoValidate(p); err != nil {
		t.Fatal(err)
	}

	if _, err := SendTest(f.app, "admin@example.com", testBase, f.now); err != nil {
		t.Fatal(err)
	}
	if f.app.TestMailer.TotalSend() != 1 {
		t.Fatalf("want exactly one mail, got %d", f.app.TestMailer.TotalSend())
	}
	msg := f.app.TestMailer.LastMessage()
	if len(msg.To) != 1 || msg.To[0].Address != "admin@example.com" {
		t.Errorf("wrong recipient: %v", msg.To)
	}
	if !strings.HasPrefix(msg.Subject, "[Test] ") {
		t.Errorf("subject should be marked as a test: %q", msg.Subject)
	}
	if !strings.Contains(msg.HTML, "Long Gone Gig") {
		t.Error("test email should include all of the post's events, even ended ones")
	}
	if !f.pick(t).GetDateTime("newsletter_sent_at").IsZero() {
		t.Error("a test send must not stamp the pick")
	}
}

func TestSendTestWithNoPicks(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	enable(t, app)
	if _, err := SendTest(app, "admin@example.com", testBase, time.Now()); err == nil {
		t.Error("expected an error when there are no picks posts")
	}
}

func testSubject(t *testing.T, f *tickFixture) string {
	t.Helper()
	f.app.TestMailer.Reset()
	if _, err := SendTest(f.app, "admin@example.com", testBase, f.now); err != nil {
		t.Fatal(err)
	}
	return f.app.TestMailer.LastMessage().Subject
}

func TestSendTestPreviewsThePostTheRealSendWouldPick(t *testing.T) {
	f := newTickFixture(t) // "weekend-picks" is this weekend's eligible post
	defer f.app.Cleanup()
	farEvent := mkEvent(t, f.app, f.now.AddDate(0, 0, 40), time.Time{}, "published", "")
	mkPick(t, f.app, pickSpec{slug: "far-future-picks", start: "2026-12-15", events: []string{farEvent}})

	if got := testSubject(t, f); got != "[Test] weekend-picks" {
		t.Errorf("subject = %q, want the eligible weekend post, not the furthest-future one", got)
	}
}

func TestSendTestFallsBackToNearestUpcomingPost(t *testing.T) {
	f := newTickFixture(t)
	defer f.app.Cleanup()
	// Hide the weekend post so nothing is eligible this week.
	wp := f.pick(t)
	wp.Set("hidden", true)
	if err := f.app.SaveNoValidate(wp); err != nil {
		t.Fatal(err)
	}
	nearEv := mkEvent(t, f.app, f.now.AddDate(0, 0, 16), time.Time{}, "published", "")
	farEv := mkEvent(t, f.app, f.now.AddDate(0, 0, 40), time.Time{}, "published", "")
	mkPick(t, f.app, pickSpec{slug: "far-future-picks", start: "2026-12-15", events: []string{farEv}})
	mkPick(t, f.app, pickSpec{slug: "near-picks", start: "2026-11-20", events: []string{nearEv}})

	if got := testSubject(t, f); got != "[Test] near-picks" {
		t.Errorf("subject = %q, want the nearest upcoming post", got)
	}
}
