package newsletter

import (
	"errors"
	"fmt"
	"html"
	"net/mail"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/pocketbase/pocketbase/tools/security"
)

var (
	ErrInvalidEmail = errors.New("invalid email address")
	ErrUnavailable  = errors.New("newsletter is unavailable")
)

const (
	confirmCooldown = 24 * time.Hour
	confirmTTL      = 48 * time.Hour
	pendingMaxAge   = 7 * 24 * time.Hour
	subscribersCol  = "newsletter_subscribers"
	dbTimeFormat    = "2006-01-02 15:04:05.000Z"
)

// PendingConfirmation is a confirmation email that still needs sending.
type PendingConfirmation struct {
	To         string
	ConfirmURL string
	PrivacyURL string
}

func baseOf(baseURL string) string { return strings.TrimRight(baseURL, "/") }

// ConfirmPageURL is the SPA page with the confirm button. The token rides in
// the fragment so it never reaches server logs or the Umami pageview.
func ConfirmPageURL(baseURL, token string) string {
	return baseOf(baseURL) + "/newsletter/confirm#token=" + token
}

// UnsubscribePageURL is the SPA page with the unsubscribe button.
func UnsubscribePageURL(baseURL, token string) string {
	return baseOf(baseURL) + "/newsletter/unsubscribe#token=" + token
}

// UnsubscribePostURL is the RFC 8058 one-click target (POST).
func UnsubscribePostURL(baseURL, token string) string {
	return baseOf(baseURL) + "/api/newsletter/unsubscribe?token=" + token
}

func normalizeEmail(raw string) (string, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil || addr.Address == "" {
		return "", ErrInvalidEmail
	}
	return strings.ToLower(addr.Address), nil
}

func findByEmail(app core.App, email string) *core.Record {
	rec, err := app.FindFirstRecordByFilter(subscribersCol, "email = {:e}", dbx.Params{"e": email})
	if err != nil {
		return nil
	}
	return rec
}

// Subscribe registers an address and returns the confirmation email that must
// be sent (nil when nothing should be sent: honeypot, already confirmed, or
// still inside the resend cooldown). Callers must answer identically in every
// non-error case so the response never reveals whether an address is known.
func Subscribe(app core.App, rawEmail, honeypot, baseURL string, now time.Time) (*PendingConfirmation, error) {
	// A localhost/empty base URL would mail strangers dead confirmation links.
	if !LoadConfig(app).Enabled() || localOnly(baseURL) {
		return nil, ErrUnavailable
	}
	if strings.TrimSpace(honeypot) != "" {
		return nil, nil // bot: pretend it worked
	}
	email, err := normalizeEmail(rawEmail)
	if err != nil {
		return nil, err
	}

	rec := findByEmail(app, email)
	switch {
	case rec != nil && rec.GetString("status") == "confirmed":
		return nil, nil
	case rec != nil:
		sent := rec.GetDateTime("confirm_sent_at").Time()
		if !sent.IsZero() && now.Sub(sent) < confirmCooldown {
			return nil, nil
		}
	default:
		col, err := app.FindCollectionByNameOrId(subscribersCol)
		if err != nil {
			return nil, err
		}
		rec = core.NewRecord(col)
		rec.Set("email", email)
		rec.Set("status", "pending")
		rec.Set("unsubscribe_token", security.RandomString(32))
	}

	rec.Set("confirm_token", security.RandomString(32))
	rec.Set("confirm_sent_at", now)
	if err := app.Save(rec); err != nil {
		// A concurrent signup for the same address wins the unique index: that
		// is still a success from the caller's point of view.
		if findByEmail(app, email) != nil && rec.IsNew() {
			return nil, nil
		}
		if strings.Contains(err.Error(), "email") {
			return nil, ErrInvalidEmail
		}
		return nil, err
	}
	return &PendingConfirmation{
		To:         email,
		ConfirmURL: ConfirmPageURL(baseURL, rec.GetString("confirm_token")),
		PrivacyURL: privacyAbsolute(app, baseURL),
	}, nil
}

func privacyAbsolute(app core.App, baseURL string) string {
	if p := PrivacyURL(app); p != "" {
		return baseOf(baseURL) + p
	}
	return ""
}

// Confirm completes double opt-in. It is idempotent for an already-confirmed
// record with the same token.
func Confirm(app core.App, token string, now time.Time) bool {
	if token == "" {
		return false
	}
	rec, err := app.FindFirstRecordByFilter(subscribersCol, "confirm_token = {:t}", dbx.Params{"t": token})
	if err != nil {
		return false
	}
	if rec.GetString("status") == "confirmed" {
		return true
	}
	sent := rec.GetDateTime("confirm_sent_at").Time()
	if sent.IsZero() || now.Sub(sent) > confirmTTL {
		return false
	}
	rec.Set("status", "confirmed")
	rec.Set("confirmed_at", now)
	return app.Save(rec) == nil
}

// Unsubscribe hard-deletes the subscriber. Unknown tokens return false.
func Unsubscribe(app core.App, token string) bool {
	if token == "" {
		return false
	}
	rec, err := app.FindFirstRecordByFilter(subscribersCol, "unsubscribe_token = {:t}", dbx.Params{"t": token})
	if err != nil {
		return false
	}
	return app.Delete(rec) == nil
}

// PurgePending deletes unconfirmed signups whose last confirmation went out
// more than 7 days ago (or never went out).
func PurgePending(app core.App, now time.Time) (int, error) {
	cutoff := now.UTC().Add(-pendingMaxAge).Format(dbTimeFormat)
	recs, err := app.FindRecordsByFilter(
		subscribersCol,
		"status = 'pending' && (confirm_sent_at = '' || confirm_sent_at < {:cutoff})",
		"", 0, 0,
		dbx.Params{"cutoff": cutoff},
	)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, r := range recs {
		if err := app.Delete(r); err != nil {
			app.Logger().Error("newsletter purge: delete failed", "id", r.Id, "error", err)
			continue
		}
		deleted++
	}
	return deleted, nil
}

func siteName(app core.App) string {
	if rec, err := app.FindFirstRecordByFilter("settings", ""); err == nil {
		if n := strings.TrimSpace(rec.GetString("instance_name")); n != "" {
			return n
		}
	}
	return "Gather"
}

// SendConfirmation emails the double opt-in link.
func SendConfirmation(app core.App, pc *PendingConfirmation) error {
	name := siteName(app)
	privacyHTML, privacyText := "", ""
	if pc.PrivacyURL != "" {
		privacyHTML = fmt.Sprintf(`<p style="font-size:13px;color:#64748b;">Privacy policy: <a href=%q>%s</a></p>`, pc.PrivacyURL, html.EscapeString(pc.PrivacyURL))
		privacyText = "\nPrivacy policy: " + pc.PrivacyURL + "\n"
	}
	htmlBody := fmt.Sprintf(`<div style="font-family:system-ui,sans-serif;max-width:520px;margin:0 auto;padding:24px;color:#0f172a;">
<h2 style="margin:0 0 12px;">Confirm your subscription</h2>
<p>You asked to get the weekend picks from %s by email, once a week.</p>
<p><a href=%q style="display:inline-block;background:#0d9488;color:#fff;padding:10px 18px;border-radius:6px;text-decoration:none;">Confirm my subscription</a></p>
<p style="font-size:13px;color:#64748b;">If that wasn't you, ignore this email and nothing will happen. The link expires in 48 hours.</p>
%s</div>`, html.EscapeString(name), pc.ConfirmURL, privacyHTML)
	textBody := fmt.Sprintf("You asked to get the weekend picks from %s by email, once a week.\n\nConfirm your subscription:\n%s\n\nIf that wasn't you, ignore this email and nothing will happen. The link expires in 48 hours.\n%s", name, pc.ConfirmURL, privacyText)

	msg := &mailer.Message{
		From:    mail.Address{Name: app.Settings().Meta.SenderName, Address: app.Settings().Meta.SenderAddress},
		To:      []mail.Address{{Address: pc.To}},
		Subject: "Confirm your subscription to " + name,
		HTML:    htmlBody,
		Text:    textBody,
	}
	return app.NewMailClient().Send(msg)
}
