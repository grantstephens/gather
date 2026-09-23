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

/** Parse a CSS color string (#rrggbb or hsl(h, s%, l%)) into 0-255 RGB channels. */
function parseRgb(color: string): [r: number, g: number, b: number] | null {
  const hex = color.match(/^#([0-9a-f]{6})$/i)
  if (hex) {
    const n = hex[1]
    return [parseInt(n.substring(0, 2), 16), parseInt(n.substring(2, 4), 16), parseInt(n.substring(4, 6), 16)]
  }

  const hsl = color.match(/^hsl\(\s*([\d.]+)\s*,\s*([\d.]+)%\s*,\s*([\d.]+)%\s*\)$/i)
  if (hsl) {
    const h = parseFloat(hsl[1]) / 360
    const s = parseFloat(hsl[2]) / 100
    const l = parseFloat(hsl[3]) / 100
    if (s === 0) {
      const v = Math.round(l * 255)
      return [v, v, v]
    }
    const q = l < 0.5 ? l * (1 + s) : l + s - l * s
    const p = 2 * l - q
    const hue2rgb = (t: number): number => {
      if (t < 0) t += 1
      if (t > 1) t -= 1
      if (t < 1 / 6) return p + (q - p) * 6 * t
      if (t < 1 / 2) return q
      if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6
      return p
    }
    return [
      Math.round(hue2rgb(h + 1 / 3) * 255),
      Math.round(hue2rgb(h) * 255),
      Math.round(hue2rgb(h - 1 / 3) * 255),
    ]
  }

  return null
}

/** Return the text color (dark or white) with better WCAG contrast against the given CSS bg color. */
export function textColorForBg(color: string): string {
  const rgb = parseRgb(color)
  if (!rgb) return '#0f172a'
  const bgLum = relativeLuminance(...rgb)
  const whiteCR = contrastRatio(1, bgLum)     // white luminance = 1
  const darkCR = contrastRatio(bgLum, 0.0135) // #0f172a luminance ≈ 0.0135
  return whiteCR >= darkCR ? '#ffffff' : '#0f172a'
}

/** Build inline style for a colored tag. */
export function tagStyle(color: string | undefined) {
  if (!color) return undefined
  return { backgroundColor: color, color: textColorForBg(color) }
}
