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

  const entries = [...grouped.entries()]

  return (
    <div class="timeline">
      {entries.map(([dateKey, dayEvents], i) => {
        const month = dateKey.slice(0, 7) // "YYYY-MM"
        const prevMonth = i > 0 ? entries[i - 1][0].slice(0, 7) : month
        const showMonthBreak = i > 0 && month !== prevMonth

        return (
          <>
            {showMonthBreak && (
              <div class="timeline-month-break" key={`month-${month}`}>
                <span class="timeline-month-label">
                  {format(parseISO(dateKey), 'MMMM yyyy')}
                </span>
              </div>
            )}
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
          </>
        )
      })}
    </div>
  )
}
