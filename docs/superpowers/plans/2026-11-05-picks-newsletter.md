# Picks Newsletter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Email the upcoming weekend's picks post to confirmed subscribers once a week, only when an eligible post exists, with signup, privacy, tracking and an admin master switch.

**Architecture:** A new Go package `internal/newsletter` owns subscribers, eligibility, rendering, sending and tracking. It is wired into `main.go` (routes, rate limit, cron, CORP header) the same way `internal/cron` and the ActivityPub routes are. The Preact frontend gets a signup box, two action pages and an admin panel. Mail goes through `app.NewMailClient()` (PocketBase's existing config), as `internal/hooks/notifications.go` does.

**Tech Stack:** Go, PocketBase v0.40.4 (`core`, `tools/mailer`, `tools/routine`, `tests.TestApp` with `app.TestMailer`), goldmark + bluemonday (already in `internal/seo/markdown.go`), Preact + TypeScript + Vite + Vitest.

**Spec:** `docs/superpowers/specs/2026-11-05-picks-newsletter-design.md`

## Global Constraints

- Subscribers: email only, **double opt-in**, one-click unsubscribe that **hard-deletes** the record. No IP or user agent stored.
- Confirm and unsubscribe are **POST-only state changes** behind an SPA button page; emailed page links carry the token in the **URL fragment** (`#token=`), never the query. The `List-Unsubscribe` header URL is `<base>/api/newsletter/unsubscribe?token=…` (query), form-encoded one-click body, angle brackets around the URL.
- Effective enabled = `newsletter_enabled` AND `app.Settings().SMTP.Enabled` AND `Meta.SenderAddress != ""`, evaluated at call time, never cached.
- Weekend = Friday to Sunday; Sunday-dated posts are included. Do NOT reuse the `picks-ap-announcement` cron window or `ap_announced`.
- Window bounds are plain strings (`YYYY-MM-DD 00:00:00` / `YYYY-MM-DD 23:59:59`) in `dbx.Params`. Never pass a `time.Time` into `dbx.Params` and never call `.UTC()` on window dates.
- An event counts as upcoming when `status == "published"` and `seo.EventLifecycle(ev, now) == seo.Upcoming`.
- Send is at-most-once: stamp `picks.newsletter_sent_at` **before** sending, via atomic compare-and-set that does not bump `updated`. Missing the scheduled minute skips the week (no catch-up).
- Schedule matches on `loc` (production passes `time.Local`, derived from `TZ`), weekday `newsletter_send_day` (0=Sunday) and `HH:MM` `newsletter_send_time`. The tick passes `time.Now().Add(30*time.Second).Truncate(time.Minute)`. Tests pass an explicit `loc`; never mutate `time.Local`.
- Only the earliest eligible post is sent. Sending refuses to run when the base URL is the localhost fallback.
- Subscribe returns the identical neutral response for new / pending / confirmed addresses. The confirmation mail is sent in the background (`routine.FireAndForget`). Honeypot field is `website`; when filled, fake success and do nothing.
- Constants: confirmation cooldown 24h, confirm-token expiry 48h, pending purge 7 days measured from `confirm_sent_at`, subscribe rate limit 5 per 60s per guest IP.
- UTM on every Gather link in the email (including links inside the blurb): `utm_source=newsletter&utm_medium=email&utm_campaign=<picks slug>`.
- Open pixel `GET /newsletter/open/{id}` (id ends `.gif`): per campaign, not per subscriber. Forward to Umami endpoint `umami_host_url` else `seo.OriginFromURL(umami_src)`, with a browser-like User-Agent, http/https only, no redirects, bounded concurrency.
- Privacy and `/privacy` links render only if a `pages` record with slug `privacy` exists.
- `picks.newsletter_sent_at` and `newsletter_sent_count` are **Hidden** fields; `settings.newsletter_*` fields are **not** Hidden.
- Every migration has up and down functions. Every test using `api.NewTestApp` must `defer app.Cleanup()`.
- Commits end with the trailers the session instructs (Co-Authored-By and Claude-Session lines).

## Review Focus

Failure modes the spec implies that no obvious task covers, most likely first:

1. A picks title or site name containing CR/LF or HTML characters must not inject mail headers or markup (Task 5).
2. A blurb with raw HTML or `javascript:` links must come out sanitised, and internal blurb links must carry UTM (Task 5).
3. A post whose events are all ended, cancelled, draft or absent must not be sent; a recurring series with a future occurrence must be (Task 4).
4. Two ticks in the same matching minute must send exactly once (Task 6).
5. Emails differing by case, whitespace or display-name syntax must not create duplicates or reveal whether an address exists (Task 3).

---

## File Structure

- Create `migrations/1709300028_newsletter.go`: collection and field additions.
- Create `internal/newsletter/{config,subscribers,eligibility,email,send,tracking,routes}.go` and one `_test.go` for each.
- Modify `internal/seo/markdown.go`: export `RenderMarkdownHTML` and `RenderMarkdownHTMLWith` (link rewrite hook).
- Modify `internal/api/testhelpers.go`: `NewTestApp` takes `testing.TB`.
- Modify `main.go`: rate-limit rules, CORP override, `newsletter.Register`, `newsletter.RegisterCron`.
- Modify `frontend/src/lib/pocketbase.ts` (`Settings` type), `frontend/src/app.tsx` (routes), `frontend/src/pages/Home.tsx` (two placements), `frontend/src/components/SettingsForm.tsx` (fields + panel).
- Create `frontend/src/components/NewsletterSignup.{tsx,css}`, `frontend/src/pages/Newsletter.tsx`, `frontend/src/components/NewsletterAdminPanel.tsx`.
- Create `docs/privacy-policy-template.md`; modify `README.md`.

---

### Task 1: Migration

**Files:**
- Create: `migrations/1709300028_newsletter.go`
- Test: `internal/newsletter/migration_test.go`

**Interfaces:**
- Produces: collection `newsletter_subscribers`: `email` (Email, required, unique index), `status` (select `pending`,`confirmed`), `confirm_token` and `unsubscribe_token` (text, each with a **unique** index), `confirm_sent_at`, `confirmed_at` (dates), `created` autodate (add explicitly; `NewBaseCollection` does not), all rules nil. `picks` gains Hidden `newsletter_sent_at` (date) and Hidden `newsletter_sent_count` (number). `settings` gains `newsletter_enabled` (bool), `newsletter_send_day` (number, `Min 0`, `Max 6`, `OnlyInt`), `newsletter_send_time` (text, `Pattern ^([01]\d|2[0-3]):[0-5]\d$`), `newsletter_footer` (text).

- [ ] **Step 1: Write failing test** `TestNewsletterSchema` (use `api.NewTestApp(t)`, `defer app.Cleanup()`, import `_ "gather/migrations"`): assert the collection and every field above, nil `ListRule`, both token indexes unique, the two `picks` fields have `Hidden == true`, and the `settings` fields exist and are not hidden. Add `TestSendTimePattern`: saving settings with `newsletter_send_time = "9:00"` fails validation and `"09:00"` passes.
- [ ] **Step 2: Run** `go test ./internal/newsletter -run 'Schema|SendTimePattern' -v`. Expected: FAIL.
- [ ] **Step 3: Implement** the migration following `1709300020_picks_ap_announced.go` and `1709300013_settings_umami_host_url.go`; down function removes the collection and added fields.
- [ ] **Step 4: Run** the same tests. Expected: PASS.
- [ ] **Step 5: Commit** `feat(newsletter): schema migration`.

---

### Task 2: Test helper tweak, config and effective-enabled

**Files:**
- Modify: `internal/api/testhelpers.go` (`NewTestApp(t testing.TB)`; update the single body use of `t.Fatalf`, existing callers still compile)
- Create: `internal/newsletter/config.go`
- Test: `internal/newsletter/config_test.go`

**Interfaces:**
- Produces:
  - `type Config struct { ToggleOn, MailConfigured bool; SendDay int; SendTime, Footer string }`
  - `func LoadConfig(app core.App) Config` (first `settings` record via `FindFirstRecordByFilter("settings", "")`; zero Config if none; `MailConfigured = app.Settings().SMTP.Enabled && app.Settings().Meta.SenderAddress != ""`)
  - `func (c Config) Enabled() bool`
  - `func PrivacyURL(app core.App) string` (`"/privacy"` if a `pages` record has slug `privacy`, else `""`)

- [ ] **Step 1: Write failing tests:** `TestEnabledRequiresMail` (toggle on + SMTP off → false; SMTP on + empty sender address → false; both fine → true; toggle off → false); `TestPrivacyURL`.
- [ ] **Step 2: Run** `go test ./internal/newsletter -run 'Enabled|Privacy' -v`. Expected: FAIL.
- [ ] **Step 3: Implement** as specified.
- [ ] **Step 4: Run** `go test ./... -count=1`. Expected: PASS (confirms the helper change broke nothing).
- [ ] **Step 5: Commit** `feat(newsletter): config and effective-enabled`.

---

### Task 3: Subscribers

**Files:**
- Create: `internal/newsletter/subscribers.go`
- Test: `internal/newsletter/subscribers_test.go`

**Interfaces:**
- Consumes: `LoadConfig`, `Config.Enabled`, `PrivacyURL` (Task 2).
- Produces:
  - `var ErrInvalidEmail, ErrUnavailable error`
  - `type PendingConfirmation struct { To, ConfirmURL, PrivacyURL string }`
  - `func Subscribe(app core.App, rawEmail, honeypot, baseURL string, now time.Time) (*PendingConfirmation, error)` (nil, nil when nothing needs sending: honeypot, already confirmed, within cooldown)
  - `func SendConfirmation(app core.App, pc *PendingConfirmation) error` (multipart mail via `app.NewMailClient()`, states plainly what they are signing up for, privacy link only if set)
  - `func Confirm(app core.App, token string, now time.Time) bool`
  - `func Unsubscribe(app core.App, token string) bool` (idempotent; unknown token false)
  - `func PurgePending(app core.App, now time.Time) (int, error)` (pending with `confirm_sent_at` older than 7 days, or empty, deleted)
  - `func normalizeEmail(raw string) (string, error)` (trim, `net/mail` parse, use `.Address`, lowercase; PocketBase validation failures on save also map to `ErrInvalidEmail`)
  - `func ConfirmPageURL(baseURL, token string) string` → `<base>/newsletter/confirm#token=<t>`; `func UnsubscribePageURL(baseURL, token string) string` → `<base>/newsletter/unsubscribe#token=<t>`; `func UnsubscribePostURL(baseURL, token string) string` → `<base>/api/newsletter/unsubscribe?token=<t>`

- [ ] **Step 1: Write failing tests** (count sends with `app.TestMailer`; call `SendConfirmation` on the returned value):
  - `TestSubscribeCreatesPendingAndReturnsConfirmation`: one pending record; returned `To` is lowercased; `ConfirmURL` uses the fragment form.
  - `TestSubscribeUnavailableWhenDisabled`: `ErrUnavailable`, no record.
  - `TestSubscribeInvalidEmail`: `"nope"` and an empty string give `ErrInvalidEmail`.
  - `TestSubscribeHoneypotFakesSuccess`: honeypot `"x"` returns `nil, nil`, no record.
  - `TestSubscribeNormalisesAndDedupes` (**Review Focus 5**): `" Foo@Example.COM "`, `"foo@example.com"` and `"Foo <foo@example.com>"` yield one record; calls within 24h return nil confirmation; an already-confirmed address returns nil, nil.
  - `TestSubscribeResendsAfterCooldown`: pending with `confirm_sent_at` 25h ago returns a confirmation and rotates `confirm_token`.
  - `TestConfirm`: valid → `confirmed` with `confirmed_at`; token older than 48h → false; record with empty `confirm_sent_at` → false; unknown → false.
  - `TestUnsubscribeDeletes`: record gone; second call false, no error.
  - `TestPurgePending`: stale and empty-`confirm_sent_at` pending deleted; a pending created 8 days ago but re-sent yesterday kept; confirmed kept.
  - `TestSendConfirmationContent`: mail contains the confirm URL; privacy link present only when a `privacy` page exists.
- [ ] **Step 2: Run** `go test ./internal/newsletter -run 'Subscribe|Confirm|Unsubscribe|Purge' -v`. Expected: FAIL.
- [ ] **Step 3: Implement.** Tokens via `security.RandomString(32)` (`github.com/pocketbase/pocketbase/tools/security`). Stamp `confirm_sent_at` when a confirmation is returned. A unique-index violation on create is treated as the neutral nil, nil.
- [ ] **Step 4: Run.** Expected: PASS.
- [ ] **Step 5: Commit** `feat(newsletter): double opt-in subscribers`.

---

### Task 4: Eligibility

**Files:**
- Create: `internal/newsletter/eligibility.go`
- Test: `internal/newsletter/eligibility_test.go`

**Interfaces:**
- Produces:
  - `func weekendWindow(now time.Time) (fri, sun time.Time)` (Mon–Thu → coming Fri/Sun; Fri–Sun → this weekend; date-only values built with `time.Date(y, m, d, 0, 0, 0, 0, now.Location())`)
  - `func EligiblePick(app core.App, now time.Time) (*core.Record, error)` (nil, nil when none; earliest `start_date` wins)
  - `func upcomingEvents(app core.App, pick *core.Record, now time.Time) ([]*core.Record, error)` (published and `seo.EventLifecycle(ev, now) == seo.Upcoming`, ordered by `start_datetime`)

- [ ] **Step 1: Write failing tests:**
  - `TestWeekendWindow` table: Wed 2026-11-04 → Fri 2026-11-06 / Sun 2026-11-08; Fri 2026-11-06 → same; Sat 2026-11-07 → same; Sun 2026-11-08 → same; Mon 2026-11-09 → 2026-11-13 / 2026-11-15; plus a row spanning a DST change in a non-UTC `loc`.
  - `TestEligiblePick` (**Review Focus 3**): eligible: Friday-start post with a future published event; Sunday-dated post; a post whose only event is a recurring series with a past first date and a future occurrence. Ineligible: hidden; `newsletter_sent_at` set; outside the window; all events ended; only cancelled/draft/pending events; no events. With two valid posts, the earlier `start_date` wins.
- [ ] **Step 2: Run** `go test ./internal/newsletter -run 'Weekend|Eligible' -v`. Expected: FAIL.
- [ ] **Step 3: Implement.** `FindRecordsByFilter` with string params for the window; `app.FindRecordsByIds` for events.
- [ ] **Step 4: Run.** Expected: PASS.
- [ ] **Step 5: Commit** `feat(newsletter): weekend picks eligibility`.

---

### Task 5: Email rendering

**Files:**
- Modify: `internal/seo/markdown.go` (add `RenderMarkdownHTMLWith(md, baseURL string, rewrite func(dest string) string) string`, applied to each absolute link destination after the base-URL join; `RenderMarkdownHTML(md, baseURL)` calls it with identity; `renderMarkdownHTML` delegates)
- Create: `internal/newsletter/email.go`
- Test: `internal/newsletter/email_test.go`, plus a rewrite-hook case in the existing `internal/seo/markdown_test.go`

**Interfaces:**
- Consumes: `UnsubscribePageURL`, `UnsubscribePostURL` (Task 3), `seo.RenderMarkdownHTMLWith`, the server-side time formatter in `internal/seo/timefmt.go`, `recurrence.Expand` for the next occurrence of recurring events.
- Produces:
  - `type Campaign struct { Pick *core.Record; Events []*core.Record; SiteName, BaseURL, PrivacyURL, Footer, UnsubscribeToken string; Now time.Time; PlaceName func(*core.Record) string; ThumbURL func(*core.Record) string }`
  - `func RenderCampaign(c Campaign) (subject, htmlBody, textBody string, headers map[string]string)` (headers: `List-Unsubscribe: <post-url>`, `List-Unsubscribe-Post: List-Unsubscribe=One-Click`, `Precedence: bulk`)
  - `func addUTM(rawURL, baseURL, campaign string) string` (compares scheme+host via `url.Parse`, never string prefix; appends the three params, preserves existing query; other hosts untouched)
  - Content: table layout, inline CSS in the style of `emailHTML` in `internal/hooks/notifications.go`; header with site name; title; full blurb; one card per event (thumbnail only if present, title, next-occurrence time, place only if present) each linking with UTM; "View on the web" link with UTM; open pixel `<img src="<base>/newsletter/open/<picksId>.gif" width="1" height="1" alt="">`; footer with unsubscribe page link, privacy link only if `PrivacyURL != ""`, operator footer text.

- [ ] **Step 1: Write failing tests:**
  - `TestAddUTM`: adds all three params; preserves `?a=1`; `https://gather.example.evil.com/x` is untouched when base is `https://gather.example`; relative and `mailto:` untouched.
  - `TestRenderCampaignLinksAndFooter`: fixture blurb includes an internal link `[x](/event/abc)`; every `href` on the base host in the HTML (including the blurb link) contains `utm_campaign=<slug>` except the unsubscribe and privacy links; headers present with `<` `>` around the URL; privacy link absent when `PrivacyURL == ""`; footer text present; pixel `src` ends `<picksId>.gif`; plain text contains the unsubscribe page URL.
  - `TestRenderCampaignSanitisesBlurb` (**Review Focus 2**): `<script>alert(1)</script> [x](javascript:alert(1))` yields no `<script` and no `javascript:`.
  - `TestRenderCampaignEscapesAndStripsHeaderInjection` (**Review Focus 1**): title `A\r\nBcc: evil@x.com <b>` → subject without `\r`/`\n`; HTML contains `&lt;b&gt;`, not `<b>`.
  - `TestRenderCampaignRecurringShowsNextOccurrence`: a weekly series started months ago shows a date on or after `Now`.
  - `TestRenderCampaignSize`: 30 events plus a long blurb stays under 100KB.
  - seo: `TestRenderMarkdownHTMLWithRewrite` shows the hook sees absolute link destinations and its return value is rendered.
- [ ] **Step 2: Run** `go test ./internal/newsletter ./internal/seo -run 'UTM|RenderCampaign|RenderMarkdown' -v`. Expected: FAIL.
- [ ] **Step 3: Implement.** Blurb through `seo.RenderMarkdownHTMLWith(blurb, baseURL, func(d string) string { return addUTM(d, baseURL, slug) })`; everything else `html.EscapeString`. Subject is the title with all control characters replaced by spaces and trimmed.
- [ ] **Step 4: Run** the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(newsletter): campaign email rendering`.

---

### Task 6: Sending, schedule tick, tracking pixel

**Files:**
- Create: `internal/newsletter/send.go`, `internal/newsletter/tracking.go`
- Test: `internal/newsletter/send_test.go`, `internal/newsletter/tracking_test.go`

**Interfaces:**
- Consumes: `LoadConfig`, `EligiblePick`, `upcomingEvents`, `RenderCampaign`, `PrivacyURL`.
- Produces:
  - `func Tick(app core.App, now time.Time, loc *time.Location, baseURL string) (sent int, err error)`
  - `func stampSent(app core.App, pickID string) (bool, error)` (`app.DB().Update("picks", dbx.Params{"newsletter_sent_at": types.NowDateTime().String()}, dbx.HashExp{"id": pickID, "newsletter_sent_at": ""}).Execute()`; true only if one row changed)
  - `func SendPick(app core.App, pick *core.Record, baseURL string, delay time.Duration) (int, error)` (confirmed subscribers only, one mail each, per-recipient failure logged and skipped, `newsletter_sent_count` written periodically and at the end by direct DB update)
  - `func SendTest(app core.App, to, baseURL string) error` (latest non-hidden picks post by `start_date` descending, **all** its events, dummy unsubscribe token, only to `to`, nothing stamped; error if no picks)
  - `func umamiTarget(app core.App) (endpoint, websiteID string)` (`umami_host_url` else `seo.OriginFromURL(umami_src)`, trailing slash trimmed; empty if either part missing)
  - `func ReportOpen(endpoint, websiteID, baseURL, slug string) error` (POST JSON `{"type":"event","payload":{"website":…,"hostname":<host of baseURL>,"url":"/picks/<slug>","name":"newsletter-open","data":{"campaign":"<slug>"}}}` to `<endpoint>/api/send`; 3s timeout; fixed browser-like `User-Agent`; `CheckRedirect` refuses redirects; endpoint scheme must be http/https)
  - `func PixelHandler(app core.App, baseURL string) func(*core.RequestEvent) error` (strip `.gif` from `PathValue("id")`; id must match `^[a-z0-9]{15}$`; synchronous lookup, forward only when the pick exists and `newsletter_sent_at` is set; forwarding goroutine guarded by a small semaphore, dropped when full; always responds with a 1×1 GIF, `image/gif`, `Cache-Control: no-store`)

- [ ] **Step 1: Write failing tests:**
  - `TestTickSendsOnceAtScheduledMinute`: explicit `loc`; config day/time match `now`; one eligible pick, two confirmed and one pending subscriber → 2 mails, pick stamped, `newsletter_sent_count == 2`; a second `Tick` with the same `now` sends 0 (**Review Focus 4**).
  - `TestTickDoesNothingOffSchedule`: one minute late, wrong weekday, toggle off, SMTP off, empty sender address, and a localhost-fallback `baseURL` each send 0 and leave the pick unstamped.
  - `TestTickNoEligiblePick`: 0, no error.
  - `TestStampSentIsAtomic`: first call true, second false; `updated` unchanged.
  - `TestSendPickSkipsFailedRecipient`: bind `app.OnMailerSend().BindFunc(...)` returning an error when `e.Message.To[0].Address == "bad@x.test"` and otherwise `e.Next()`; other recipients still receive mail.
  - `TestSendTestOnlyToRecipient`: one mail to the address, pick not stamped, includes cards for past events too.
  - `TestUmamiTarget`: host url wins; falls back to `umami_src` origin; empty when no website id.
  - `TestReportOpenPostsToUmami`: `httptest.Server`; asserts path `/api/send`, payload name and campaign, a non-Go `User-Agent`, and that a 302 from the server is not followed.
  - `TestPixelHandler`: valid sent pick id → GIF (`GIF89a`) and one forward; unsent pick, unknown id and a malformed id → GIF and no forward and no DB error.
- [ ] **Step 2: Run** `go test ./internal/newsletter -run 'Tick|Stamp|SendPick|SendTest|Umami|ReportOpen|Pixel' -v`. Expected: FAIL.
- [ ] **Step 3: Implement.** `Tick`: bail unless enabled and base URL is not the localhost fallback; compare `now.In(loc)` weekday and `Format("15:04")` to config; `EligiblePick`; `stampSent` (stop if false); log "stamped, sending to N"; `SendPick` with a 200ms delay.
- [ ] **Step 4: Run.** Expected: PASS.
- [ ] **Step 5: Commit** `feat(newsletter): scheduled send and open tracking`.

---

### Task 7: HTTP routes, cron, main.go wiring

**Files:**
- Create: `internal/newsletter/routes.go`
- Modify: `main.go` (rate-limit rule list near line 61; CORP middleware near line 128; call sites next to `cron.Register(se.App, baseURL)` near line 803)
- Test: `internal/newsletter/routes_test.go`

**Interfaces:**
- Consumes: everything above.
- Produces:
  - `func Register(se *core.ServeEvent, baseURL string)` with:
    - `GET /api/newsletter/config` → `{"enabled": bool, "privacy_url": string|null}`
    - `POST /api/newsletter/subscribe` body `{email, website}`: `ErrInvalidEmail` → 400 `{"message":"Please enter a valid email address."}`; `ErrUnavailable` → 503; otherwise 200 `{"message":"Check your inbox to confirm."}`; a non-nil confirmation is sent via `routine.FireAndForget(func(){ SendConfirmation(...) })` with errors logged
    - `POST /api/newsletter/confirm?token=` → 200 `{"status":"confirmed"}` or 400 `{"status":"invalid"}`
    - `POST /api/newsletter/unsubscribe?token=` → 200 `{"status":"ok"}` always; must accept a form-encoded body
    - `GET /newsletter/open/{id}` → `PixelHandler`
    - `GET /api/newsletter/admin` (admin or superuser, copy the check from `/api/ap/test-delivery`) → `{"enabled","toggle_on","mail_configured": bool, "confirmed","pending": int, "timezone": string, "send_day","send_time": …}`
    - `POST /api/newsletter/test` (admin or superuser) → `SendTest` to the caller's own email
  - `func RegisterCron(app core.App, baseURL string)`: `newsletter-tick` `* * * * *` calling `Tick(app, time.Now().Add(30*time.Second).Truncate(time.Minute), time.Local, baseURL)`; `newsletter-purge` `0 4 * * *` calling `PurgePending`.
  - `main.go`: guest and auth rate-limit rules for label `/api/newsletter/subscribe` (5/60s guest, 1000/60s auth); CORP header becomes `cross-origin` for paths starting `/api/files/` or `/newsletter/open/`, `same-origin` otherwise.

- [ ] **Step 1: Write failing tests** with `tests.ApiScenario` (new to this repo; set it up explicitly): `TestAppFactory: func(t testing.TB) *tests.TestApp { return api.NewTestApp(t) }`; `BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) { newsletter.Register(e, "https://x.test"); app.Settings().SMTP.Enabled = true; app.Settings().Meta.SenderAddress = "a@x.test"; /* seed settings with toggle on */ }`; `ExpectedContent` is required for every non-empty body (`"GIF89a"` for the pixel; the JSON message for subscribe); auth via `{"Authorization": user.Token}` from `api.CreateTestUser`. Cases: config disabled without SMTP, enabled with it and `privacy_url` set once a `privacy` page exists; subscribe 400 bad email, 503 disabled, 200 and a pending record when enabled, honeypot 200 and no record; confirm 200 then 400 on reuse of a bad token; unsubscribe 200 for unknown token and 200 for a form-encoded body; admin 401 guest / 403 plain user / 200 admin; pixel route returns a GIF for `.gif` ids.
- [ ] **Step 2: Run** `go test ./internal/newsletter -run Routes -v`. Expected: FAIL.
- [ ] **Step 3: Implement** and wire into `main.go`.
- [ ] **Step 4: Run** `go test ./... -count=1 -race`. Expected: PASS, including existing suites.
- [ ] **Step 5: Smoke** with `make dev-backend`: `curl -s localhost:8090/api/newsletter/config` returns JSON (not SPA HTML); `curl -sI localhost:8090/api/files/x` shows the `cross-origin` CORP header; check startup logs show no "failed to configure rate limits" warning.
- [ ] **Step 6: Commit** `feat(newsletter): routes, cron, rate limit and CORP`.

---

### Task 8: Frontend signup, action pages

**Files:**
- Modify: `frontend/src/lib/pocketbase.ts` (add `newsletter_enabled: boolean; newsletter_send_day: number; newsletter_send_time: string; newsletter_footer: string` to `Settings`)
- Create: `frontend/src/components/NewsletterSignup.tsx`, `frontend/src/components/NewsletterSignup.css`
- Create: `frontend/src/pages/Newsletter.tsx`
- Modify: `frontend/src/app.tsx` (routes `/newsletter/confirm` and `/newsletter/unsubscribe` before `<Page path="/:slug" />`), `frontend/src/pages/Home.tsx`

**Interfaces:**
- Produces: `NewsletterSignup()` (no props); `NewsletterConfirm({ path?: string })` and `NewsletterUnsubscribe({ path?: string })`, each reading the token from `window.location.hash` (`#token=…`), showing one button, calling `pb.send('/api/newsletter/confirm' | '/api/newsletter/unsubscribe' + '?token=…', { method: 'POST' })` and then the result state ("You're subscribed", "That link has expired, please sign up again", "You're unsubscribed and your address has been deleted"). After a click the page calls `history.replaceState` to drop the hash.
- Signup: on mount `pb.send('/api/newsletter/config', { requestKey: null })`; render nothing unless `enabled`. Form uses the existing `.form-group` input styles and `.btn .btn-primary` button (`frontend/src/style.css`), the off-screen honeypot `website` (CSS off-screen, `tabindex=-1`, `autocomplete="off"`, `aria-hidden="true"`), purpose line "Weekend picks by email. Unsubscribe any time." and a privacy link when `privacy_url` is set. Submit with `pb.send('/api/newsletter/subscribe', { method: 'POST', body: { email, website }, requestKey: null })`; show the returned `message` or the error inline. Button carries `data-umami-event="newsletter-signup"`.
- Placement: inside `sidebarContent` as a new `sidebar-section` directly after "Browse by date", **and** at the end of the main column in a wrapper shown only below 900px (`.home-sidebar` is `display:none` there, `Home.css:8-10`). Both instances must work together (`requestKey: null`).

- [ ] **Step 1:** Add types, components, routes and both placements.
- [ ] **Step 2: Verify** `cd frontend && npx tsc --noEmit && npm run build`. Expected: clean.
- [ ] **Step 3: Manual smoke** with `make dev`: feature off → no box anywhere; enable SMTP and the toggle → box under the calendar at desktop width and at the end of the page at 800px width, never both visible; submit a valid address, see the confirmation message; open `/newsletter/unsubscribe#token=bogus`, press the button, see a result without console errors.
- [ ] **Step 4: Commit** `feat(newsletter): signup box and action pages`.

---

### Task 9: Admin settings and panel

**Files:**
- Modify: `frontend/src/components/SettingsForm.tsx` (extend the form state, load and save blocks and markup following the `umami_*` pattern at roughly lines 30–60, 120–150 and the form body)
- Create: `frontend/src/components/NewsletterAdminPanel.tsx`

**Interfaces:**
- Consumes: `GET /api/newsletter/admin`, `POST /api/newsletter/test`.
- Produces: form fields: enabled checkbox, day-of-week select (0=Sunday), time input (`HH:MM`) with the server's `timezone` shown next to it, footer textarea. `NewsletterAdminPanel()`: status line ("Mail isn't configured in PocketBase" when `mail_configured` is false; "Switched off" when `toggle_on` is false; "No send time set" when empty; otherwise "Sending <day> at <time> (<timezone>)"), confirmed and pending counts, and a "Send test to me" button with inline result text. Rendered inside the settings tab beneath the form.

- [ ] **Step 1:** Implement both.
- [ ] **Step 2: Verify** `cd frontend && npx tsc --noEmit && npm run build`. Expected: clean.
- [ ] **Step 3: Manual smoke:** fields persist across reload; the status line reflects SMTP off and an empty time; with working mail, "Send test to me" delivers the latest picks email to the logged-in admin only.
- [ ] **Step 4: Commit** `feat(newsletter): admin settings and status panel`.

---

### Task 10: Privacy template and docs

**Files:**
- Create: `docs/privacy-policy-template.md`
- Modify: `README.md`

**Produces:** a plain-language template with `[bracketed]` placeholders covering what is collected (email, confirmation timestamp), purpose (weekly picks email only), lawful basis (consent), processors (the configured mail provider; Umami for aggregate stats, no personal data), retention (deleted on unsubscribe; unconfirmed addresses deleted after 7 days), data subject rights, controller identity and contact, and a prompt to note processing outside the operator's region, with a line saying it is a starting point, not legal advice. README "Newsletter" section: how to enable it; PocketBase mail (SMTP and sender address) must be configured; the send schedule follows the `TZ` env var and Docker defaults to `UTC`; operators should publish a `/privacy` page from the template; DKIM should cover `List-Unsubscribe` and `List-Unsubscribe-Post` for one-click unsubscribe to be honoured.

- [ ] **Step 1:** Write both.
- [ ] **Step 2: Verify** `grep -c '\[' docs/privacy-policy-template.md` is greater than 0; read the README section back.
- [ ] **Step 3: Commit** `docs(newsletter): privacy template and README`.

---

## Self-Review Notes

- **Spec coverage:** data model and hidden fields (1); master switch (2, 7, 9); subscribe/confirm/unsubscribe/purge (3, 7, 8); anti-abuse (3, 7, 8); eligibility incl. recurrence and status (4); content, UTM (incl. blurb), headers, conditional privacy link, footer (5); send job, atomic stamp, TZ (6); pixel, Umami fallback and hardening (6, 7); CORP for webmail images (7); admin and test send (7, 9); mobile placement (8); privacy template and DKIM note (10). The AP cron bug stays out of scope.
- **Review provenance:** an Opus review of the spec and plan against the code found the issues folded in here (link-scanner confirm, Umami host fallback and User-Agent, recurring and cancelled events, mobile sidebar hidden, `ApiScenario` not used anywhere, public API field leakage, CORP header, tick jitter, token in fragment). Several were spot-checked against the code before accepting.
