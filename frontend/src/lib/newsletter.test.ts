import { describe, expect, it } from 'vitest'
import { signupSlotIndex, tokenFromHash } from './newsletter'

describe('tokenFromHash', () => {
  it('reads the token from the URL fragment', () => {
    expect(tokenFromHash('#token=abc123')).toBe('abc123')
  })
  it('ignores other fragment params', () => {
    expect(tokenFromHash('#x=1&token=abc&y=2')).toBe('abc')
  })
  it('returns an empty string when there is no token', () => {
    expect(tokenFromHash('')).toBe('')
    expect(tokenFromHash('#')).toBe('')
    expect(tokenFromHash('#other')).toBe('')
  })
  it('never reads a token from a query string (those leak into analytics)', () => {
    expect(tokenFromHash('?token=abc')).toBe('')
  })
})

import { statusLine, type NewsletterAdminInfo } from './newsletter'

const info = (over: Partial<NewsletterAdminInfo> = {}): NewsletterAdminInfo => ({
  enabled: true, toggle_on: true, mail_configured: true, base_url_local: false,
  confirmed: 12, pending: 3, timezone: 'Europe/London', send_day: 4, send_time: '09:00',
  ...over,
})

describe('statusLine', () => {
  it('explains that missing mail config switches the feature off', () => {
    expect(statusLine(info({ mail_configured: false, enabled: false }))).toMatch(/mail isn't configured/i)
  })
  it('says when it is switched off', () => {
    expect(statusLine(info({ toggle_on: false, enabled: false }))).toMatch(/switched off/i)
  })
  it('flags a missing send time instead of silently never sending', () => {
    expect(statusLine(info({ send_time: '' }))).toMatch(/no send time/i)
  })
  it('describes the schedule with the server time zone', () => {
    expect(statusLine(info())).toBe('Sending every Thursday at 09:00 (Europe/London).')
  })
  it('puts missing mail ahead of the toggle', () => {
    expect(statusLine(info({ mail_configured: false, toggle_on: false, enabled: false }))).toMatch(/mail isn't configured/i)
  })
  it('tells the operator to set the application URL when it is localhost', () => {
    expect(statusLine(info({ base_url_local: true, enabled: false }))).toMatch(/application url|base_url/i)
  })
  it('reports missing mail before the localhost warning', () => {
    expect(statusLine(info({ mail_configured: false, base_url_local: true, enabled: false }))).toMatch(/mail isn't configured/i)
  })
})

describe('signupSlotIndex', () => {
  it('goes after the day that takes the running total past the threshold', () => {
    expect(signupSlotIndex([3, 4, 5, 2], 10)).toBe(2) // 3,7,12 -> after day index 2
    expect(signupSlotIndex([10, 1], 10)).toBe(0)
  })
  it('goes at the end when there are fewer events than the threshold', () => {
    expect(signupSlotIndex([2, 3], 10)).toBe(1)
  })
  it('is -1 with no days', () => {
    expect(signupSlotIndex([], 10)).toBe(-1)
  })
})
