import type { ComponentChildren } from 'preact'
import { useMemo, useRef } from 'preact/hooks'
import { format, parseISO } from 'date-fns'
import { Event } from '../lib/pocketbase'
import { groupEventsByDayShuffled } from '../lib/shuffleDayEvents'
import { EventCard } from './EventCard'
import { signupSlotIndex } from '../lib/newsletter'
import './EventTimeline.css'

interface Props {
  events: Event[]
  /** Rendered between day sections once `afterEvents` events have been shown (or at the end if fewer). */
  inlineSlot?: { afterEvents: number; node: ComponentChildren }
}

function formatDayHeading(dateKey: string): string {
  try {
    return format(parseISO(dateKey), 'EEEE, MMMM d')
  } catch {
    return dateKey
  }
}

export function EventTimeline({ events, inlineSlot }: Props) {
  const dayOrderCache = useRef(new Map<string, string[]>())
  const grouped = useMemo(
    () => groupEventsByDayShuffled(events, dayOrderCache.current),
    [events]
  )

  if (events.length === 0) return null

  const entries = [...grouped.entries()]
  const slotAfter = inlineSlot
    ? signupSlotIndex(entries.map(([, d]) => d.length), inlineSlot.afterEvents)
    : -1

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
                <EventCard event={dayEvents[0]} variant="featured" priority={i === 0} />
                {dayEvents.length > 1 && (
                  <div class="timeline-compact-row">
                    {dayEvents.slice(1).map(e => (
                      <EventCard key={e.id} event={e} variant="compact" />
                    ))}
                  </div>
                )}
              </div>
            </section>
            {inlineSlot && i === slotAfter && inlineSlot.node}
          </>
        )
      })}
    </div>
  )
}
