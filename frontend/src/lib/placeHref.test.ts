import { describe, expect, it } from 'vitest'
import { placeHref } from './placeHref'

describe('placeHref', () => {
  it('prefers the slug', () => {
    expect(placeHref({ id: 'z7k4c8qg71zto5f', slug: 'the-space' })).toBe('/place/the-space')
  })
  it('falls back to the record id when there is no slug yet', () => {
    expect(placeHref({ id: 'z7k4c8qg71zto5f' })).toBe('/place/z7k4c8qg71zto5f')
    expect(placeHref({ id: 'z7k4c8qg71zto5f', slug: '' })).toBe('/place/z7k4c8qg71zto5f')
  })
  it('encodes unusual characters so the path stays valid', () => {
    expect(placeHref({ id: 'x', slug: 'a b/c' })).toBe('/place/a%20b%2Fc')
  })
})
