import { marked } from 'marked'

const ENTITIES: Record<string, string> = {
  '&amp;': '&',
  '&lt;': '<',
  '&gt;': '>',
  '&quot;': '"',
  '&#39;': "'",
}

// Plain-text version of a Markdown blurb for meta descriptions. Must produce the
// same text as the server's truncateText(stripMarkdownInline(...), maxLen) in
// internal/seo (opengraph.go / jsonld.go).
export function blurbToPlainText(md: string, maxLen = 160): string {
  const html = marked.parse(md, { async: false }) as string
  const text = html
    .replace(/<\/(p|h[1-6]|li|ul|ol|blockquote)>|<br\s*\/?>/gi, ' ')
    .replace(/<[^>]*>/g, '')
    .replace(/&(?:amp|lt|gt|quot|#39);/g, (e) => ENTITIES[e])
    .replace(/\s+/g, ' ')
    .trim()

  if (text.length <= maxLen) return text
  const cut = text.slice(0, maxLen)
  const lastSpace = cut.lastIndexOf(' ')
  return (lastSpace > 0 ? cut.slice(0, lastSpace) : cut) + '...'
}
