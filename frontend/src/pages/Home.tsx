import { useEffect, useState } from 'preact/hooks'
import { pb, Event, Tag } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import './Home.css'

interface Props {
  path?: string
}

export function Home(_props: Props) {
  const [events, setEvents] = useState<Event[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function load() {
      try {
        const [eventsResult, tagsResult] = await Promise.all([
          pb.collection('events').getList<Event>(1, 20, {
            filter: `status = 'published' && start_datetime >= '${new Date().toISOString()}'`,
            sort: 'start_datetime',
            expand: 'place,tags',
          }),
          pb.collection('tags').getFullList<Tag>(),
        ])
        setEvents(eventsResult.items)
        setTags(tagsResult)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load events')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  if (loading) {
    return <div class="loading">Loading events...</div>
  }

  if (error) {
    return <div class="error">{error}</div>
  }

  return (
    <div class="home">
      <div class="home-main">
        <h2>Upcoming Events</h2>
        {events.length === 0 ? (
          <p class="no-events">No upcoming events</p>
        ) : (
          <div class="events-grid">
            {events.map(event => (
              <EventCard key={event.id} event={event} />
            ))}
          </div>
        )}
      </div>
      <aside class="home-sidebar">
        <div class="sidebar-section">
          <h3>Tags</h3>
          <div class="tag-cloud">
            {tags.map(tag => (
              <a
                key={tag.id}
                href={`/tag/${tag.name}`}
                class="tag"
                style={tag.color ? { backgroundColor: tag.color } : undefined}
              >
                {tag.name}
              </a>
            ))}
          </div>
        </div>
        <div class="sidebar-section">
          <a href="/submit" class="btn btn-primary">
            + Add Event
          </a>
        </div>
      </aside>
    </div>
  )
}
