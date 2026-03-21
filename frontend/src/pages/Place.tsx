import { useEffect, useState } from 'preact/hooks'
import { pb, Event, Place as PlaceType } from '../lib/pocketbase'
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
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Place not found')
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [id])

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
        <p class="place-count">{events.length} event{events.length !== 1 ? 's' : ''}</p>
      </header>

      {events.length === 0 ? (
        <p class="no-events">No events at this place yet.</p>
      ) : (
        <EventTimeline events={events} />
      )}
    </div>
  )
}
