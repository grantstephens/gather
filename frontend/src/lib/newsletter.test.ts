import { describe, expect, it } from 'vitest'
import { tokenFromHash } from './newsletter'

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
