/** WCAG relative luminance from an sRGB channel (0-255). */
function chanLum(c: number): number {
  const s = c / 255
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}

function relativeLuminance(r: number, g: number, b: number): number {
  return 0.2126 * chanLum(r) + 0.7152 * chanLum(g) + 0.0722 * chanLum(b)
}

function contrastRatio(l1: number, l2: number): number {
  const [lighter, darker] = l1 > l2 ? [l1, l2] : [l2, l1]
  return (lighter + 0.05) / (darker + 0.05)
}

/** Return the text color (dark or white) with better WCAG contrast against the given hex bg. */
export function textColorForBg(hex: string): string {
  const c = hex.replace('#', '')
  const r = parseInt(c.substring(0, 2), 16)
  const g = parseInt(c.substring(2, 4), 16)
  const b = parseInt(c.substring(4, 6), 16)
  const bgLum = relativeLuminance(r, g, b)
  const whiteCR = contrastRatio(1, bgLum)     // white luminance = 1
  const darkCR = contrastRatio(bgLum, 0.0135) // #0f172a luminance ≈ 0.0135
  return whiteCR >= darkCR ? '#ffffff' : '#0f172a'
}

/** Build inline style for a colored tag. */
export function tagStyle(color: string | undefined) {
  if (!color) return undefined
  return { backgroundColor: color, color: textColorForBg(color) }
}
