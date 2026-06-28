import { useEffect, useState } from 'preact/hooks'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { pb, PicksRecord, getImageUrl } from '../lib/pocketbase'
import { usePageTitle } from '../lib/title'
import { EventCard } from '../components/EventCard'
import './PicksPost.css'

interface Props {
  path?: string
  slug?: string
}

function setMetaTag(attr: string, value: string, content: string) {
  let el = document.querySelector(`meta[${attr}="${value}"]`)
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute(attr, value)
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

export function PicksPost({ slug }: Props) {
  const [post, setPost] = useState<PicksRecord | null>(null)
  const [loading, setLoading] = useState(true)
  usePageTitle(post?.title ?? '')

  useEffect(() => {
    if (!slug) {
      setLoading(false)
      return
    }
    async function load() {
      try {
        const record = await pb.collection('picks').getFirstListItem<PicksRecord>(
          pb.filter('slug = {:slug} && hidden = false', { slug }),
          { expand: 'events,events.place,events.tags' }
        )
        setPost(record)

        const plainBlurb = (marked.parse(record.blurb) as string).replace(/<[^>]*>/g, '').slice(0, 160)
        const pageUrl = window.location.href

        setMetaTag('name', 'description', plainBlurb)
        setMetaTag('property', 'og:type', 'article')
        setMetaTag('property', 'og:title', record.title)
        setMetaTag('property', 'og:description', plainBlurb)
        setMetaTag('property', 'og:url', pageUrl)
        setMetaTag('name', 'twitter:card', 'summary')
        setMetaTag('name', 'twitter:title', record.title)
        setMetaTag('name', 'twitter:description', plainBlurb)

        const firstEvent = record.expand?.events
          ?.slice()
          .sort((a, b) => new Date(a.start_datetime).getTime() - new Date(b.start_datetime).getTime())[0]
        if (firstEvent) {
          const imageUrl = getImageUrl(firstEvent, '800x600')
          if (imageUrl) {
            setMetaTag('property', 'og:image', imageUrl)
            setMetaTag('name', 'twitter:card', 'summary_large_image')
            setMetaTag('name', 'twitter:image', imageUrl)
          }
        }
      } catch {
        setPost(null)
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [slug])

  if (loading) return <div class="loading">Loading...</div>

  if (!post) {
    return (
      <div class="page-not-found">
        <h1>Post not found</h1>
        <p><a href="/picks">Back to Picks</a></p>
      </div>
    )
  }

  return (
    <article class="picks-post">
      <header class="picks-post-header">
        <h1>{post.title}</h1>
        {post.blurb && (
          <div
            class="picks-post-blurb"
            dangerouslySetInnerHTML={{
              __html: DOMPurify.sanitize(marked.parse(post.blurb) as string)
            }}
          />
        )}
      </header>
      {post.expand?.events && post.expand.events.length > 0 && (
        <div class="picks-post-events">
          {[...post.expand.events]
            .sort((a, b) => new Date(a.start_datetime).getTime() - new Date(b.start_datetime).getTime())
            .map(event => (
              <EventCard key={event.id} event={event} variant="featured" />
            ))}
        </div>
      )}
    </article>
  )
}
