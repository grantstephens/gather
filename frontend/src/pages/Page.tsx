import { useEffect, useState } from 'preact/hooks'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { pb } from '../lib/pocketbase'
import type { PageRecord } from '../lib/pocketbase'

interface Props {
  path?: string
  slug?: string
}

export function Page({ slug }: Props) {
  const [page, setPage] = useState<PageRecord | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!slug) {
      setLoading(false)
      return
    }
    async function load() {
      try {
        const record = await pb.collection('pages').getFirstListItem<PageRecord>(
          pb.filter('slug = {:slug}', { slug })
        )
        setPage(record)
      } catch {
        setPage(null)
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [slug])

  if (loading) return <div class="loading">Loading...</div>

  if (!page) {
    return (
      <div class="page-not-found">
        <h1>Page not found</h1>
        <p><a href="/">Return to home</a></p>
      </div>
    )
  }

  return (
    <article class="custom-page">
      <h1>{page.title}</h1>
      {page.content && (
        <div
          class="page-content"
          dangerouslySetInnerHTML={{
            __html: DOMPurify.sanitize(marked.parse(page.content) as string)
          }}
        />
      )}
    </article>
  )
}
