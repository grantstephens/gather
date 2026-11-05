package newsletter

import (
	"strings"
	"testing"
	"time"

	"gather/internal/api"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const testBase = "https://gather.test"

// enable turns the whole feature on (toggle + working mail).
func enable(t testing.TB, app *tests.TestApp) {
	t.Helper()
	setNewsletterSettings(t, app, true, nil)
	setMail(app, true, "news@gather.test")
}

func allSubscribers(t testing.TB, app core.App) []*core.Record {
	t.Helper()
	recs, err := app.FindAllRecords("newsletter_subscribers")
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

// seedSubscriber inserts a subscriber directly.
func seedSubscriber(t testing.TB, app core.App, email, status string, confirmSentAt time.Time) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("newsletter_subscribers")
	if err != nil {
		t.Fatal(err)
	}
	rec := core.NewRecord(col)
	rec.Set("email", email)
	rec.Set("status", status)
	rec.Set("confirm_token", "ct-"+email)
	rec.Set("unsubscribe_token", "ut-"+email)
	if !confirmSentAt.IsZero() {
		rec.Set("confirm_sent_at", confirmSentAt)
	}
	if status == "confirmed" {
		rec.Set("confirmed_at", time.Now())
	}
	if err := app.Save(rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestSubscribeCreatesPendingAndReturnsConfirmation(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	enable(t, app)

	pc, err := Subscribe(app, "Reader@Example.com", "", testBase, time.Now())
	if err != nil || pc == nil {
		t.Fatalf("Subscribe = %v, %v", pc, err)
	}
	if pc.To != "reader@example.com" {
		t.Errorf("To = %q, want lowercased", pc.To)
	}
	recs := allSubscribers(t, app)
	if len(recs) != 1 || recs[0].GetString("status") != "pending" {
		t.Fatalf("want one pending record, got %d", len(recs))
	}
	if want := testBase + "/newsletter/confirm#token=" + recs[0].GetString("confirm_token"); pc.ConfirmURL != want {
		t.Errorf("ConfirmURL = %q, want %q", pc.ConfirmURL, want)
	}
	if recs[0].GetString("confirm_token") == "" || recs[0].GetString("unsubscribe_token") == "" {
		t.Error("tokens must be generated")
	}
}

func TestSubscribeUnavailableWhenDisabled(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	setNewsletterSettings(t, app, true, nil)
	setMail(app, false, "") // no mail config -> feature is off

	pc, err := Subscribe(app, "a@example.com", "", testBase, time.Now())
	if err != ErrUnavailable || pc != nil {
		t.Errorf("want ErrUnavailable, got %v, %v", pc, err)
	}
	if n := len(allSubscribers(t, app)); n != 0 {
		t.Errorf("no record should be created, got %d", n)
	}
}

func TestSubscribeInvalidEmail(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	enable(t, app)
	for _, bad := range []string{"nope", "", "   ", "a@", "@b.com"} {
		if _, err := Subscribe(app, bad, "", testBase, time.Now()); err != ErrInvalidEmail {
			t.Errorf("Subscribe(%q) err = %v, want ErrInvalidEmail", bad, err)
		}
	}
}

func TestSubscribeHoneypotFakesSuccess(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	enable(t, app)
	pc, err := Subscribe(app, "bot@example.com", "http://spam", testBase, time.Now())
	if pc != nil || err != nil {
		t.Errorf("honeypot should return nil, nil; got %v, %v", pc, err)
	}
	if n := len(allSubscribers(t, app)); n != 0 {
		t.Errorf("honeypot must not create a record, got %d", n)
	}
}

func TestSubscribeNormalisesAndDedupes(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	enable(t, app)
	now := time.Now()

	first, err := Subscribe(app, " Foo@Example.COM ", "", testBase, now)
	if err != nil || first == nil {
		t.Fatalf("first: %v %v", first, err)
	}
	for _, variant := range []string{"foo@example.com", "Foo <foo@example.com>", "FOO@example.com"} {
		pc, err := Subscribe(app, variant, "", testBase, now.Add(time.Hour))
		if err != nil || pc != nil {
			t.Errorf("variant %q within cooldown: want nil, nil; got %v, %v", variant, pc, err)
		}
	}
	if n := len(allSubscribers(t, app)); n != 1 {
		t.Errorf("want 1 record, got %d", n)
	}

	seedSubscriber(t, app, "done@example.com", "confirmed", now)
	pc, err := Subscribe(app, "done@example.com", "", testBase, now)
	if pc != nil || err != nil {
		t.Errorf("already confirmed: want nil, nil; got %v, %v", pc, err)
	}
}

func TestSubscribeResendsAfterCooldown(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	enable(t, app)
	now := time.Now()
	old := seedSubscriber(t, app, "late@example.com", "pending", now.Add(-25*time.Hour))
	oldToken := old.GetString("confirm_token")

	pc, err := Subscribe(app, "late@example.com", "", testBase, now)
	if err != nil || pc == nil {
		t.Fatalf("want a fresh confirmation, got %v, %v", pc, err)
	}
	rec, _ := app.FindRecordById("newsletter_subscribers", old.Id)
	if rec.GetString("confirm_token") == oldToken {
		t.Error("confirm_token should rotate on resend")
	}
	if !strings.Contains(pc.ConfirmURL, rec.GetString("confirm_token")) {
		t.Error("ConfirmURL should carry the new token")
	}
}

func TestConfirm(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Now()
	fresh := seedSubscriber(t, app, "fresh@example.com", "pending", now.Add(-time.Hour))
	stale := seedSubscriber(t, app, "stale@example.com", "pending", now.Add(-49*time.Hour))
	never := seedSubscriber(t, app, "never@example.com", "pending", time.Time{})

	if !Confirm(app, fresh.GetString("confirm_token"), now) {
		t.Fatal("valid token should confirm")
	}
	rec, _ := app.FindRecordById("newsletter_subscribers", fresh.Id)
	if rec.GetString("status") != "confirmed" || rec.GetDateTime("confirmed_at").IsZero() {
		t.Errorf("want confirmed with confirmed_at, got %q", rec.GetString("status"))
	}
	if Confirm(app, stale.GetString("confirm_token"), now) {
		t.Error("token older than 48h must not confirm")
	}
	if Confirm(app, never.GetString("confirm_token"), now) {
		t.Error("record with no confirm_sent_at must not confirm")
	}
	if Confirm(app, "bogus", now) || Confirm(app, "", now) {
		t.Error("unknown/empty token must not confirm")
	}
	staleRec, _ := app.FindRecordById("newsletter_subscribers", stale.Id)
	if staleRec.GetString("status") != "pending" {
		t.Error("expired record must stay pending")
	}
}

func TestUnsubscribeDeletes(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	rec := seedSubscriber(t, app, "bye@example.com", "confirmed", time.Now())

	if !Unsubscribe(app, rec.GetString("unsubscribe_token")) {
		t.Fatal("first unsubscribe should succeed")
	}
	if n := len(allSubscribers(t, app)); n != 0 {
		t.Errorf("record should be deleted, %d left", n)
	}
	if Unsubscribe(app, rec.GetString("unsubscribe_token")) {
		t.Error("second unsubscribe should report false (already gone)")
	}
	if Unsubscribe(app, "") {
		t.Error("empty token must be false")
	}
}

func TestPurgePending(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	now := time.Now()
	seedSubscriber(t, app, "stale@example.com", "pending", now.AddDate(0, 0, -8))
	seedSubscriber(t, app, "empty@example.com", "pending", time.Time{})
	seedSubscriber(t, app, "resent@example.com", "pending", now.AddDate(0, 0, -1))
	seedSubscriber(t, app, "ok@example.com", "confirmed", now.AddDate(0, 0, -30))

	n, err := PurgePending(app, now)
	if err != nil || n != 2 {
		t.Fatalf("PurgePending = %d, %v; want 2, nil", n, err)
	}
	var left []string
	for _, r := range allSubscribers(t, app) {
		left = append(left, r.GetString("email"))
	}
	if len(left) != 2 {
		t.Errorf("want resent+ok left, got %v", left)
	}
}

func TestSendConfirmationContent(t *testing.T) {
	app := api.NewTestApp(t)
	defer app.Cleanup()
	enable(t, app)
	pc := &PendingConfirmation{To: "a@example.com", ConfirmURL: testBase + "/newsletter/confirm#token=abc"}

	if err := SendConfirmation(app, pc); err != nil {
		t.Fatal(err)
	}
	msg := app.TestMailer.LastMessage()
	if len(msg.To) != 1 || msg.To[0].Address != "a@example.com" {
		t.Errorf("wrong recipient: %v", msg.To)
	}
	if !strings.Contains(msg.HTML, pc.ConfirmURL) || !strings.Contains(msg.Text, pc.ConfirmURL) {
		t.Error("confirm URL must be in both html and text")
	}
	if strings.Contains(msg.HTML, "/privacy") {
		t.Error("no privacy link without a privacy page")
	}

	pages, _ := app.FindCollectionByNameOrId("pages")
	page := core.NewRecord(pages)
	page.Set("title", "Privacy")
	page.Set("slug", "privacy")
	if err := app.Save(page); err != nil {
		t.Fatal(err)
	}
	pc.PrivacyURL = testBase + "/privacy"
	if err := SendConfirmation(app, pc); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(app.TestMailer.LastMessage().HTML, pc.PrivacyURL) {
		t.Error("privacy link should appear when set")
	}
}
