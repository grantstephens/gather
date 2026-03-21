import { useEffect, useState, useRef } from 'preact/hooks'
import { pb, Event, Tag } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import { MiniCalendar } from '../components/MiniCalendar'
import './Home.css'

const PAGE_SIZE = 20

interface Props {
  path?: string
}

export function Home(_props: Props) {
  const [events, setEvents] = useState<Event[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [tagCounts, setTagCounts] = useState<Record<string, number>>({})
  const [eventDates, setEventDates] = useState<Map<string, string[]>>(new Map())
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedDate, setSelectedDate] = useState<string | null>(null)
  const pageRef = useRef(1)
  const loadingMoreRef = useRef(false)
  const sentinelRef = useRef<HTMLDivElement>(null)
  const today = new Date().toISOString().split('T')[0]

  // Fetch dates + tag colors for the calendar.
  // '$autoCancel': false prevents the SDK from cancelling this request when the
  // main events getList fires (both use the same collection path as requestKey).
  useEffect(() => {
    pb.collection('events').getFullList({
      filter: `status = 'published' && start_datetime >= '${today}'`,
      fields: 'start_datetime,expand.tags.color',
      expand: 'tags',
      '$autoCancel': false,
    }).then((items: any[]) => {
      const dateColors = new Map<string, string[]>()
      items.forEach(e => {
        const date = e.start_datetime.split(' ')[0]
        const colors: string[] = (e.expand?.tags ?? [])
          .map((t: any) => t.color)
          .filter(Boolean)
        if (!dateColors.has(date)) dateColors.set(date, [])
        const existing = dateColors.get(date)!
        // Add unique colors; fall back to empty string (→ default CSS color) if untagged
        if (colors.length === 0) {
          if (!existing.includes('')) existing.push('')
        } else {
          colors.forEach(c => { if (!existing.includes(c)) existing.push(c) })
        }
      })
      setEventDates(dateColors)
    }).catch(() => {})
  }, [])

  // Fetch tag counts from backend
  useEffect(() => {
    const controller = new AbortController()
    fetch('/api/tags/counts', { signal: controller.signal })
      .then(r => r.json())
      .then((rows: { id: string; count: number }[]) => {
        const counts: Record<string, number> = {}
        rows.forEach(r => { counts[r.id] = r.count })
        setTagCounts(counts)
      })
      .catch(() => {})
    return () => controller.abort()
  }, [])

  useEffect(() => {
    pb.collection('tags').getFullList<Tag>().then(setTags).catch(() => {})
  }, [])

  const fetchPage = async (page: number, date: string | null): Promise<boolean> => {
    const dateFilter = date
      ? `start_datetime >= '${date} 00:00:00' && start_datetime <= '${date} 23:59:59'`
      : `start_datetime >= '${today}'`
    const result = await pb.collection('events').getList<Event>(page, PAGE_SIZE, {
      filter: `status = 'published' && ${dateFilter}`,
      sort: 'start_datetime',
      expand: 'place,tags',
    })
    setEvents(prev => page === 1 ? result.items : [...prev, ...result.items])
    return result.page < result.totalPages
  }

  // Reset and load page 1 whenever date filter changes
  useEffect(() => {
    pageRef.current = 1
    setLoading(true)
    setError(null)
    fetchPage(1, selectedDate)
      .then(more => setHasMore(more))
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load events'))
      .finally(() => setLoading(false))
  }, [selectedDate])

  // Infinite scroll — only when no date filter active
  useEffect(() => {
    const sentinel = sentinelRef.current
    if (!sentinel || !hasMore || selectedDate) return

    const observer = new IntersectionObserver(async (entries) => {
      if (!entries[0].isIntersecting || loadingMoreRef.current) return
      loadingMoreRef.current = true
      setLoadingMore(true)
      const nextPage = pageRef.current + 1
      try {
        const more = await fetchPage(nextPage, null)
        pageRef.current = nextPage
        setHasMore(more)
      } finally {
        loadingMoreRef.current = false
        setLoadingMore(false)
      }
    }, { rootMargin: '300px' })

    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [hasMore, selectedDate])

  const handleDateSelect = (date: string) => {
    setSelectedDate(prev => prev === date ? null : date)
  }

  const formatSelectedDate = (dateStr: string) => {
    const date = new Date(dateStr + 'T00:00:00')
    return date.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })
  }

  if (loading) return <div class="loading">Loading events...</div>
  if (error) return <div class="error">{error}</div>

  return (
    <div class="home">
      <div class="home-main">
        <div class="events-header">
          <h2>{selectedDate ? formatSelectedDate(selectedDate) : 'Upcoming Events'}</h2>
          {selectedDate && (
            <button class="clear-filter" onClick={() => setSelectedDate(null)}>Show all</button>
          )}
        </div>
        {events.length === 0 ? (
          <p class="no-events">
            {selectedDate ? 'No events on this date' : 'No upcoming events'}
          </p>
        ) : (
          <div class="events-grid">
            {events.map(event => (
              <EventCard key={event.id} event={event} />
            ))}
          </div>
        )}
        <div ref={sentinelRef} />
        {loadingMore && <p class="loading-more">Loading more events...</p>}
      </div>
      <aside class="home-sidebar">
        <div class="sidebar-section">
          <MiniCalendar
            eventDates={eventDates}
            selectedDate={selectedDate ?? undefined}
            onDateSelect={handleDateSelect}
          />
        </div>
        <div class="sidebar-section">
          <h3>Tags</h3>
          <div class="tag-cloud">
            {[...tags]
              .sort((a, b) => (tagCounts[b.id] ?? 0) - (tagCounts[a.id] ?? 0))
              .map(tag => (
                <a
                  key={tag.id}
                  href={`/tag/${tag.name}`}
                  class="tag"
                  style={tag.color ? { backgroundColor: tag.color } : undefined}
                >
                  {tag.name}{tagCounts[tag.id] ? ` (${tagCounts[tag.id]})` : ''}
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
