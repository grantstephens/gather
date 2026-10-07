/**
 * hasEnded mirrors the server's rule (internal/seo/lifecycle.go): an event is
 * over at its end time, or 3 hours after it starts when it has none.
 */
export function hasEnded(start: string, end?: string, now: Date = new Date()): boolean {
  const s = new Date(start.replace(' ', 'T'))
  if (isNaN(s.getTime())) return false
  const e = end ? new Date(end.replace(' ', 'T')) : null
  const endedAt = e && !isNaN(e.getTime()) && e > s ? e : new Date(s.getTime() + 3 * 3600 * 1000)
  return now >= endedAt
}
