import { useEffect, useState } from 'preact/hooks'
import { pb, Event, Tag as TagType } from '../lib/pocketbase'
import { EventTimeline } from '../components/EventTimeline'
import { SkeletonTimeline } from '../components/Skeleton'
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
  const [page, setPage] = useState(1)
  const [hasMore, setHasMore] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [total, setTotal] = useState(0)

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
        setTotal(eventRecords.totalItems)
        setPage(1)
        setHasMore(eventRecords.totalPages > 1)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Tag not found')
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [name])

  const loadMore = async () => {
    if (!tag) return
    setLoadingMore(true)
    const nextPage = page + 1
    const now = new Date().toISOString()
    const more = await pb.collection('events').getList<Event>(nextPage, 50, {
      filter: pb.filter('status = "published" && tags.id ?= {:tagId} && start_datetime >= {:now}', {
        tagId: tag.id,
        now,
      }),
      sort: 'start_datetime',
      expand: 'place,tags',
    })
    setEvents(prev => [...prev, ...more.items])
    setPage(nextPage)
    setHasMore(nextPage < more.totalPages)
    setLoadingMore(false)
  }

  if (loading) {
    return (
      <div class="tag-page">
        <SkeletonTimeline />
      </div>
    )
  }

  if (error || !tag) {
    return <div class="error">{error || 'Tag not found'}</div>
  }

  return (
    <div class="tag-page">
      <header class="tag-header">
        <h1 style={tag.color ? { color: tag.color } : undefined}>#{tag.name}</h1>
        <p class="tag-count">{total} event{total !== 1 ? 's' : ''}</p>
        <a href={`/feed/tag/${tag.name}.rss`} class="feed-link" data-umami-event="tag-rss">RSS Feed</a>
      </header>

      {events.length === 0 ? (
        <p class="no-events">No events with this tag yet.</p>
      ) : (
        <>
          <EventTimeline events={events} />
          {hasMore && (
            <button class="btn btn-secondary" onClick={loadMore} disabled={loadingMore}>
              {loadingMore ? 'Loading...' : 'Load more'}
            </button>
          )}
        </>
      )}
    </div>
  )
}
