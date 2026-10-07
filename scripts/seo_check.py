#!/usr/bin/env python3
"""Acceptance crawler for Gather's SEO output (stdlib only).

    scripts/seo_check.py https://perthshire.events            # whole sitemap
    scripts/seo_check.py http://127.0.0.1:8090 --limit 40     # quick run

Fetches the sitemap and every URL in it as Googlebot, sequentially with a small
delay (never hammer a live site), and reports:

  * non-200 URLs in the sitemap
  * double-encoded text in titles/descriptions ("&amp;#039;", "&amp;amp;")
  * duplicate titles, missing descriptions
  * event pages without a canonical or parsable JSON-LD, with past start dates
    (retired events must not be in the sitemap), or missing `offers` when the
    page shows a cost
  * internal links on event pages that do not resolve
  * the homepage: Vary: User-Agent and server-rendered (not the SPA shell)
  * /place/<record id> answering 301 to the slug

Exit status is non-zero if anything fails.
"""
import argparse
import collections
import datetime as dt
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET

BOT = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
# html.EscapeString legitimately writes " as &#34; and ' as &#39;. Anything else
# escaped twice is a bug.
DOUBLE_ENCODED = re.compile(r"&amp;(#\d+|#x[0-9a-fA-F]+|amp;|quot;|lt;|gt;|nbsp;)")
RETIRE_AFTER = dt.timedelta(days=7)


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *a, **k):
        return None


def fetch(url, follow=True):
    req = urllib.request.Request(url, headers={"User-Agent": BOT})
    opener = urllib.request.build_opener() if follow else urllib.request.build_opener(NoRedirect)
    try:
        with opener.open(req, timeout=30) as r:
            return r.status, r.headers, r.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        return e.code, e.headers, e.read().decode("utf-8", "replace")


def tag(body, pattern):
    m = re.search(pattern, body, re.S)
    return m.group(1) if m else None


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("base")
    ap.add_argument("--limit", type=int, default=0, help="only check the first N sitemap URLs")
    ap.add_argument("--delay", type=float, default=0.3)
    args = ap.parse_args()
    base = args.base.rstrip("/")

    failures = collections.defaultdict(list)

    def fail(kind, detail):
        failures[kind].append(detail)

    # ---- homepage
    status, headers, body = fetch(base + "/")
    # a response may carry several Vary lines; read them all
    vary = ",".join(headers.get_all("Vary") or [])
    if "user-agent" not in vary.lower():
        fail("homepage missing Vary: User-Agent", base + "/")
    if 'rel="canonical"' not in body or "modulepreload" in body:
        fail("homepage served as SPA shell to Googlebot", base + "/")

    # ---- sitemap
    status, _, xml = fetch(base + "/sitemap.xml")
    if status != 200:
        print(f"sitemap.xml returned {status}")
        return 1
    ns = {"s": "http://www.sitemaps.org/schemas/sitemap/0.9"}
    urls = [e.text for e in ET.fromstring(xml).findall(".//s:loc", ns)]
    if args.limit:
        urls = urls[: args.limit]
    print(f"checking {len(urls)} sitemap URLs on {base} ...")

    titles = collections.Counter()
    internal_links = collections.defaultdict(set)
    now = dt.datetime.now(dt.timezone.utc)
    stale = 0

    for i, url in enumerate(urls, 1):
        time.sleep(args.delay)
        status, headers, body = fetch(url)
        if status != 200:
            fail("non-200 in sitemap", f"{status} {url}")
            continue
        title = tag(body, r"<title>(.*?)</title>") or ""
        desc = tag(body, r'<meta name="description" content="(.*?)"')
        titles[title] += 1
        for label, text in (("title", title), ("description", desc or "")):
            if DOUBLE_ENCODED.search(text):
                fail("double-encoded " + label, f"{url}  {text[:80]}")
        if desc is None or not desc.strip():
            fail("missing meta description", url)
        if "noindex" in ",".join(headers.get_all("X-Robots-Tag") or []).lower():
            fail("noindex page listed in sitemap", url)

        if "/event/" in url:
            if 'rel="canonical"' not in body:
                fail("event without canonical", url)
            ld = tag(body, r'<script type="application/ld\+json">(.*?)</script>')
            data = None
            try:
                data = json.loads(ld) if ld else None
            except ValueError:
                pass
            if not isinstance(data, dict):
                fail("event without parsable JSON-LD", url)
            else:
                try:
                    end = dt.datetime.fromisoformat((data.get("endDate") or data["startDate"]).replace("Z", "+00:00"))
                    if end.tzinfo is None:
                        end = end.replace(tzinfo=dt.timezone.utc)
                    if now - end > RETIRE_AFTER:
                        stale += 1
                        fail("retired event listed in sitemap", f"{url} ended {end.date()}")
                except (KeyError, ValueError):
                    fail("event JSON-LD without a usable date", url)
                if re.search(r"Cost: [^<]*(£\d|[Ff]ree)", body) and "offers" not in data:
                    fail("cost shown but no offers in JSON-LD", url)
            for href in re.findall(r'<a href="([^"]+)"', body):
                if href.startswith(base) and href != url:
                    internal_links[href].add(url)
        if i % 25 == 0:
            print(f"  {i}/{len(urls)}")

    for t, n in titles.items():
        if n > 1 and t:
            fail("duplicate title", f"{n}x {t[:90]}")

    # ---- internal links from event pages must resolve (no dead venue/tag links)
    for href in sorted(internal_links)[:200]:
        time.sleep(args.delay)
        status, _, _ = fetch(href)
        if status != 200:
            fail("event page links to a non-200 page", f"{status} {href}  (from {sorted(internal_links[href])[0]})")

    # ---- /place/<record id> must 301 to the slug
    pstatus, _, pbody = fetch(base + "/api/collections/places/records?perPage=1&filter=slug!=''")
    if pstatus == 200:
        try:
            items = json.loads(pbody).get("items", [])
        except ValueError:
            items = []
        if items:
            p = items[0]
            st, hdrs, _ = fetch(f"{base}/place/{p['id']}", follow=False)
            loc = hdrs.get("Location", "") or ""
            if st != 301 or not loc.endswith("/place/" + p["slug"]):
                fail("place id URL does not 301 to its slug", f"{st} {loc}")
    else:
        print("(skipping place redirect check: places API not readable)")

    # ---- report
    print()
    if not failures:
        print("OK - no problems found")
        return 0
    total = 0
    for kind, details in sorted(failures.items()):
        total += len(details)
        print(f"FAIL  {kind}: {len(details)}")
        for d in details[:5]:
            print(f"        {d}")
        if len(details) > 5:
            print(f"        ... and {len(details) - 5} more")
    print(f"\n{total} problem(s) in {len(failures)} categories")
    return 1


if __name__ == "__main__":
    sys.exit(main())
