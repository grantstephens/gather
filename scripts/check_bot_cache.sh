#!/usr/bin/env bash
# Usage: scripts/check_bot_cache.sh https://perthshire.events [/extra/path ...]
# Warms each URL with a browser UA, then fetches it as Googlebot. The bot must
# get server-rendered HTML (canonical present, no modulepreload shell) and the
# response must carry Vary: User-Agent.
set -u
BASE="${1:?base url}"
B='Mozilla/5.0 (X11; Linux x86_64) Chrome/120 Safari/537.36'
G='Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)'
fail=0
for path in / /picks "${@:2}"; do
  curl -s -o /dev/null -A "$B" "$BASE$path"; curl -s -o /dev/null -A "$B" "$BASE$path"
  body=$(curl -s -A "$G" "$BASE$path")
  vary=$(curl -sI -A "$G" "$BASE$path" | tr -d '\r' | grep -i '^vary:' | tr '\n' ' ' || true)
  if echo "$body" | grep -q 'rel="canonical"' && ! echo "$body" | grep -q modulepreload; then ok=OK; else ok=SHELL; fail=1; fi
  echo "$vary" | grep -qi 'user-agent' || { ok="$ok+NOVARY"; fail=1; }
  echo "$ok  $path  [$vary]"
done
exit $fail
