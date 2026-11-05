# Weekly Picks Newsletter Design

**Date:** 2026-11-05
**Status:** Draft, awaiting approval

## Intent

Gather publishes hand-written "Weekend Picks" posts, but today the only ways to
get them are visiting the site or following the ActivityPub actor. Most people
have neither habit. They do have an email address. This feature emails the
weekend's picks post to anyone who signed up, to broaden Gather's reach beyond
the Fediverse.

## Goals

- Anyone can subscribe with just an email address (no account), from a small
  box on the home page.
- Once a week, at an admin-configured day and time, subscribers get the
  upcoming weekend's picks post as an email.
- It is *smart*: if there is no eligible picks post, nothing is sent. No
  "nothing this week" emails.
- Admins can switch the whole feature on or off, and it is forced off when
  PocketBase has no working mail configuration.
- Engagement is visible in Umami: link clicks and (roughly) opens.

## Non-goals (v1)

- Per-town, per-tag or otherwise segmented lists.
- Any content other than picks posts (no digest of all events, no announcements).
- A built-in analytics dashboard (Umami is the dashboard).
- A third-party email provider integration. We use PocketBase's existing mail
  config only.
- Captcha / Cloudflare Turnstile (see Anti-abuse; can be added later).
- Subscriber import/export UI. The PocketBase admin dashboard is enough.

## Anti-goals (failure even if it technically works)

- Emailing someone who did not confirm their own address.
- Double-sending a picks post to the same subscriber, ever.
- Sending anything when there is no eligible post.
- Becoming a way to spam arbitrary third parties with confirmation emails.
- Breaking or changing existing picks, ActivityPub or settings behaviour.

## Decisions (made by the product owner)

1. Signup is open to anyone, email only, double opt-in, one-click unsubscribe.
2. The email is the picks post (not a separate authored newsletter).
3. Send timing is configurable (day and time) and only fires if an upcoming
   picks post exists.
4. If a picks post is published after the scheduled time, **do nothing** for
   that week. No late send.
5. Reuse PocketBase's existing mail configuration. No new SMTP settings.
6. A master on/off switch exists, and the feature is off whenever PocketBase
   mail is not configured.
7. Tracking: **both** UTM-tagged links and a tracking pixel that reports to Umami.
8. The signup box is small, on the home page under the calendar, in the existing
   site layout and style.
9. Anti-abuse baseline only for v1: honeypot, rate limiting, double opt-in,
   confirmation cooldown. No captcha.

## Existing system facts the builder needs

- Picks live in the `picks` collection (migration `1709300015_picks.go`): `title`,
  `slug`, `blurb` (markdown), `events` (multi relation), `hidden`, `start_date`,
  `ap_announced`.
- "Weekend" for picks means **Friday to Sunday**. The home page
  (`frontend/src/pages/Home.tsx`) selects the current post with
  `hidden = false && start_date >= <Friday> && start_date <= <Sunday>`, and
  "next weekend" by shifting those by 7 days. Most posts have a Friday `start_date`.
- Settings are a singleton in the `settings` collection. Umami fields already
  exist: `umami_src`, `umami_website_id`, `umami_host_url`.
- Scheduled work uses `app.Cron()` in `internal/cron/cron.go`.
- Rate limiting is PocketBase's built-in rule list, set in `main.go` (~line 59).
  There is no captcha anywhere today.
- Long-form blurbs are supported (`docs/superpowers/plans/2026-10-21-picks-longform-blurbs.md`).
  ActivityPub federates only the intro paragraph; the newsletter does not.

## Known unrelated bug (out of scope, flag only)

`internal/cron/cron.go` "picks-ap-announcement" filters
`start_date >= <upcoming Saturday>`. A Friday `start_date` is before that, so
Friday-start posts are probably never announced over ActivityPub. The newsletter
must NOT reuse this window or the `ap_announced` flag. It gets its own flag.
Fixing the AP cron is a separate task.

## Design

### Data model

**New collection `newsletter_subscribers`** (no public API rules; all access via
custom endpoints and superusers):

| Field | Type | Notes |
|---|---|---|
| `email` | Email, required | Lowercased and trimmed on write. Unique index. |
| `status` | Select | `pending`, `confirmed` |
| `confirm_token` | Text | Random, unguessable. Unique index. Expires 48h after `confirm_sent_at`; a record with no `confirm_sent_at` can never be confirmed. |
| `unsubscribe_token` | Text | Random, permanent per subscriber. Unique index. |
| `confirm_sent_at` | Date | Drives the confirmation cooldown and token expiry. |
| `confirmed_at` | Date | Record of consent. |

**`picks` additions:** `newsletter_sent_at` (Date), `newsletter_sent_count`
(Number). `newsletter_sent_at` is the double-send guard. **Both are Hidden
fields**: `picks` is publicly listable, and these must not leak through the
public API (superusers and the admin endpoint still see them).

**`settings` additions:** `newsletter_enabled` (Bool), `newsletter_send_day`
(Number, 0–6 integer, Sunday = 0), `newsletter_send_time` (Text, validated
against `^([01]\d|2[0-3]):[0-5]\d$` so `9:00` is rejected), `newsletter_footer`
(Text, optional). `settings` is publicly readable and the admin form runs as
role=admin (not superuser), so these fields are deliberately **not** Hidden;
none of them is sensitive.

Every migration needs both up and down functions, as existing ones do.

### Master switch

`effective_enabled = newsletter_enabled AND app.Settings().SMTP.Enabled AND
Meta.SenderAddress is non-empty`. (Deliberate choice: operators relying on
PocketBase's sendmail fallback are not supported in v1.)
Evaluated at request/tick time, never cached. When false:

- the signup endpoint refuses with a clear "unavailable" error (this is the one
  case where it does not return the neutral "check your inbox" response),
- the home-page signup box is not rendered,
- the scheduler does nothing,
- the admin settings panel shows the reason ("Mail isn't configured in
  PocketBase" vs "Newsletter is switched off").

The frontend learns the state from a small public endpoint,
`GET /api/newsletter/config` → `{ "enabled": bool, "privacy_url": string|null }`
(`privacy_url` is `/privacy` when that page exists, else null).

### Signup flow

`POST /api/newsletter/subscribe` with `{ email, website }`.

- `website` is the honeypot (see Anti-abuse). If non-empty: return the normal
  success response and do nothing.
- Validate email format. Invalid → 400 with a friendly message.
- **Same response whether or not the address is already known** (no
  enumeration): "Check your inbox to confirm."
- New address: create `pending` record with fresh tokens, send confirmation
  email. The mail is sent **in the background** after the response, so response
  time does not reveal whether an address was already known. A unique-index
  race or a mail failure is logged and still answers with the neutral success.
- Existing `pending`: resend only if `confirm_sent_at` is older than the
  cooldown (24h); otherwise silently do nothing.
- Existing `confirmed`: do nothing (silently).

**Confirmation is a button, not a bare link.** Corporate link scanners fetch
every link in an email, so a state-changing GET would auto-confirm strangers.
The email links to the SPA page `/newsletter/confirm#token=…`; the page shows a
"Confirm my subscription" button which calls `POST /api/newsletter/confirm?token=…`
(valid and unexpired → `confirmed`, stamp `confirmed_at`; otherwise a friendly
"link expired, sign up again" state). The result is shown on the same page.

**Unsubscribe** works the same way for the visible link: the email links to the
SPA page `/newsletter/unsubscribe#token=…`, whose button calls
`POST /api/newsletter/unsubscribe?token=…`, which **deletes the subscriber
record** (see Privacy) and is idempotent. The same POST URL is the RFC 8058
one-click target in `List-Unsubscribe: <https://…/api/newsletter/unsubscribe?token=…>`
with `List-Unsubscribe-Post: List-Unsubscribe=One-Click` (form-encoded body;
the handler must not require JSON). Operators need DKIM covering both headers
for providers to honour one-click (README note).

**Tokens go in the URL fragment** (`#token=`), never the query, on pages the
SPA serves, so the global Umami pageview never captures a per-person secret.

Unconfirmed `pending` records whose `confirm_sent_at` is older than 7 days are
deleted by a daily cron.

### Privacy and compliance

Gather will now collect personal data (email addresses), which it did not before.
Gather is self-hostable, so the product must work for any operator in any
region. **The product ships the mechanics; each operator supplies their own
privacy policy.** Nothing region-specific (GDPR, POPIA, CAN-SPAM) is hardcoded.
The first deployment (the product owner's) is subject to GDPR, which is a good
strict baseline: consent as the lawful basis, double opt-in, one-click
withdrawal, data minimisation. This is a design for sensible practice, not legal
advice.

- **Data minimised:** only email, token(s), and confirmation timestamps. No IP
  address or user agent is stored on the subscriber record.
- **Consent record:** `confirmed_at` (double opt-in) is the proof of consent.
  Withdrawing consent (unsubscribing) is one click, as easy as giving it.
- **Purpose statement at the point of collection:** the signup box says what
  they will get and links to the privacy policy ("Weekend picks by email. Unsubscribe
  any time. Privacy policy").
- **Privacy policy is per server:** an operator-managed page at `/privacy`, a
  record in the existing `pages` collection (served by the catch-all `/:slug`
  route, so no new frontend code). It is **not auto-created**. We ship a
  plain-language template in `docs/` for operators to adapt, covering: what
  is collected and why, that it is used only for the weekly picks email, who
  processes it (the configured mail provider; Umami for aggregate, non-personal
  usage stats), how long it is kept, the lawful basis (consent), the data subject's
  rights (access, rectification, erasure, withdrawal, complaint to a
  supervisory authority), the controller's identity and a contact address for
  requests, and a note on any processing outside the EU/EEA (mail provider,
  Umami host) for the operator to fill in accurately.
- **Rights requests:** access, correction and deletion requests are handled by
  the operator via the contact address using the PocketBase dashboard. No
  self-service export in v1.
- **Right to erasure:** unsubscribing **hard-deletes** the subscriber record
  (no suppression list). Pending records are purged after 7 days. Admins can
  also delete a subscriber in the PocketBase dashboard on request.
- **Conditional links:** the privacy policy link appears in the signup box, the
  confirmation email and the newsletter footer **only if a `pages` record with
  slug `privacy` exists**. Legally the points that matter are collection time
  (signup box and confirmation email); the footer link is good practice. No
  page means no dead link.
- **Operator footer text:** new optional `newsletter_footer` setting (plain
  text) appended to every email footer, so operators can add a contact or
  postal address where their region expects one.
- **No third-party sharing:** subscriber emails are never sent anywhere except
  the configured mail server.

### Anti-abuse (baseline)

- **Honeypot:** a text input named `website`, wrapped in an element positioned
  off-screen with CSS (not `display:none`, which some bots detect), `tabindex="-1"`,
  `autocomplete="off"`, `aria-hidden="true"`. Humans never fill it. Server
  fakes success when it is populated.
- **Rate limit:** add a guest rule for `/api/newsletter/subscribe` in the existing
  PocketBase rate-limit list in `main.go`, tight (starting point: 5 per minute
  per IP; builder to tune).
- **Double opt-in** with expiring tokens.
- **Per-address confirmation cooldown** (24h) so one address cannot be
  bombed with confirmation emails.
- Captcha/Turnstile is explicitly deferred; revisit if abuse shows up in logs.

### Send job

**Time zone:** the schedule is evaluated in `time.Local`, which Go sets from the
`TZ` environment variable (product owner's decision: use the existing `TZ`
envvar, no per-instance time zone setting). If `TZ` is unset it falls back to
the host zone. The admin UI should show the active zone next to the time picker.

A cron job ticking **every minute** (cron expression `* * * * *`) reads the
settings and acts only when `effective_enabled` and the current server-local
weekday and `HH:MM` equal the configured day and time. This is how a
configurable schedule works with PocketBase's static cron registration, and it
implements decision 4 (no catch-up): if the minute is missed, that week is
skipped.

At the matching minute:

1. Find eligible picks: `hidden = false` AND `newsletter_sent_at` empty AND
   `start_date` falls in the relevant Fri–Sun window (if today is Mon–Thu, the
   coming Fri–Sun; if Fri–Sun, the current weekend; Sunday-dated posts are
   included, which is slightly more lenient than the home page's string compare)
   AND the post has at least one event that is `published` and still upcoming
   per the site's existing `seo.EventLifecycle` (which handles recurring series,
   exceptions and the default 3h event length). If none: log at debug level,
   stop. If several: take the one with the earliest `start_date` only. Note
   `start_date` is auto-set from the earliest event, so a post containing a
   Thursday event is dated Thursday and is not sent (same as the home page).
2. **Stamp `newsletter_sent_at` before sending** (at-most-once), using an atomic
   compare-and-set (`UPDATE … WHERE id=? AND newsletter_sent_at=''`, proceed
   only if one row changed) that does not bump `updated` (which would alter
   sitemap lastmod and JSON-LD). A crash mid-send can under-deliver but can
   never double-send.
3. Load all `confirmed` subscribers and send one email each, sequentially with a
   small delay between sends to be gentle on the SMTP server. A failed send is
   logged with the subscriber id and skipped; the job continues.
4. Save `newsletter_sent_count` (updated periodically during the loop and at the
   end, so a restart leaves an accurate record).

The tick passes `time.Now()` plus 30 seconds, truncated to the minute, to the
matcher, so a tick landing at 08:59:59.9 still counts as 09:00. Sending refuses
to run if the public base URL is the localhost fallback, since every link
would be wrong.

### Email content

Multipart (HTML + plain text), HTML built with table layout and inline CSS for
email-client compatibility, readable in dark mode.

- Header with site name (from settings).
- Picks title.
- **Full blurb** rendered from markdown (not just the intro paragraph).
- Event cards: thumbnail image (absolute URL), title, the **next occurrence**
  date/time for recurring events, place name, link. Times are rendered with
  the same formatter the server-rendered site uses, so the email matches the
  website. Keep the HTML well under Gmail's ~102KB clipping limit.
- "View on the web" link to `/picks/<slug>`.
- Footer: why you got this, unsubscribe link, privacy policy link (only if a
  `privacy` page exists), operator's optional footer text.
- Sent via PocketBase's mailer, with `List-Unsubscribe`, `List-Unsubscribe-Post`
  and `Precedence: bulk` headers.

### Tracking (decision 7)

- **UTM links:** every link to Gather in the email carries
  `utm_source=newsletter&utm_medium=email&utm_campaign=<picks slug>`. This is the
  reliable signal (clicks, and visits attributed in Umami).
- **Open pixel:** a 1×1 GIF at `GET /newsletter/open/<picks-id>.gif`. The handler
  responds with the GIF (`Cache-Control: no-store`) and, in the background,
  posts a `newsletter-open` event to `<umami_host_url>/api/send` (falling back
  to the origin of `umami_src`, as the tracker does) using `umami_website_id`,
  with the picks slug as event data. The forward sends a browser-like
  `User-Agent` (Umami drops bot UAs), accepts only http/https, refuses
  redirects, and is capped by a small concurrency limit. The handler validates
  the id format, forwards only for a post that exists and has been sent, and
  does the lookup before returning. If Umami settings are missing it just
  serves the GIF.
- The global `Cross-Origin-Resource-Policy: same-origin` header in `main.go`
  is relaxed to `cross-origin` for `/api/files/` and `/newsletter/open/` so
  webmail clients that load images directly can display them.
- The pixel is **per campaign, not per subscriber**: no per-person open
  tracking, so no personal data goes to Umami.
- Opens are inflated by Apple Mail Privacy Protection and Gmail image proxying;
  they are a rough indicator only. The admin UI should not present them as exact.

### Admin

In the existing settings form (`frontend/src/components/SettingsForm.tsx`, shown
in the Admin settings tab): the
enabled toggle, day and time pickers, an explanatory status line (see Master
switch), counts of confirmed / pending subscribers, and a **"Send test to me"**
button that sends the latest picks post to the logged-in admin's address only
(builder's call whether to include it; it makes testing much less painful).

### Frontend

- `NewsletterSignup` component in the home sidebar, in the section directly
  under the "Browse by date" calendar, using the existing `sidebar-section` /
  `sidebar-section-title` pattern and existing input/button styles. The sidebar
  is `display:none` below 900px, so the same component is also rendered at the
  end of the home page's main column in a mobile-only wrapper. Title along
  the lines of "Get the weekend picks by email"; one email field, one button,
  inline success/error message.
- Honeypot field per Anti-abuse.
- Short purpose line plus privacy policy link per Privacy and compliance.
- Rendered only when `/api/newsletter/config` says enabled.
- SPA pages: `/newsletter/confirm` (button, then result) and
  `/newsletter/unsubscribe` (button, then result). Tokens are read from the URL
  fragment.
- Add `data-umami-event` attributes consistent with how the sidebar already
  does it (e.g. `newsletter-signup`).

## Left to the builder

- Exact styling and copy, matching the existing site, including drafting the
  privacy policy template.
- Exact rate-limit numbers, send throttle delay, token length and format.
- Whether to include "Send test to me".
- Email HTML implementation details.

## Calls made without the product owner (marked for review)

- Every-minute tick with exact-minute match (30s tolerance for tick jitter), so
  a restart or outage at the scheduled minute skips the week.
- Event times inside the email use the site's server-side formatter (currently
  Europe/London), not `TZ`; `TZ` only decides *when* the send fires. Docker
  defaults to `TZ=UTC`, so operators must set it for the schedule to be local.
- At-most-once sending (stamp before send, no retry of failed recipients).
- If several picks are eligible, only the earliest is sent.
- A post with no upcoming events is not eligible.
- Pixel is per campaign, not per subscriber.
- 24h confirmation cooldown, 48h confirm-token expiry, 7-day purge of unconfirmed.
- Unsubscribing hard-deletes the record, so re-subscribing is just a new signup
  with a fresh confirmation.
- The newsletter has its own `newsletter_sent_at` flag, independent of
  `ap_announced`.
