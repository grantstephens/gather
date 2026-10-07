import { describe, expect, it } from 'vitest'
import { hasEnded } from './lifecycle'

const NOW = new Date('2026-10-06T12:00:00Z')
const iso = (hoursFromNow: number) => new Date(NOW.getTime() + hoursFromNow * 3600_000).toISOString()

describe('hasEnded', () => {
  it('is false for a future event', () => {
    expect(hasEnded(iso(24), undefined, NOW)).toBe(false)
  })
  it('is false while an event with no end time is within its default 3 hours', () => {
    expect(hasEnded(iso(-2), undefined, NOW)).toBe(false)
  })
  it('is true once the default 3 hours have passed', () => {
    expect(hasEnded(iso(-5), undefined, NOW)).toBe(true)
  })
  it('uses the end time when there is one', () => {
    expect(hasEnded(iso(-60 * 24), iso(10 * 24), NOW)).toBe(false) // exhibition still running
    expect(hasEnded(iso(-60 * 24), iso(-3 * 24), NOW)).toBe(true)
  })
  it('ignores an end time that is not after the start', () => {
    expect(hasEnded(iso(-5), iso(-6), NOW)).toBe(true)
    expect(hasEnded(iso(-2), iso(-3), NOW)).toBe(false)
  })
  it('accepts PocketBase date strings with a space separator', () => {
    expect(hasEnded('2026-10-01 18:00:00.000Z', undefined, NOW)).toBe(true)
    expect(hasEnded('2026-10-20 18:00:00.000Z', '2026-10-20 20:00:00.000Z', NOW)).toBe(false)
  })
  it('is false for an unreadable date rather than throwing', () => {
    expect(hasEnded('not a date', undefined, NOW)).toBe(false)
  })
})
