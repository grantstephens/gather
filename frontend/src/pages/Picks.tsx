import { useEffect, useState } from 'preact/hooks'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { pb, PicksRecord } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import './Picks.css'

interface Props {
  path?: string
}

function isCurrentPost(post: PicksRecord): boolean {
  if (post.hidden) return false
  const now = new Date().toISOString().replace('T', ' ').slice(0, 19)
  return (post.expand?.events ?? []).some(e => {
    const cutoff = e.end_datetime ?? e.start_datetime
    return cutoff >= now
  })
}

export function Picks(_props: Props) {
  const [posts, setPosts] = useState<PicksRecord[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    document.title = 'Picks'
    async function load() {
      try {
        const records = await pb.collection('picks').getFullList<PicksRecord>({
          filter: 'hidden = false',
          sort: '-created',
          expand: 'events,events.place,events.tags',
        })
        setPosts(records)
      } catch {
        // show empty state on failure
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  if (loading) return <div class="loading">Loading...</div>

  return (
    <div class="picks-page">
      <h1>Picks</h1>
      {posts.length === 0 ? (
        <p class="no-events">No picks yet.</p>
      ) : (
        <div class="picks-list">
          {posts.map(post => {
            const current = isCurrentPost(post)
            return (
              <article key={post.id} class={`picks-item${current ? ' picks-item--current' : ''}`}>
                <header class="picks-item-header">
                  {current && <span class="picks-badge">This Weekend</span>}
                  <h2><a href={`/picks/${post.slug}`}>{post.title}</a></h2>
                  {post.blurb && (
                    <div
                      class="picks-item-blurb"
                      dangerouslySetInnerHTML={{
                        __html: DOMPurify.sanitize(marked.parse(post.blurb) as string)
                      }}
                    />
                  )}
                </header>
                {post.expand?.events && post.expand.events.length > 0 && (
                  <div class="picks-item-events">
                    {post.expand.events.map(event => (
                      <EventCard key={event.id} event={event} variant="compact" />
                    ))}
                  </div>
                )}
              </article>
            )
          })}
        </div>
      )}
    </div>
  )
}
