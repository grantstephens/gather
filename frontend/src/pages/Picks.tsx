import { useEffect, useState } from 'preact/hooks'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { pb, PicksRecord, Event } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import './Picks.css'

interface Props {
  path?: string
}

function getThisWeekendWindow(): { start: Date; end: Date } {
  const now = new Date()
  const day = now.getDay() // 0=Sun, 1=Mon, ..., 5=Fri, 6=Sat
  let daysToFri: number
  if (day === 5) daysToFri = 0
  else if (day === 6) daysToFri = -1
  else if (day === 0) daysToFri = -2
  else daysToFri = 5 - day

  const friStart = new Date(now)
  friStart.setDate(now.getDate() + daysToFri)
  friStart.setHours(0, 0, 0, 0)

  const sunEnd = new Date(friStart)
  sunEnd.setDate(friStart.getDate() + 2)
  sunEnd.setHours(23, 59, 59, 999)

  return { start: friStart, end: sunEnd }
}

function isThisWeekend(post: PicksRecord): boolean {
  if (post.hidden) return false
  const { start, end } = getThisWeekendWindow()
  return (post.expand?.events ?? []).some(e => {
    const eventStart = new Date(e.start_datetime)
    return eventStart >= start && eventStart <= end
  })
}

function sortedEvents(events: Event[]): Event[] {
  return [...events].sort((a, b) =>
    new Date(a.start_datetime).getTime() - new Date(b.start_datetime).getTime()
  )
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
          expand: 'events,events.place,events.tags',
        })
        records.sort((a, b) => {
          const aMin = Math.min(...(a.expand?.events ?? []).map(e => new Date(e.start_datetime).getTime()))
          const bMin = Math.min(...(b.expand?.events ?? []).map(e => new Date(e.start_datetime).getTime()))
          return aMin - bMin
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
            const current = isThisWeekend(post)
            return (
              <article key={post.id} class={`picks-item${current ? ' picks-item--current' : ''}`}>
                <header class="picks-item-header">
                  {current && <span class="picks-badge">This Weekend</span>}
                  <h2><a href={`/picks/${post.slug}`} data-umami-event="picks-post-click" data-umami-event-slug={post.slug}>{post.title}</a></h2>
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
                    {sortedEvents(post.expand!.events!).map(event => (
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
