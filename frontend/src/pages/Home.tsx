import { useEffect, useState, useRef } from 'preact/hooks'
import { pb, Event, Tag } from '../lib/pocketbase'
import { MiniCalendar } from '../components/MiniCalendar'
import { tagStyle } from '../lib/color'
import { EventTimeline } from '../components/EventTimeline'
import { SkeletonTimeline } from '../components/Skeleton'
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
  const [mobilePanel, setMobilePanel] = useState<'calendar' | 'tags' | null>(null)
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
  }, [hasMore, selectedDate, loading])

  const handleDateSelect = (date: string) => {
    setSelectedDate(prev => prev === date ? null : date)
  }

  const formatSelectedDate = (dateStr: string) => {
    const date = new Date(dateStr + 'T00:00:00')
    return date.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })
  }

  const togglePanel = (panel: 'calendar' | 'tags') => {
    setMobilePanel(prev => prev === panel ? null : panel)
  }

  const sidebarContent = (
    <aside class="home-sidebar">
      <div class="sidebar-section">
        <div class="sidebar-section-title">Browse by date</div>
        <MiniCalendar
          eventDates={eventDates}
          selectedDate={selectedDate ?? undefined}
          onDateSelect={handleDateSelect}
        />
      </div>
      <div class="sidebar-section">
        <div class="sidebar-section-title">Browse by tag</div>
        <div class="tag-cloud">
          {[...tags]
            .sort((a, b) => (tagCounts[b.id] ?? 0) - (tagCounts[a.id] ?? 0))
            .map(tag => (
              <a
                key={tag.id}
                href={`/tag/${tag.name}`}
                class="tag"
                style={tagStyle(tag.color)}
              >
                {tag.name}{tagCounts[tag.id] ? ` (${tagCounts[tag.id]})` : ''}
              </a>
            ))}
        </div>
      </div>
    </aside>
  )

  if (loading) return (
    <div class="home">
      <div class="home-main">
        <SkeletonTimeline />
      </div>
      {sidebarContent}
    </div>
  )
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
          <EventTimeline events={events} />
        )}
        <div ref={sentinelRef} />
        {loadingMore && <p class="loading-more">Loading more events...</p>}
      </div>
      {sidebarContent}

      {/* Mobile floating buttons + panels */}
      {mobilePanel && (
        <div class="mobile-panel-backdrop" onClick={() => setMobilePanel(null)} />
      )}
      <div class={`mobile-panel ${mobilePanel === 'calendar' ? 'open' : ''}`}>
        <MiniCalendar
          eventDates={eventDates}
          selectedDate={selectedDate ?? undefined}
          onDateSelect={(date) => { handleDateSelect(date); setMobilePanel(null) }}
        />
      </div>
      <div class={`mobile-panel mobile-panel-tags ${mobilePanel === 'tags' ? 'open' : ''}`}>
        <div class="tag-cloud">
          {[...tags]
            .sort((a, b) => (tagCounts[b.id] ?? 0) - (tagCounts[a.id] ?? 0))
            .map(tag => (
              <a
                key={tag.id}
                href={`/tag/${tag.name}`}
                class="tag"
                style={tagStyle(tag.color)}
              >
                {tag.name}{tagCounts[tag.id] ? ` (${tagCounts[tag.id]})` : ''}
              </a>
            ))}
        </div>
      </div>
      <div class="mobile-fab-bar">
        <button
          class={`mobile-fab ${mobilePanel === 'calendar' ? 'active' : ''}`}
          onClick={() => togglePanel('calendar')}
          aria-label="Calendar"
        >
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
            <line x1="16" y1="2" x2="16" y2="6" />
            <line x1="8" y1="2" x2="8" y2="6" />
            <line x1="3" y1="10" x2="21" y2="10" />
          </svg>
        </button>
        <button
          class={`mobile-fab ${mobilePanel === 'tags' ? 'active' : ''}`}
          onClick={() => togglePanel('tags')}
          aria-label="Tags"
        >
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20.59 13.41l-7.17 7.17a2 2 0 01-2.83 0L2 12V2h10l8.59 8.59a2 2 0 010 2.82z" />
            <line x1="7" y1="7" x2="7.01" y2="7" />
          </svg>
        </button>
      </div>
    </div>
  )
}
