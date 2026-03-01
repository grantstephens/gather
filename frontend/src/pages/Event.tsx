import { useEffect, useState, useRef } from 'preact/hooks'
import { format } from 'date-fns'
import L from 'leaflet'
import { pb, Event as EventType, getImageUrl } from '../lib/pocketbase'
import './Event.css'

interface Props {
  path?: string
  id?: string
}

export function Event({ id }: Props) {
  const [event, setEvent] = useState<EventType | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const mapRef = useRef<HTMLDivElement>(null)
  const mapInstance = useRef<L.Map | null>(null)

  useEffect(() => {
    if (!id) return
    async function load() {
      try {
        const result = await pb.collection('events').getOne<EventType>(id!, {
          expand: 'place,tags,author',
        })
        setEvent(result)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Event not found')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [id])

  useEffect(() => {
    if (!event?.expand?.place || !mapRef.current || mapInstance.current) return

    const place = event.expand.place
    const map = L.map(mapRef.current).setView([place.latitude, place.longitude], 15)
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; OpenStreetMap contributors',
    }).addTo(map)
    L.marker([place.latitude, place.longitude]).addTo(map)
    mapInstance.current = map

    return () => {
      map.remove()
      mapInstance.current = null
    }
  }, [event])

  if (loading) {
    return <div class="loading">Loading...</div>
  }

  if (error || !event) {
    return <div class="error">{error || 'Event not found'}</div>
  }

  const startDate = new Date(event.start_datetime)
  const endDate = event.end_datetime ? new Date(event.end_datetime) : null
  const imageUrl = getImageUrl(event, '800x600')

  return (
    <article class="event-page">
      {imageUrl && (
        <div class="event-hero">
          <img src={imageUrl} alt="" />
        </div>
      )}
      <div class="event-content">
        <header class="event-header">
          <time class="event-datetime">
            {format(startDate, 'EEEE, MMMM d, yyyy · h:mm a')}
            {endDate && ` - ${format(endDate, 'h:mm a')}`}
          </time>
          <h1>{event.title}</h1>
          {event.expand?.tags && event.expand.tags.length > 0 && (
            <div class="event-tags">
              {event.expand.tags.map(tag => (
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
          )}
        </header>

        {event.expand?.place && (
          <section class="event-location">
            <h2>Location</h2>
            <p class="place-name">{event.expand.place.name}</p>
            {event.expand.place.address && (
              <p class="place-address">{event.expand.place.address}</p>
            )}
            <div ref={mapRef} class="event-map" />
          </section>
        )}

        {event.description && (
          <section class="event-description">
            <h2>About</h2>
            <div dangerouslySetInnerHTML={{ __html: event.description }} />
          </section>
        )}

        <footer class="event-actions">
          <a href={`/event/${event.id}.ics`} class="btn">
            Download .ics
          </a>
        </footer>
      </div>
    </article>
  )
}
