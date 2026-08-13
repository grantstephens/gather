import { Event } from './pocketbase'

export function shuffle<T>(array: T[]): T[] {
  const result = [...array]
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[result[i], result[j]] = [result[j], result[i]]
  }
  return result
}

export function groupEventsByDayShuffled(
  events: Event[],
  cache: Map<string, string[]>
): Map<string, Event[]> {
  const byDay = new Map<string, Event[]>()
  for (const event of events) {
    const dateKey = event.start_datetime?.split(' ')[0] ?? ''
    if (!byDay.has(dateKey)) byDay.set(dateKey, [])
    byDay.get(dateKey)!.push(event)
  }

  const result = new Map<string, Event[]>()
  for (const [dateKey, dayEvents] of byDay) {
    const byId = new Map(dayEvents.map(e => [e.id, e]))
    const currentIds = [...byId.keys()].sort().join(',')
    const cached = cache.get(dateKey)
    const cachedIds = cached ? [...cached].sort().join(',') : null

    let order: string[]
    if (cached && cachedIds === currentIds) {
      order = cached
    } else {
      order = shuffle(dayEvents).map(e => e.id)
      cache.set(dateKey, order)
    }

    result.set(dateKey, order.map(id => byId.get(id)!))
  }

  return result
}
