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
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set())
  const [mobilePanel, setMobilePanel] = useState<'calendar' | 'tags' | 'towns' | null>(null)
  const [towns, setTowns] = useState<{ name: string; count: number }[]>([])
  const [selectedTown, setSelectedTown] = useState<string | null>(null)
  const pageRef = useRef(1)
  const loadingMoreRef = useRef(false)
  const sentinelRef = useRef<HTMLDivElement>(null)
  const today = new Date().toISOString().split('T')[0]

  // Serialise selectedTags for use as a dependency key (Sets aren't comparable)
  const selectedTagsKey = [...selectedTags].sort().join(',')

  // Fetch dates + tag colors for the calendar.
  // '$autoCancel': false prevents the SDK from cancelling this request when the
  // main events getList fires (both use the same collection path as requestKey).
  useEffect(() => {
    const filters = [`status = 'published'`, `start_datetime >= '${today}'`]
    if (selectedTags.size > 0) {
      filters.push(`(${[...selectedTags].map(id => `tags.id ?= '${id}'`).join(' || ')})`)
    }
    pb.collection('events').getFullList({
      filter: filters.join(' && '),
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
  }, [selectedTagsKey])

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

  useEffect(() => {
    const controller = new AbortController()
    fetch('/api/towns/counts', { signal: controller.signal })
      .then(r => r.json())
      .then((rows: { name: string; count: number }[]) => setTowns(rows))
      .catch(() => {})
    return () => controller.abort()
  }, [])

  const fetchPage = async (page: number, date: string | null, town: string | null, tagIds: Set<string>): Promise<boolean> => {
    const filters = [`status = 'published'`]
    if (date) {
      filters.push(`start_datetime >= '${date} 00:00:00' && start_datetime <= '${date} 23:59:59'`)
    } else {
      filters.push(`start_datetime >= '${today}'`)
    }
    if (town) {
      filters.push(`place.city = '${town}'`)
    }
    if (tagIds.size > 0) {
      filters.push(`(${[...tagIds].map(id => `tags.id ?= '${id}'`).join(' || ')})`)
    }
    const result = await pb.collection('events').getList<Event>(page, PAGE_SIZE, {
      filter: filters.join(' && '),
      sort: 'start_datetime',
      expand: 'place,tags',
    })
    setEvents(prev => page === 1 ? result.items : [...prev, ...result.items])
    return result.page < result.totalPages
  }

  // Reset and load page 1 whenever any filter changes
  useEffect(() => {
    pageRef.current = 1
    setLoading(true)
    setError(null)
    fetchPage(1, selectedDate, selectedTown, selectedTags)
      .then(more => setHasMore(more))
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load events'))
      .finally(() => setLoading(false))
  }, [selectedDate, selectedTown, selectedTagsKey])

  // Infinite scroll — only when no date/tag filter active
  useEffect(() => {
    const sentinel = sentinelRef.current
    if (!sentinel || !hasMore || selectedDate || selectedTags.size > 0) return

    const observer = new IntersectionObserver(async (entries) => {
      if (!entries[0].isIntersecting || loadingMoreRef.current) return
      loadingMoreRef.current = true
      setLoadingMore(true)
      const nextPage = pageRef.current + 1
      try {
        const more = await fetchPage(nextPage, null, selectedTown, selectedTags)
        pageRef.current = nextPage
        setHasMore(more)
      } finally {
        loadingMoreRef.current = false
        setLoadingMore(false)
      }
    }, { rootMargin: '300px' })

    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [hasMore, selectedDate, selectedTown, selectedTagsKey, loading])

  const handleDateSelect = (date: string) => {
    setSelectedDate(prev => prev === date ? null : date)
  }

  const handleTagToggle = (tagId: string) => {
    setSelectedTags(prev => {
      const next = new Set(prev)
      if (next.has(tagId)) next.delete(tagId)
      else next.add(tagId)
      return next
    })
  }

  const formatSelectedDate = (dateStr: string) => {
    const date = new Date(dateStr + 'T00:00:00')
    return date.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })
  }

  const handleTownSelect = (town: string) => {
    setSelectedTown(prev => prev === town ? null : town)
  }

  const togglePanel = (panel: 'calendar' | 'tags' | 'towns') => {
    setMobilePanel(prev => prev === panel ? null : panel)
  }

  const hasActiveFilters = selectedDate || selectedTown || selectedTags.size > 0

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
              <button
                key={tag.id}
                class={`tag${selectedTags.has(tag.id) ? ' tag--selected' : ''}`}
                style={tagStyle(tag.color)}
                onClick={() => handleTagToggle(tag.id)}
                data-umami-event="home-tag-click"
              >
                {tag.name}{tagCounts[tag.id] ? ` (${tagCounts[tag.id]})` : ''}
              </button>
            ))}
        </div>
      </div>
      {towns.length > 0 && (
        <div class="sidebar-section">
          <div class="sidebar-section-title">Browse by town</div>
          <div class="town-cloud">
            {towns.map(town => (
              <button
                key={town.name}
                class={`town ${selectedTown === town.name ? 'active' : ''}`}
                onClick={() => handleTownSelect(town.name)}
              >
                {town.name} ({town.count})
              </button>
            ))}
          </div>
        </div>
      )}
    </aside>
  )

  const mobileFilterBar = (
    <div class="mobile-filter-bar">
      <button
        class={`mobile-filter-btn ${mobilePanel === 'calendar' ? 'active' : ''}`}
        onClick={() => togglePanel('calendar')}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
          <line x1="16" y1="2" x2="16" y2="6" />
          <line x1="8" y1="2" x2="8" y2="6" />
          <line x1="3" y1="10" x2="21" y2="10" />
        </svg>
        Dates
      </button>
      <button
        class={`mobile-filter-btn ${mobilePanel === 'tags' ? 'active' : ''}`}
        onClick={() => togglePanel('tags')}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20.59 13.41l-7.17 7.17a2 2 0 01-2.83 0L2 12V2h10l8.59 8.59a2 2 0 010 2.82z" />
          <line x1="7" y1="7" x2="7.01" y2="7" />
        </svg>
        Tags
      </button>
      <button
        class={`mobile-filter-btn ${mobilePanel === 'towns' ? 'active' : ''}`}
        onClick={() => togglePanel('towns')}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7z"/>
          <circle cx="12" cy="9" r="2.5"/>
        </svg>
        Towns
      </button>
    </div>
  )

  if (loading) return (
    <div class="home">
      <div class="home-main">
        {mobileFilterBar}
        <SkeletonTimeline />
      </div>
      {sidebarContent}
    </div>
  )
  if (error) return <div class="error">{error}</div>

  return (
    <div class="home">
      <div class="home-main">
        {mobileFilterBar}
        <div class="events-header">
          <h2>
            {selectedDate && selectedTown
              ? `${formatSelectedDate(selectedDate)} in ${selectedTown}`
              : selectedDate
                ? formatSelectedDate(selectedDate)
                : selectedTown
                  ? `Events in ${selectedTown}`
                  : 'Upcoming Events'}
          </h2>
          {hasActiveFilters && (
            <div class="active-filters">
              {selectedDate && (
                <button class="clear-filter" onClick={() => setSelectedDate(null)}>
                  Clear date
                </button>
              )}
              {selectedTown && (
                <button class="clear-filter" onClick={() => setSelectedTown(null)}>
                  Clear town
                </button>
              )}
              {selectedTags.size > 0 && (
                <button class="clear-filter" onClick={() => setSelectedTags(new Set())}>
                  Clear tags
                </button>
              )}
            </div>
          )}
        </div>
        {selectedTags.size > 0 && (
          <div class="filter-chips">
            {tags.filter(t => selectedTags.has(t.id)).map(tag => (
              <button
                key={tag.id}
                class="filter-chip"
                style={tagStyle(tag.color)}
                onClick={() => handleTagToggle(tag.id)}
              >
                {tag.name}
                <span class="filter-chip-x">&times;</span>
              </button>
            ))}
          </div>
        )}
        {events.length === 0 ? (
          <p class="no-events">
            {hasActiveFilters ? 'No events matching your filters' : 'No upcoming events'}
          </p>
        ) : (
          <EventTimeline events={events} />
        )}
        <div ref={sentinelRef} />
        {loadingMore && <p class="loading-more">Loading more events...</p>}
      </div>
      {sidebarContent}

      {/* Mobile filter bar + panels */}
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
              <button
                key={tag.id}
                class={`tag${selectedTags.has(tag.id) ? ' tag--selected' : ''}`}
                style={tagStyle(tag.color)}
                onClick={() => handleTagToggle(tag.id)}
                data-umami-event="home-tag-click"
              >
                {tag.name}{tagCounts[tag.id] ? ` (${tagCounts[tag.id]})` : ''}
              </button>
            ))}
        </div>
      </div>
      <div class={`mobile-panel mobile-panel-towns ${mobilePanel === 'towns' ? 'open' : ''}`}>
        <div class="town-cloud">
          {towns.map(town => (
            <button
              key={town.name}
              class={`town ${selectedTown === town.name ? 'active' : ''}`}
              onClick={() => { handleTownSelect(town.name); setMobilePanel(null) }}
            >
              {town.name} ({town.count})
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
