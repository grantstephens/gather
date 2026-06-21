import { useEffect, useState } from 'preact/hooks'
import { pb, Event, Place as PlaceType } from '../lib/pocketbase'
import { usePageTitle } from '../lib/title'
import { EventTimeline } from '../components/EventTimeline'
import { SkeletonTimeline } from '../components/Skeleton'
import './Place.css'

interface Props {
  path?: string
  id?: string
}

export function Place({ id }: Props) {
  const [place, setPlace] = useState<PlaceType | null>(null)
  const [events, setEvents] = useState<Event[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [hasMore, setHasMore] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [total, setTotal] = useState(0)
  usePageTitle(place ? place.name : '')

  useEffect(() => {
    if (!id) return

    async function load() {
      try {
        const placeRecord = await pb.collection('places').getOne<PlaceType>(id!)
        setPlace(placeRecord)

        const now = new Date().toISOString()
        const eventRecords = await pb.collection('events').getList<Event>(1, 50, {
          filter: pb.filter('status = "published" && place = {:placeId} && start_datetime >= {:now}', {
            placeId: id,
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
        setError(err instanceof Error ? err.message : 'Place not found')
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [id])

  const loadMore = async () => {
    if (!id) return
    setLoadingMore(true)
    const nextPage = page + 1
    const now = new Date().toISOString()
    const more = await pb.collection('events').getList<Event>(nextPage, 50, {
      filter: pb.filter('status = "published" && place = {:placeId} && start_datetime >= {:now}', {
        placeId: id,
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
      <div class="place-page">
        <SkeletonTimeline />
      </div>
    )
  }

  if (error || !place) {
    return <div class="error">{error || 'Place not found'}</div>
  }

  return (
    <div class="place-page">
      <header class="place-header">
        <h1>{place.name}</h1>
        {place.address && <p class="place-address">{place.address}</p>}
        {place.city && <p class="place-city">{place.city}</p>}
        <p class="place-count">{total} event{total !== 1 ? 's' : ''}</p>
      </header>

      {events.length === 0 ? (
        <p class="no-events">No events at this place yet.</p>
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
