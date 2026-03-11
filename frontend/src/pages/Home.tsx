import { useEffect, useState, useMemo } from 'preact/hooks'
import { pb, Event, Tag } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import { MiniCalendar } from '../components/MiniCalendar'
import './Home.css'

interface Props {
  path?: string
}

export function Home(_props: Props) {
  const [allEvents, setAllEvents] = useState<Event[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedDate, setSelectedDate] = useState<string | null>(null)

  useEffect(() => {
    async function load() {
      try {
        const [eventsResult, tagsResult] = await Promise.all([
          pb.collection('events').getList<Event>(1, 100, {
            filter: `status = 'published' && start_datetime >= '${new Date().toISOString().split('T')[0]}'`,
            sort: 'start_datetime',
            expand: 'place,tags',
          }),
          pb.collection('tags').getFullList<Tag>(),
        ])
        setAllEvents(eventsResult.items)
        setTags(tagsResult)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load events')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  // Extract dates that have events
  const eventDates = useMemo(() => {
    const dates = new Set<string>()
    allEvents.forEach(event => {
      // PocketBase uses space separator, not 'T'
      const date = event.start_datetime.split(' ')[0]
      dates.add(date)
    })
    return dates
  }, [allEvents])

  // Filter events by selected date
  const filteredEvents = useMemo(() => {
    if (!selectedDate) return allEvents
    return allEvents.filter(event => event.start_datetime.startsWith(selectedDate))
  }, [allEvents, selectedDate])

  const handleDateSelect = (date: string) => {
    // Toggle selection - if already selected, clear it
    setSelectedDate(prev => prev === date ? null : date)
  }

  const clearDateFilter = () => {
    setSelectedDate(null)
  }

  if (loading) {
    return <div class="loading">Loading events...</div>
  }

  if (error) {
    return <div class="error">{error}</div>
  }

  const formatSelectedDate = (dateStr: string) => {
    const date = new Date(dateStr + 'T00:00:00')
    return date.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })
  }

  return (
    <div class="home">
      <div class="home-main">
        <div class="events-header">
          <h2>
            {selectedDate ? formatSelectedDate(selectedDate) : 'Upcoming Events'}
          </h2>
          {selectedDate && (
            <button class="clear-filter" onClick={clearDateFilter}>
              Show all
            </button>
          )}
        </div>
        {filteredEvents.length === 0 ? (
          <p class="no-events">
            {selectedDate ? 'No events on this date' : 'No upcoming events'}
          </p>
        ) : (
          <div class="events-grid">
            {filteredEvents.map(event => (
              <EventCard key={event.id} event={event} />
            ))}
          </div>
        )}
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
