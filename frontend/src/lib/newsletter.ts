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
