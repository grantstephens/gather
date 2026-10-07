/**
 * placeHref is the URL of a place's page: its readable slug when it has one,
 * else the record id (the older URL form, which the server redirects to the slug).
 */
export function placeHref(place: { id: string; slug?: string }): string {
  return '/place/' + encodeURIComponent(place.slug || place.id)
}
