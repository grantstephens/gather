import { describe, it, expect } from 'vitest'
import { textColorForBg, tagStyle } from './color'

// WCAG relative luminance / contrast, mirrored here to assert the *contract*
// (readable text) rather than the implementation's internal branch. Named
// because these encode the WCAG 2.x formula's fixed coefficients, which an
// inlined expression at the call site wouldn't explain.
function chanLum(c: number): number {
  const s = c / 255
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}
function relativeLuminance(r: number, g: number, b: number): number {
  return 0.2126 * chanLum(r) + 0.7152 * chanLum(g) + 0.0722 * chanLum(b)
}
function contrastRatio(l1: number, l2: number): number {
  const [a, b] = l1 > l2 ? [l1, l2] : [l2, l1]
  return (a + 0.05) / (b + 0.05)
}

describe('textColorForBg', () => {
  it('picks white text for a dark hsl() background (real tag.color format)', () => {
    // Auto-generated tag colors are stored as hsl(hue, 65%, 45%) — see
    // internal/hooks/moderation.go hashColor(). The old hex-only parser
    // returned NaN channels for hsl() input and always fell back to dark
    // text, failing WCAG AA against dark/saturated backgrounds like this one.
    const text = textColorForBg('hsl(235, 65%, 45%)')
    const ratio = contrastRatio(relativeLuminance(40, 53, 189), relativeLuminance(255, 255, 255))
    expect(text).toBe('#ffffff')
    expect(ratio).toBeGreaterThanOrEqual(4.5)
  })

  it('picks dark text for a light hsl() background', () => {
    expect(textColorForBg('hsl(60, 65%, 85%)')).toBe('#0f172a')
  })

  it('still handles #rrggbb (manual admin color picker format)', () => {
    expect(textColorForBg('#0d1117')).toBe('#ffffff')
    expect(textColorForBg('#f5f5f5')).toBe('#0f172a')
  })

  it('falls back to a defined color for unparseable input instead of throwing', () => {
    expect(() => textColorForBg('not-a-color')).not.toThrow()
    expect(textColorForBg('not-a-color')).toBe('#0f172a')
  })
})

describe('tagStyle', () => {
  it('returns undefined when no color is set', () => {
    expect(tagStyle(undefined)).toBeUndefined()
  })

  it('pairs the background with a contrast-appropriate text color', () => {
    expect(tagStyle('hsl(235, 65%, 45%)')).toEqual({
      backgroundColor: 'hsl(235, 65%, 45%)',
      color: '#ffffff',
    })
  })
})
