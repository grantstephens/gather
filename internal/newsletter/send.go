package newsletter

import (
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	defaultSendDelay = 200 * time.Millisecond
	countFlushEvery  = 25
)

// localOnly reports whether baseURL is empty or points at this machine (the
// dev fallback), in which case every link in the email would be wrong.
func localOnly(baseURL string) bool {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return true
	}
	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		return true
	}
	return false
}

// Available reports whether signups are open: feature on, mail working and a
// real (non-localhost) base URL.
func Available(app core.App, baseURL string) bool {
	return LoadConfig(app).Enabled() && !localOnly(baseURL)
}

// Tick is called every minute by cron. It sends only when the newsletter is
// enabled, the clock (in loc) is exactly the configured day and HH:MM, and an
// eligible picks post exists. It returns the number of emails sent.
func Tick(app core.App, now time.Time, loc *time.Location, baseURL string) (int, error) {
	cfg := LoadConfig(app)
	if !cfg.Enabled() {
		return 0, nil
	}
	local := now.In(loc)
	if int(local.Weekday()) != cfg.SendDay || local.Format("15:04") != cfg.SendTime {
		return 0, nil
	}
	if localOnly(baseURL) {
		app.Logger().Warn("newsletter: send time reached but the base URL is localhost; set the Application URL (or BASE_URL) so email links work", "base_url", baseURL)
		return 0, nil
	}
	pick, err := EligiblePick(app, local)
	if err != nil {
		return 0, err
	}
	if pick == nil {
		app.Logger().Debug("newsletter: no eligible picks post this week")
		return 0, nil
	}
	// Stamp first: at-most-once. A crash mid-send under-delivers but can never
	// double-send.
	ok, err := stampSent(app, pick.Id)
	if err != nil || !ok {
		return 0, err
	}
	app.Logger().Info("newsletter: picks post stamped, sending", "picks", pick.Id, "slug", pick.GetString("slug"))
	return SendPick(app, pick, baseURL, defaultSendDelay)
}

// stampSent atomically marks the pick as sent. It only succeeds once, and goes
// around the record save so `updated` (sitemap lastmod, JSON-LD) is untouched.
func stampSent(app core.App, pickID string) (bool, error) {
	res, err := app.DB().Update(
		"picks",
		dbx.Params{"newsletter_sent_at": types.NowDateTime().String()},
		dbx.HashExp{"id": pickID, "newsletter_sent_at": ""},
	).Execute()
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func writeSentCount(app core.App, pickID string, n int) {
	if _, err := app.DB().Update("picks", dbx.Params{"newsletter_sent_count": n}, dbx.HashExp{"id": pickID}).Execute(); err != nil {
		app.Logger().Error("newsletter: failed to record sent count", "picks", pickID, "error", err)
	}
}

func buildMessage(app core.App, c Campaign, to string) *mailer.Message {
	subject, htmlBody, textBody, headers := RenderCampaign(c)
	return &mailer.Message{
		From:    mail.Address{Name: app.Settings().Meta.SenderName, Address: app.Settings().Meta.SenderAddress},
		To:      []mail.Address{{Address: to}},
		Subject: subject,
		HTML:    htmlBody,
		Text:    textBody,
		Headers: headers,
	}
}

// SendPick emails the post to every confirmed subscriber, one message each. A
// failure for one recipient is logged and skipped. It returns how many sends
// succeeded.
func SendPick(app core.App, pick *core.Record, baseURL string, delay time.Duration) (int, error) {
	now := time.Now()
	events, err := upcomingEvents(app, pick, now)
	if err != nil {
		return 0, err
	}
	subs, err := app.FindRecordsByFilter(subscribersCol, "status = 'confirmed'", "created", 0, 0)
	if err != nil {
		return 0, err
	}
	client := app.NewMailClient()
	sent := 0
	for i, sub := range subs {
		msg := buildMessage(app, NewCampaign(app, pick, events, baseURL, now, sub.GetString("unsubscribe_token")), sub.GetString("email"))
		if err := client.Send(msg); err != nil {
			app.Logger().Error("newsletter: send failed", "subscriber", sub.Id, "error", err)
		} else {
			sent++
		}
		if sent > 0 && sent%countFlushEvery == 0 {
			writeSentCount(app, pick.Id, sent)
		}
		if delay > 0 && i < len(subs)-1 {
			time.Sleep(delay)
		}
	}
	writeSentCount(app, pick.Id, sent)
	app.Logger().Info("newsletter: send finished", "picks", pick.Id, "sent", sent, "subscribers", len(subs))
	return sent, nil
}

// testPick chooses which post a test email previews: the one the real send
// would pick; failing that the nearest upcoming post; failing that the most
// recent one. Never "the furthest-future post", which is what a plain
// newest-first sort would give.
func testPick(app core.App, now time.Time) (*core.Record, error) {
	if p, err := EligiblePick(app, now); err != nil || p != nil {
		return p, err
	}
	today := now.Format("2006-01-02") + " 00:00:00"
	upcoming, err := app.FindRecordsByFilter("picks", "hidden = false && start_date >= {:today}", "start_date", 1, 0, dbx.Params{"today": today})
	if err != nil {
		return nil, err
	}
	if len(upcoming) > 0 {
		return upcoming[0], nil
	}
	latest, err := app.FindRecordsByFilter("picks", "hidden = false", "-start_date", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(latest) == 0 {
		return nil, errors.New("no picks posts to send")
	}
	return latest[0], nil
}

// SendTest emails a preview (see testPick), with all of the post's events, to
// one address and returns the post it used. Nothing is stamped.
func SendTest(app core.App, to, baseURL string, now time.Time) (*core.Record, error) {
	pick, err := testPick(app, now)
	if err != nil {
		return nil, err
	}
	var events []*core.Record
	if ids := pick.GetStringSlice("events"); len(ids) > 0 {
		events, err = app.FindRecordsByIds("events", ids)
		if err != nil {
			return nil, err
		}
	}
	msg := buildMessage(app, NewCampaign(app, pick, events, baseURL, now, "test"), to)
	msg.Subject = "[Test] " + msg.Subject
	if err := app.NewMailClient().Send(msg); err != nil {
		return nil, err
	}
	return pick, nil
}
