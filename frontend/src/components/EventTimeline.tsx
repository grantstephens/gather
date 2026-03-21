import { useMemo } from 'preact/hooks'
import { format, parseISO } from 'date-fns'
import { Event } from '../lib/pocketbase'
import { EventCard } from './EventCard'
import './EventTimeline.css'

interface Props {
  events: Event[]
}

function formatDayHeading(dateKey: string): string {
  try {
    return format(parseISO(dateKey), 'EEEE, MMMM d')
  } catch {
    return dateKey
  }
}

export function EventTimeline({ events }: Props) {
  const grouped = useMemo(() => {
    const map = new Map<string, Event[]>()
    for (const event of events) {
      const dateKey = event.start_datetime?.split(' ')[0] ?? ''
      if (!map.has(dateKey)) map.set(dateKey, [])
      map.get(dateKey)!.push(event)
    }
    return map
  }, [events])

  if (events.length === 0) return null

  return (
    <div class="timeline">
      {[...grouped.entries()].map(([dateKey, dayEvents]) => (
        <section key={dateKey} class="timeline-day">
          <div class="timeline-date-marker">
            <time class="timeline-date-label">{formatDayHeading(dateKey)}</time>
          </div>
          <div class="timeline-day-events">
            <EventCard event={dayEvents[0]} variant="featured" />
            {dayEvents.length > 1 && (
              <div class="timeline-compact-row">
                {dayEvents.slice(1).map(e => (
                  <EventCard key={e.id} event={e} variant="compact" />
                ))}
              </div>
            )}
          </div>
        </section>
      ))}
    </div>
  )
}
