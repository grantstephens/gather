// Newsletter action links carry their token in the URL fragment (#token=...),
// never the query string: the global Umami pageview reports location.search, and
// these tokens are per-person secrets.
export function tokenFromHash(hash: string): string {
  if (!hash.startsWith('#')) return ''
  return new URLSearchParams(hash.slice(1)).get('token') ?? ''
}

export interface NewsletterConfig {
  enabled: boolean
  privacy_url: string | null
}

export interface NewsletterAdminInfo {
  enabled: boolean
  toggle_on: boolean
  mail_configured: boolean
  base_url_local: boolean
  confirmed: number
  pending: number
  timezone: string
  send_day: number
  send_time: string
}

export const DAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

// One plain-English line saying whether the newsletter will actually send, and
// if not, why. Missing mail config outranks the toggle: it forces the feature off.
export function statusLine(i: NewsletterAdminInfo): string {
  if (!i.mail_configured) return "Mail isn't configured in PocketBase, so the newsletter is off."
  if (i.base_url_local) return "The Application URL is still localhost, so emails would carry dead links and sending is off. Set it in PocketBase (Settings > Application) or via BASE_URL."
  if (!i.toggle_on) return 'Switched off.'
  if (!i.send_time) return 'No send time set, so nothing will be sent.'
  return `Sending every ${DAY_NAMES[i.send_day] ?? 'week'} at ${i.send_time} (${i.timezone}).`
}

/** How many events the mobile list shows before the signup card appears. */
export const MOBILE_SIGNUP_AFTER = 10

/**
 * Index of the day (in a list of per-day event counts) after which the mobile
 * signup card goes: the first day that takes the running total to `threshold`,
 * or the last day when there are fewer events than that. -1 when there are no days.
 */
export function signupSlotIndex(dayCounts: number[], threshold: number): number {
  let total = 0
  for (let i = 0; i < dayCounts.length; i++) {
    total += dayCounts[i]
    if (total >= threshold) return i
  }
  return dayCounts.length - 1
}
