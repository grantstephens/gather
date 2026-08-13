import { describe, it, expect, vi, afterEach } from 'vitest'
import { shuffle, groupEventsByDayShuffled } from './shuffleDayEvents'
import { Event } from './pocketbase'

function makeEvent(id: string, startDatetime: string): Event {
  return {
    id,
    title: `Event ${id}`,
    description: '',
    start_datetime: startDatetime,
    status: 'published',
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('shuffle', () => {
  it('returns a permutation containing the same elements', () => {
    const input = [1, 2, 3, 4, 5]
    const result = shuffle(input)
    expect(result).toHaveLength(input.length)
    expect([...result].sort()).toEqual([...input].sort())
  })

  it('does not mutate the input array', () => {
    const input = [1, 2, 3, 4, 5]
    const copy = [...input]
    shuffle(input)
    expect(input).toEqual(copy)
  })

  it('applies the Fisher-Yates algorithm deterministically given a fixed random source', () => {
    vi.spyOn(Math, 'random').mockReturnValue(0)
    const result = shuffle([1, 2, 3, 4])
    // With Math.random() always 0, each step swaps the current top
    // element to index 0, producing this exact known permutation.
    expect(result).toEqual([2, 3, 4, 1])
  })
})

describe('groupEventsByDayShuffled', () => {
  it('groups events by the date portion of start_datetime', () => {
    const events = [
      makeEvent('a', '2026-08-13 09:00:00.000Z'),
      makeEvent('b', '2026-08-13 18:00:00.000Z'),
      makeEvent('c', '2026-08-14 10:00:00.000Z'),
    ]
    const cache = new Map<string, string[]>()
    const grouped = groupEventsByDayShuffled(events, cache)

    expect([...grouped.keys()]).toEqual(['2026-08-13', '2026-08-14'])
    expect(grouped.get('2026-08-13')!.map(e => e.id).sort()).toEqual(['a', 'b'])
    expect(grouped.get('2026-08-14')!.map(e => e.id)).toEqual(['c'])
  })

  it('reuses the cached order when a day\'s event ID set is unchanged', () => {
    const events = [
      makeEvent('a', '2026-08-13 09:00:00.000Z'),
      makeEvent('b', '2026-08-13 18:00:00.000Z'),
      makeEvent('c', '2026-08-13 12:00:00.000Z'),
    ]
    const cache = new Map<string, string[]>()

    const first = groupEventsByDayShuffled(events, cache)
    const firstOrder = first.get('2026-08-13')!.map(e => e.id)

    // Simulate a refetch: new array/object references, same IDs and dates.
    const refetched = events.map(e => ({ ...e }))
    const second = groupEventsByDayShuffled(refetched, cache)
    const secondOrder = second.get('2026-08-13')!.map(e => e.id)

    expect(secondOrder).toEqual(firstOrder)
  })

  it('reshuffles when a day\'s event ID set changes', () => {
    const events = [
      makeEvent('a', '2026-08-13 09:00:00.000Z'),
      makeEvent('b', '2026-08-13 18:00:00.000Z'),
    ]
    const cache = new Map<string, string[]>()
    cache.set('2026-08-13', ['a', 'b'])

    const withNewEvent = [...events, makeEvent('c', '2026-08-13 12:00:00.000Z')]
    const result = groupEventsByDayShuffled(withNewEvent, cache)

    expect(result.get('2026-08-13')!.map(e => e.id).sort()).toEqual(['a', 'b', 'c'])
    expect(cache.get('2026-08-13')!.sort()).toEqual(['a', 'b', 'c'])
  })

  it('evicts cache entries for days no longer present in events', () => {
    const cache = new Map<string, string[]>()
    groupEventsByDayShuffled([makeEvent('a', '2026-08-13 09:00:00.000Z')], cache)
    expect(cache.has('2026-08-13')).toBe(true)

    groupEventsByDayShuffled([makeEvent('b', '2026-08-14 09:00:00.000Z')], cache)

    expect(cache.has('2026-08-13')).toBe(false)
    expect(cache.has('2026-08-14')).toBe(true)
  })
})
