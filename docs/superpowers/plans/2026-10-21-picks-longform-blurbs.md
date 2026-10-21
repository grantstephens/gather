# Picks Long-Form Blurbs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make ~300-450 word structured Markdown picks blurbs render properly on the SPA, in crawler HTML, in meta descriptions (client and server identical), and in ActivityPub Notes.

**Architecture:** Fix `stripMarkdown` first (paragraph-aware, shared by SEO and AP), then render blurb to sanitised HTML server-side, then mirror the description logic client-side, then trim the AP Note to the intro, then CSS. Announcement is cron-only (`internal/cron/cron.go:55`, gated on `ap_announced = false`), so edits never re-announce. No change needed there; a test pins it.

**Tech Stack:** Go (PocketBase), `github.com/yuin/goldmark` + `github.com/microcosm-cc/bluemonday` (new deps, neither is in go.mod; only `golang.org/x/net` is, indirect), Preact, `marked`, DOMPurify.

**Spec:** `bug.md`

## Global Constraints

- Meta description: first 160 chars of plain text (`truncateText(..., 160)`).
- Sanitiser allowlist: `h3, p, a, strong, em, ul, li` (plus `ol`, `br`, `h2`, `h4` harmless to allow).
- Blurb links are relative (`/event/slug`); bot HTML must emit them absolute using `baseURL`.
- AP Note: title + plain intro (first paragraph only, <= ~500 chars, cut at sentence/word boundary) + "Read more" link. No relative links.
- Existing "This weekend's picks" `<ul>` stays.

## Review Focus

- Blurb with only one paragraph, no headings: description and AP intro unchanged from today's behaviour.
- Blurb containing raw `<script>`/`onerror=` HTML: must be stripped from bot HTML.
- Link with `javascript:` href or protocol-relative `//evil.com`: must not become an absolute link on our origin or survive as `javascript:`.
- First paragraph > 500 chars with no sentence end: cuts at a word, adds `…`, never mid-word or mid-entity.
- Empty/whitespace blurb: no empty `<p>`, no empty description meta.
- `stripMarkdown` also feeds events and picks index (`event_html.go`, `opengraph.go`, `jsonld.go:66`, `picks_index_html.go:40`): existing `jsonld_test.go` cases must still pass.

---

### Task 1: Paragraph-aware `stripMarkdown`

**Files:**
- Modify: `internal/seo/jsonld.go:166`
- Test: `internal/seo/jsonld_test.go`

**Interfaces:**
- Produces: `stripMarkdown(string) string` (same signature). New behaviour: block boundaries (blank line, heading, list item) become a single space in output rather than `\n`... but see step 3 for the compat decision.

- [ ] **Step 1: Failing table test** `TestStripMarkdown_LongPicksBlurb` using the sample blurb from `bug.md`. Assert exact output: `Birnam Book Festival turns five this weekend, with poetry and a pastry on Saturday and a Hogmanay ghost story on Sunday. Poetry Brunch, Birnam Arts, Birnam Saturday morning at Birnam Arts, ...` (heading text followed by a space, link text kept, `**Also on:**` markers gone, no `###`, no `(/event/...)`).
- [ ] **Step 2: Run** `go test ./internal/seo -run StripMarkdown -v`. Expected: FAIL (output contains `\n`).
- [ ] **Step 3: Change the `\n\n+` collapse to a single space, and `(?m)^#+\s+` removal to keep a leading space.** Check every existing case in `jsonld_test.go` and the event description callers; if any existing case expects `\n`, add a second function `stripMarkdownInline(string) string` (newlines -> spaces) used only for picks description, rather than changing event behaviour.
- [ ] **Step 4: Run** `go test ./internal/seo -v`. Expected: PASS.
- [ ] **Step 5: Commit** `fix: stripMarkdown keeps paragraph breaks as spaces`

### Task 2: Server-side blurb HTML in crawler page

**Files:**
- Modify: `go.mod`, `go.sum` (`go get github.com/yuin/goldmark github.com/microcosm-cc/bluemonday`)
- Create: `internal/seo/markdown.go`, `internal/seo/markdown_test.go`
- Modify: `internal/seo/picks_html.go:39,76-79,173-175`
- Test: `internal/seo/picks_html_test.go` (new)

**Interfaces:**
- Produces: `renderMarkdownHTML(md, baseURL string) string` in `markdown.go`: goldmark -> bluemonday (strict allowlist, `AllowRelativeURLs(false)`, `http/https/mailto` schemes only) with relative `/...` hrefs rewritten to `baseURL + href` *before* sanitising. Rejects `//host` and `javascript:` hrefs (drops the `href`).
- Consumes: `stripMarkdown` from Task 1.
- Changes `buildPicksHTML` param `fullBlurb string` into `blurbHTML string`; `GeneratePicksHTML` computes `description := truncateText(stripMarkdown(blurb), 160)` and `blurbHTML := renderMarkdownHTML(blurb, baseURL)`.

- [ ] **Step 1: Failing tests** in `markdown_test.go`: sample blurb -> output contains `<h3>Poetry Brunch, Birnam Arts, Birnam</h3>`, `<a href="https://example.com/event/poetry-brunch-l9lbpw21">`, `<strong>Also on:</strong>`; table rows for `<script>alert(1)</script>`, `[x](javascript:alert(1))`, `[x](//evil.com)`, `<img src=x onerror=...>` (all neutralised); empty string -> `""`.
- [ ] **Step 2: Failing test** in `picks_html_test.go` calling `buildPicksHTML` directly: body contains `<h3>`, two `<p>`, absolute links, still contains `This weekend's picks` `<ul>`; `<meta name="description">` is <= 163 chars with no `###`/`](`; og/twitter descriptions are plain text.
- [ ] **Step 3: Run** `go test ./internal/seo -run 'Markdown|Picks' -v`. Expected: FAIL (undefined).
- [ ] **Step 4: Implement** `renderMarkdownHTML`; goldmark without `html.WithUnsafe()`; rewrite links via a goldmark AST transformer or bluemonday `RewriteSrc`-style hook, whichever is smaller.
- [ ] **Step 5: Wire into `GeneratePicksHTML`/`buildPicksHTML`; write `blurbHTML` raw (already sanitised) instead of `<p>%s</p>`.**
- [ ] **Step 6: Run** `go test ./internal/seo -v`. Expected: PASS.
- [ ] **Step 7 (optional, only if cheap):** add `wordCount` and `articleBody` (plain text) to `articleLD`. Skip `ItemList` unless trivial.
- [ ] **Step 8: Commit** `feat: render picks blurb markdown to sanitised HTML for crawlers`

### Task 3: Client meta description matches server

**Files:**
- Modify: `frontend/src/pages/PicksPost.tsx:42`
- Create: `frontend/src/lib/plainText.ts` (`blurbToPlainText(md: string, maxLen = 160): string`)
- Test: whichever runner the frontend uses (check `frontend/package.json`; if there is none, add a `vitest` test or fall back to a documented manual check in Step 3)

**Interfaces:**
- Produces: `blurbToPlainText(md, maxLen)`: `marked.parse` -> insert a space at block boundaries (`</p>`, `</h1-6>`, `</li>`, `<br>`) -> strip tags -> collapse whitespace -> trim -> truncate at word boundary with `...` exactly like Go `truncateText` (`opengraph.go:77`).

- [ ] **Step 1: Failing test** with the sample blurb; expected string identical to the Go test's expected description (copy the literal across, comment that the two must stay in sync).
- [ ] **Step 2: Run, expect FAIL.** Implement. **Run, expect PASS.**
- [ ] **Step 3: Replace the inline expression at `PicksPost.tsx:42` with `blurbToPlainText(record.blurb)`.**
- [ ] **Step 4: Commit** `fix: client meta description spaces block boundaries`

### Task 4: ActivityPub Note carries intro only

**Files:**
- Modify: `internal/activitypub/picks.go:11-21`
- Test: `internal/activitypub/activitypub_test.go` (add), plus a cron test if `internal/cron` has a harness

**Interfaces:**
- Produces: `picksIntro(blurb string, maxLen int) string` (unexported, in `picks.go`): first paragraph (split on blank line), plain text, cut at last sentence end `. ! ?` before `maxLen`, else last space, else hard cut, append `…` when cut.
- `stripMarkdown` is unexported in `seo`. Export it as `seo.StripMarkdown` (rename + update callers) only if `activitypub` importing `seo` creates no import cycle (verify with `go build ./...`); otherwise move it to a tiny `internal/textutil` package and have both import it.

- [ ] **Step 1: Failing table tests** `TestPicksToNote_LongBlurb`: Note content contains title and `Read more` link to `baseURL/picks/<slug>`; does NOT contain `###`, `Swan Lake`, `/event/`, or `Also on`; length of the intro <= 500. Extra rows: single-sentence blurb (unchanged), 700-char unpunctuated paragraph (word cut + `…`), HTML-special chars (`<`, `&`) escaped in the output.
- [ ] **Step 2: Run** `go test ./internal/activitypub -run Picks -v`. Expected: FAIL.
- [ ] **Step 3: Implement `picksIntro`, use it in `picksToNote` with `html.EscapeString`.**
- [ ] **Step 4: Run** `go test ./internal/activitypub ./internal/seo -v`. Expected: PASS.
- [ ] **Step 5: Pin no-re-announce.** Already guaranteed by `cron.go:55` (`ap_announced = false`) and `:70` (set true after send); `worker.go` only drains `ap_delivery_queue`. Add a test only if cron has an existing harness; otherwise note it in the commit message. No code change.
- [ ] **Step 6: Commit** `fix: federate only the intro paragraph of picks blurbs`

### Task 5: Blurb CSS

**Files:**
- Modify: `frontend/src/pages/PicksPost.css:20-27`

- [ ] **Step 1: Replace `.picks-post-blurb p { margin: 0; }`** with: `p { margin: 0 0 var(--space-4); }`, `p:last-child { margin-bottom: 0; }`, `h3 { font-size: var(--text-xl); font-weight: var(--font-bold); color: var(--color-text); margin: var(--space-8) 0 var(--space-2); line-height: 1.3; }`, `a { color: var(--color-accent); text-decoration: underline; text-underline-offset: 2px; }`, `strong { color: var(--color-text); font-weight: var(--font-semibold); }`. Check these CSS variable names exist in the theme file first and substitute the real ones.
- [ ] **Step 2: Drop `.picks-post-blurb` font size from `--text-lg` to `--text-base`** (long copy; lead paragraph can keep `--text-lg` via `p:first-child`). Add `margin-bottom: var(--space-8)` to `.picks-post-blurb` so it is separated from `.picks-post-events`.
- [ ] **Step 3: Visual check** with `make dev`: seed a picks row using the sample blurb, check `/picks/:slug` in light, dark, and a 375px viewport. Heading, links and bold must all be legible in both themes.
- [ ] **Step 4: Commit** `style: paragraph, heading and link styles for long picks blurbs`

### Task 6: End-to-end verify

- [ ] `go test ./...` and `cd frontend && npm run build`: both clean.
- [ ] `curl -A Googlebot http://127.0.0.1:8090/picks/<slug>` shows real `<h3>`, `<p>`, absolute `<a href>`, clean 160-char description.

---

## Separate: the timezone question (answer, no code in this repo)

The app expects **true UTC instants**. Evidence: events are stored as `YYYY-MM-DD HH:MM:SS.000Z` (parsed with the `Z` layout in `home_shell.go:168`), the frontend does `new Date(event.start_datetime)` (`EventCard.tsx:13`, `PicksPost.tsx:56`) which converts to the viewer's local zone, and the server renders in `Europe/London`. So there is no "floating time" concept anywhere.

**Convention for the scrapers:** parse the page time as `Europe/London` local, convert to UTC, send that. The Birnam scraper storing 16:15 local as `16:15Z` is the bug; the theatre scraper is correct. During BST it shows an hour late (16:15Z renders as 17:15 BST). Fix belongs in the scraper repo; nothing to change here.

## Self-review

- Spec coverage: CSS (T5), SEO HTML + description (T1, T2), client description (T3), AP (T4), tests (all), tz answer (above). Optional ItemList/wordCount left as optional in T2 step 7.
- Open decision for the user: adding goldmark + bluemonday as new deps. If you'd rather avoid deps, the fallback is a hand-rolled tiny renderer, which I'd advise against for sanitisation.
