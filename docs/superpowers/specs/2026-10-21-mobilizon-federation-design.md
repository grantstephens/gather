# Mobilizon federation — Design Spec

Date: 2026-10-21
Repo: `gather`. Status: draft, nothing built.

## Goal

Events published on a Gather instance show up on Mobilizon instances, so Mobilizon users can follow
Gather and see its events. Existing Mastodon-style federation (Notes) must keep working unchanged.

## Current state

`internal/activitypub/` already has: a single instance-level actor (`type: Service`, `/ap/actor`),
WebFinger, a signed inbox (Follow/Undo only, replies Accept), an outbox, a delivery queue with worker,
and HTTP signature signing/verification. Events are sent as `Create`/`Update`/`Delete` wrapping a
**`Note`** (`eventToNote` in `outbox.go`). Mobilizon only ingests objects of type `Event`, so today
Mobilizon can follow the actor but gets nothing useful.

## Research findings (Mobilizon source + docs.mobilizon.org, checked 2026-10-21)

- Mobilizon's actor converter accepts `Application`, `Group`, `Organization`, `Person`, `Service`.
  The existing `Service` actor is **not** rejected at the actor level. Required in practice: `id`
  (crash if missing/malformed), `preferredUsername`, `inbox`, `outbox`, `followers`, `publicKey.publicKeyPem`.
- The event type module does no check on actor type, `attributedTo`, or follow status. Any
  authorisation happens in the caller, which I did not read. **Unverified: whether a non-Group,
  non-Person organiser is accepted end to end.** Phase 0 settles this empirically.
- Incoming `Event` fields: `id` (required), `name`, `startTime` (effectively required), `endTime`,
  `timezone` (default `Etc/UTC`), `location`, `tag`, `to`/`cc` (public vs unlisted), `category`,
  `joinMode` (default `free`), `status` (falls back to `ical:status`, then `CONFIRMED`),
  `attachment` (images; a `Link` named `Website` becomes the online address), `inLanguage`,
  `externalParticipationUrl`, `isOnline`, `draft`, `published`/`updated`, `content`.
- `endTime` equal to 23:59:59 in the event's timezone is treated as "no end".
- `location` is a `Place` with `name`, top-level `latitude`/`longitude`, and `address` as a
  `PostalAddress` (`streetAddress`, `postalCode`, `addressLocality`, `addressRegion`, `addressCountry`).
  **Coordinates are only read if both are floats**; integers are silently dropped. `geo` is ignored.
  A string `address` is accepted as the street.
- `joinMode: external` requires `externalParticipationUrl`.
- Signatures are draft-cavage HTTP Signatures (draft 12), same family as the current code. No LD signatures.
- Mobilizon has an internal `relay@instance` actor; instances subscribe to each other and the relay
  `Announce`s content. Whether the relay follows non-Mobilizon instances is not confirmed.
- Context: **`mz` = `https://mobilizon.org/ns#`** (the docs page says `joinmobilizon.org`, which is wrong;
  caught by capturing a real Event, saved as `internal/activitypub/testdata/mobilizon_event.json`).
  The converter does not interpret `mz:` terms itself; they're for JSON-LD correctness.

Sources: docs.mobilizon.org (Interoperability / ActivityPub), Mobilizon `lib/federation/activity_stream/converter/{event,address,actor}.ex`
and `lib/federation/activity_pub/types/events.ex` on framagit.

## Phase 0 results (2026-10-21, Mobilizon 5.2.2 vs a Gather dev instance, over two public tunnels)

Hand-built `Create`/`Update`/`Delete` activities were signed by Gather's existing delivery code and
POSTed to Mobilizon's shared inbox (`/inbox`).

| Question | Result |
|---|---|
| Does Mobilizon accept our HTTP signatures? | **Yes.** "Got something with valid signature inside inbox". Actor was fetched and stored. |
| Is the existing `Service` actor accepted as organiser? | **Yes.** Stored as actor type `Service`; event created with it as organiser. **No `Group` actor needed for ingestion.** |
| Event fields | `name`, `startTime`/`endTime`, `timezone`, `joinMode: external` + `externalParticipationUrl`, `status`, `tag`, and `location` (`Place` + `PostalAddress` + `latitude`/`longitude`) all landed correctly. Geometry stored as `POINT(-3.7344 56.7012)`. |
| Integer coordinates | **Dropped silently** (no geometry stored). Floats work. The float-marshalling test in Phase 1 is required. |
| `Update` with `status: CANCELLED` | **Works.** Event shows as `cancelled`. |
| `Delete` with a bare `Tombstone` | **Not applied.** Mobilizon re-fetches the object URL to confirm it's gone and failed with `content_not_json`, because Gather has no `/ap/events/{id}` route and the SPA fallback returned HTML. |
| `Delete` once `/ap/events/{id}` returns `410` | **Works.** Mobilizon logs "410 Gone ... object is really gone", records a tombstone, and removes the event. |
| WebP image attachment | **Works**, with one trap: the attachment **must have a `name`** (we used `"Banner"`) or Mobilizon crashes the whole `Create` with a `FunctionClauseError` and the event is not created at all. `type: Document` + `url` + `name` is enough; Mobilizon downloaded the file and stored it as `image/webp` (400x300) as the event picture. `mediaType` is ignored. |
| Anonymous visibility after a bare `Create` | Event is stored and visible to a logged-in Mobilizon user, but **anonymous visitors get "not found" and it is absent from Explore/search.** |
| Relay: admin adds Gather as an instance | **Works.** Mobilizon resolved WebFinger `relay@host`, fetched the actor (`type: Application`), sent a Follow, and the follow was approved (`followers.approved = t`) after our Accept. NodeInfo `application_actor` was not needed. |
| Relay: `Announce` of an Event URL | **Works.** Mobilizon dereferenced `/ap/events/{id}`, created the event, and wrote a `Share` row. The event then appears in **Explore (`events`), search (`searchEvents`), and anonymously by UUID.** |

Consequences, which change the design:

1. **The relay is required, not optional.** Mobilizon's Explore/search only list remote events that have a
   `Share` row from an instance its relay actor follows (`filter_local_or_from_followed_instances_events`).
   A bare `Create` is stored but not discoverable. Admins follow an instance via a bare host: Mobilizon
   tries NodeInfo's application-actor URL first, then WebFinger `relay@host`, and the target must be a
   `Group` or instance-type actor (a `Person` is rejected). Gather therefore needs a relay actor
   (`relay@host`, type `Application`) that accepts the Follow and `Announce`s each event to its followers.
   **Verified end to end in the spike:** WebFinger `relay@host` -> `Application` actor -> Follow ->
   Accept -> `Announce` (object = the event's `/ap/events/{id}` URL) -> event listed in Explore/search.
   The spike reused the instance key for the relay actor (publicKey id `/ap/relay#main-key`, owner
   `/ap/relay`), skipped inbox signature verification, and kept followers in memory.
   Promote this from "out of scope" to a new **Phase 1b** before Phase 2. It needs: a persisted relay
   follower list, signature verification on `/ap/relay/inbox`, `Undo Follow` handling, and an
   `Announce` for each Create/Update (and a `Delete` to the same followers).
   Delivery model: **Mobilizon instances that follow the relay get `Announce`; per-user follows
   are not needed.** This also makes Decision 1 (per-follower `Event` vs `Note`) simpler: relay
   followers always get the Event shape.
2. **`/ap/events/{id}` must exist now.** Return the `Event` JSON for live events and `410` with a
   `Tombstone` for deleted ones. Mobilizon dereferences it on `Delete`. Move this from Phase 4 to Phase 1.
3. Decision 2 (add a `Group` actor if the `Service` is rejected) is **resolved: not needed for ingestion.**
   Phase 4 subset actors remain `Group` because users follow Groups.

Spike code (not for merge): `POST /api/ap/test-delivery` temporarily accepts `object` and `type`
overrides on branch `worktree-mobilizon-phase0`.

## Source of truth

There is no formal Mobilizon federation spec (the docs page is a dev note; the real behaviour is in
the Elixir converters). **A real Event captured from a Mobilizon instance
(`GET` the event URL with `Accept: application/activity+json`) is the source of truth.** If it
disagrees with the field table below, the capture wins and this spec gets corrected.

## Decisions

1. **Per-follower object shape.** Sending both a Note and an Event per event would double-post on
   Mastodon, and Mastodon renders `Event` objects poorly (usually a bare link), so Notes stay for the
   general fediverse. Each `ap_followers` row records a `software` hint
   (from NodeInfo of the follower's host, fetched once on Follow). Mobilizon followers get `Event`;
   everyone else gets `Note`. Unknown software defaults to `Note`.
2. **Keep the single `Service` actor** unless Phase 0 shows Mobilizon rejects it as an event organiser.
   Fallback: add a second actor of type `Group` (`/ap/group`) that owns `attributedTo` and mirrors the
   main actor's followers. Do not build the Group up front.
3. **Don't build the relay/`Application` actor** in v1. Mobilizon users follow Gather's actor directly
   (`@name@gather.host`). Revisit if instance-level subscription is wanted.
4. **Object IDs.** The Event keeps the same ID as the current Note (`{base}/ap/events/{id}`) so
   existing Update/Delete history stays coherent. A follower never receives both shapes for one event.

5. **Timezone is the `TZ` env var** (IANA name, default `Europe/London`; invalid values fall back to the default). Not a settings field.
   Needs a migration and an admin UI field.
6. **Subset actors: places and towns only** in Phase 4. Tags get no actors for now.
7. **No backfill.** A new follower only receives events published after they follow. (Interpreted
   from "past events only" — if the intent was to backfill *upcoming* events, change this and add a
   bounded send-on-Follow step.)
8. **WebP images:** try WebP attachments as-is; Phase 0 confirms Mobilizon renders them. Fallback is
   a JPEG/PNG rendition for Mobilizon followers.
9. **Descriptions for Mobilizon:** strip the scraper-appended `Cost:` and `Source:` lines from
   `content` (match trailing lines starting with those prefixes, case-insensitive), then append a
   link to the Gather event page (`<p><a href="{eventURL}">{eventURL}</a></p>`). The Note keeps its
   current content unchanged.
10. **Discovery** is by word of mouth and the README; no directory listing. Accepted.

## Design

### 1. `Event` serializer — `internal/activitypub/event.go` (new)

`eventToAPEvent(event, place, tags, baseURL) APEvent`, parallel to `eventToNote`. Fields:

| AP field | Source |
|---|---|
| `@context` | ActivityStreams plus the term definitions Mobilizon itself uses for the properties we emit (`eventContext` in `event.go`, copied from the captured Event; `mz` = `https://mobilizon.org/ns#`) |
| `type` | `Event` |
| `id`, `url` | `{base}/ap/events/{id}`, public event URL |
| `attributedTo` | actor ID |
| `to` / `cc` | Public / followers (same as Note) |
| `name` | title (plain text, **not** HTML-escaped) |
| `content` | description as HTML (reuse the Note's sanitiser/markdown render) |
| `mediaType` | `text/html` |
| `startTime` / `endTime` | RFC 3339 with offset. If no end, **omit** `endTime`. |
| `timezone` | IANA zone from the `TZ` env var (Gather is single-region) |
| `published` / `updated` | `created` / `updated` |
| `status` | `CONFIRMED`, or `CANCELLED` when `status = cancelled` |
| `joinMode` | `external` + `externalParticipationUrl` = event page URL. Gather has no RSVP, so never `free`. |
| `isOnline` | false |
| `inLanguage` | `en` |
| `tag` | `Hashtag` objects from approved tags (names ≤ 40 chars, Mobilizon drops longer) |
| `attachment` | image as `{type: Document, mediaType, url}` like the Note |
| `location` | see below |

`location` (only when the event has an approved place):

```json
{ "type": "Place", "id": "{base}/ap/places/{id}", "name": "…",
  "latitude": 56.39, "longitude": -3.43,
  "address": { "type": "PostalAddress", "streetAddress": "…", "addressLocality": "…",
               "postalCode": "…", "addressRegion": "…", "addressCountry": "GB" } }
```

- `latitude`/`longitude` must be Go `float64` and marshal with a decimal point. Guard against whole
  numbers serialising as `56` (use a custom marshaller or assert non-integral in a test).
- Places store an OSM `GeoPoint` (`location` field) plus a free-text `address`. Put the free text in
  `streetAddress`; don't try to parse it. Leave other address parts out rather than guessing.
- Recurring events: each occurrence is already its own record, so each is its own `Event`. Put the
  `recurrence.HumanLabel` text at the end of `content`, as the Note does.

### 2. Per-follower shape selection

- Migration: add `software` (text, optional) and `software_checked` (date) to `ap_followers`.
- On Follow, after Accept, fetch `https://{host}/.well-known/nodeinfo` → `software.name` in a goroutine
  with a 5s timeout; store lowercase name. Failure leaves it empty.
- Delivery builds the activity per recipient inbox: `software == "mobilizon"` → Event, else Note.
  Shared-inbox dedupe in `delivery.go` must key on (inbox, shape), not inbox alone.
- Admin can override a follower's shape in the admin UI (stretch; skip in v1).

### 3. Update and Delete

- `Update`: same Event ID, full object, `updated` bumped. Already generated by the existing hooks; swap the object builder.
- Cancelled events: send `Update` with `status: CANCELLED` (not Delete), so Mobilizon shows "cancelled".
- Deleted/unpublished events: `Delete` with a `Tombstone` for the event ID, as today.
- Events that become past: nothing sent.

### 4. Outbox and actor

- Outbox collection stays on Notes (it's what crawlers and Mastodon read). Optionally add
  `GET /ap/events/{id}` content negotiation: `Accept: application/activity+json` returns the **Event**
  when the requester's UA/host is a known Mobilizon follower, else the Note. Mobilizon may dereference
  the object ID when handling Create, so this endpoint must return the Event shape for them.
  If that's flaky, serve the Event shape to all AP clients on `/ap/events/{id}` and keep Notes only
  inside outbound Create payloads.
- Actor doc: add `"mz"` to the context and `endpoints.sharedInbox` only if not already present. No type change.

### 5. Inbound

Mobilizon sends `Follow` (handled), `Undo Follow` (handled), and may send `Join`/`Leave` on events.
Ignore those (current default branch logs and drops). No change needed.

## Phases

**Phase 0 — prove it (half a day, blocks everything).** Run a local Mobilizon (Docker, from the
upstream compose). Hand-craft one Event JSON, sign a `Create` with the existing delivery code via
`POST /api/ap/test-delivery`, from a Gather dev instance reachable over a tunnel. Confirm the event
appears on the Mobilizon follower's feed. Answers the open question about the `Service` actor. If
rejected, add the `Group` actor (Decision 2) before Phase 1.

**Phase 1 — serializer + tests.** `event.go`, golden-file test against a captured Mobilizon-produced
Event (export one from the local instance to compare shape), float-coordinate test, no-place test,
cancelled test, no-end-time test.

**Phase 1b — relay actor (done, 2026-10-21).** `relay.go`: `relay@host` (type `Application`), followers in
`ap_relay_followers`, inbox verifies HTTP signatures, Follow → persist + Accept, Undo Follow → remove.
`QueueRelayEvent` sends Create (+ Announce 10s later), Update or Delete to relay followers, deduped by
inbox. Hook rules: published→cancelled is an `Update` (`/ap/events/{id}` still serves cancelled events, so
a Delete would be ignored); anything else leaving published, and deleting or un-cancelling-to-draft a
cancelled event, is a `Delete`; cancelled→published is an `Update`. Reuses the instance keypair rather
than a separate relay key (the relay actor publishes the same public key under its own `keyId`).
Verified live: follow → Accept → listed in public search → rename → cancel → delete, and follower
persistence across a Gather restart. Live testing found two bugs in the *existing* inbox verifier, both
fixed with tests: (1) Mobilizon publishes PKCS#1 public keys (`BEGIN RSA PUBLIC KEY`), which
`fetchPublicKey` rejected; (2) the `host` header was read from `r.Header`, where Go never puts it, so
every signature covering `host` failed. Neither existing Follow path could have verified a real remote.

**Phase 2 — per-follower shape.** Migration, NodeInfo lookup, delivery split, dedupe fix.

**Phase 3 — real-world test.** Follow from a public Mobilizon instance, publish, edit, cancel, delete.
Add notes to the README on how Mobilizon users find and follow the instance.

**Phase 4 — subset actors (places, towns, tags), created on demand.** Lets people follow
`@venue@gather.host` or `@town@gather.host` instead of the whole instance. Starts only after Phases
0-3 work. Actor type is `Group` (Mobilizon's native shape); if Phase 0 forced a `Group` actor already,
reuse that code.

- **Lazy creation.** Nothing is pre-created. WebFinger (`acct:{slug}@host`) looks the slug up against
  *approved* places, towns and tags; found → return the actor link, not found → 404. The first
  `GET /ap/actor/{kind}/{id}` generates and stores the RSA keypair (unique constraint or mutex so
  concurrent first hits don't create two). The first Follow creates the follower row against that actor.
- **Stable identity.** Actor ID is `/ap/actor/{kind}/{recordId}` (kind = `place|town|tag`). The slug
  is a WebFinger alias only, so renaming never orphans followers.
- **Storage.** New `ap_actors` collection: `kind`, `ref_id`, `public_key`, `private_key`. Add an
  `actor` relation to `ap_followers` (empty = the instance actor, so existing rows stay valid).
- **Routing.** `/ap/actor/{kind}/{id}` serves actor, inbox, outbox, followers. The inbox handler and
  signer take the target actor rather than assuming the instance actor. Signature `keyId` is the
  subset actor's own key.
- **Fan-out.** At publish time compute the event's actor set (instance, its place, its town, its
  approved tags), union their followers, dedupe by (inbox, shape) as in Phase 2, send one copy each.
  Only actors with followers cost anything.
- **Moderation.** Pending places/tags never resolve. If a place or tag is deleted or un-approved:
  serve 410 and send `Delete` to its followers.
- **Abuse.** Rate-limit WebFinger and actor routes; only existing approved entities resolve, so
  nothing can be minted by guessing names.
- **Tests.** WebFinger found/not-found/pending; lazy keygen race; follow a place and receive only its
  events; event with place+town followers in common is delivered once; rename keeps the actor ID;
  delete sends Tombstone/410.

## Testing

- Unit: serializer output vs golden JSON; float coordinate marshalling; timezone/offset handling around BST changes.
- Unit: delivery picks the right shape per follower, no duplicates for shared inboxes.
- Integration: local Mobilizon container follows Gather, receives Create/Update/Cancel/Delete.
- Regression: existing `activitypub_test.go` Note tests untouched and passing.

## Risks and unknowns

- **Organiser type** (Service vs Group) — settled in Phase 0.
- **Object dereferencing** — Mobilizon may refetch `/ap/events/{id}`; the endpoint must return an `Event` to it.
- **Mobilizon version drift** — the converter code was read from `main`; the version on target instances may differ. Test against a released image, not just `main`.
- **Mastodon double-post** — avoided by per-follower shape; verify no follower gets both.
- **Timezone** — if Gather stores UTC without a configured zone, `startTime` offsets will look wrong in Mobilizon. Uses the `TZ` env var (default `Europe/London`).
- **Moderation** — Mobilizon admins can block a domain; nothing to do on our side.

## Out of scope

Accepting `Join` (RSVP), Mobilizon groups/posts/resources, importing events
*from* Mobilizon, pre-created actors for every place/tag, `mz:` extras like capacity or comment moderation.
