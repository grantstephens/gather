import { useEffect, useState } from 'preact/hooks'
import { pb, Event, Tag as TagType } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import './Tag.css'

interface Props {
  path?: string
  name?: string
}

export function Tag({ name }: Props) {
  const [tag, setTag] = useState<TagType | null>(null)
  const [events, setEvents] = useState<Event[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!name) return

    async function load() {
      try {
        const tagRecord = await pb.collection('tags').getFirstListItem<TagType>(
          pb.filter('name = {:name}', { name })
        )
        setTag(tagRecord)

        const now = new Date().toISOString()
        const eventRecords = await pb.collection('events').getList<Event>(1, 50, {
          filter: pb.filter('status = "published" && tags.id ?= {:tagId} && start_datetime >= {:now}', {
            tagId: tagRecord.id,
            now,
          }),
          sort: 'start_datetime',
          expand: 'place,tags',
        })
        setEvents(eventRecords.items)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Tag not found')
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [name])

  if (loading) {
    return <div class="loading">Loading...</div>
  }

  if (error || !tag) {
    return <div class="error">{error || 'Tag not found'}</div>
  }

  return (
    <div class="tag-page">
      <header class="tag-header">
        <h1 style={tag.color ? { color: tag.color } : undefined}>#{tag.name}</h1>
        <p class="tag-count">{events.length} event{events.length !== 1 ? 's' : ''}</p>
        <a href={`/feed/tag/${tag.name}.rss`} class="feed-link">RSS Feed</a>
      </header>

      {events.length === 0 ? (
        <p class="no-events">No events with this tag yet.</p>
      ) : (
        <div class="events-grid">
          {events.map(event => (
            <EventCard key={event.id} event={event} />
          ))}
        </div>
      )}
    </div>
  )
}
