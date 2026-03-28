import { useEffect, useState, useRef } from 'preact/hooks'
import { format } from 'date-fns'
import { tagStyle } from '../lib/color'
import { route } from 'preact-router'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { pb, Event as EventType, getImageUrl, canModerate } from '../lib/pocketbase'
import { SkeletonEventDetailPage } from '../components/Skeleton'
import 'leaflet/dist/leaflet.css'
import './Event.css'

interface Props {
  path?: string
  id?: string
}

export function Event({ id }: Props) {
  const [event, setEvent] = useState<EventType | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionLoading, setActionLoading] = useState(false)
  const mapRef = useRef<HTMLDivElement>(null)
  const mapInstance = useRef<any>(null)
  const isModerator = canModerate()

  const handleDelete = async () => {
    if (!event || !confirm('Are you sure you want to delete this event?')) return
    setActionLoading(true)
    try {
      await pb.collection('events').delete(event.id)
      route('/')
    } catch (err) {
      alert('Failed to delete event')
      setActionLoading(false)
    }
  }

  const handleStatusChange = async (newStatus: 'published' | 'cancelled' | 'pending') => {
    if (!event) return
    setActionLoading(true)
    try {
      const updated = await pb.collection('events').update<EventType>(event.id, { status: newStatus })
      setEvent({ ...event, status: updated.status })
    } catch (err) {
      alert('Failed to update event status')
    } finally {
      setActionLoading(false)
    }
  }

  useEffect(() => {
    if (!id) return
    async function load() {
      try {
        let result: EventType
        try {
          result = await pb.collection('events').getOne<EventType>(id!, {
            expand: 'place,tags,author',
          })
        } catch {
          // Fall back to slug lookup
          const records = await pb.collection('events').getList<EventType>(1, 1, {
            filter: `slug = "${id}"`,
            expand: 'place,tags,author',
          })
          if (records.items.length === 0) throw new Error('Event not found')
          result = records.items[0]
        }
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

    // Dynamically import Leaflet only when needed (CSS imported at module level)
    import('leaflet').then((L) => {
      if (!mapRef.current || mapInstance.current) return

      const accentColor = getComputedStyle(document.documentElement).getPropertyValue('--color-accent').trim() || '#0d9488'
      const markerSvg = `<svg xmlns="http://www.w3.org/2000/svg" width="25" height="41" viewBox="0 0 25 41"><path d="M12.5 0C5.6 0 0 5.6 0 12.5 0 21.9 12.5 41 12.5 41S25 21.9 25 12.5C25 5.6 19.4 0 12.5 0z" fill="${accentColor}"/><circle cx="12.5" cy="12.5" r="5.5" fill="white"/></svg>`
      const markerIcon = L.default.divIcon({
        html: markerSvg,
        className: '',
        iconSize: [25, 41],
        iconAnchor: [12, 41],
        popupAnchor: [1, -34],
      })

      const map = L.default.map(mapRef.current).setView([place.latitude, place.longitude], 15)
      L.default.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; OpenStreetMap contributors',
      }).addTo(map)
      L.default.marker([place.latitude, place.longitude], { icon: markerIcon }).addTo(map)
      mapInstance.current = map
    }).catch(err => {
      console.error('Failed to load Leaflet:', err)
    })

    return () => {
      if (mapInstance.current) {
        mapInstance.current.remove()
        mapInstance.current = null
      }
    }
  }, [event])

  if (loading) {
    return <SkeletonEventDetailPage />
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
          {event.status !== 'published' && (
            <span class={`status-badge status-${event.status}`}>
              {event.status}
            </span>
          )}
          <h1>{event.title}</h1>
          <div class="event-meta">
            <time class="event-datetime">
              {format(startDate, 'EEEE, MMMM d, yyyy · h:mm a')}
              {endDate && ` - ${format(endDate, 'h:mm a')}`}
            </time>
            {event.expand?.tags && event.expand.tags.length > 0 && (
              <div class="event-tags">
                {event.expand.tags.map(tag => (
                  <a
                    key={tag.id}
                    href={`/tag/${tag.name}`}
                    class="tag"
                    style={tagStyle(tag.color)}
                    data-umami-event="event-tag-click"
                  >
                    {tag.name}
                  </a>
                ))}
              </div>
            )}
          </div>
        </header>

        {event.expand?.place && (
          <section class="event-location">
            <p class="section-label">Location</p>
            <p class="place-name">{event.expand.place.name}</p>
            {event.expand.place.address && (
              <p class="place-address">{event.expand.place.address}</p>
            )}
            <div ref={mapRef} class="event-map" />
          </section>
        )}

        {event.description && (
          <section class="event-description">
            <p class="section-label">About</p>
            <div dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(marked.parse(event.description) as string) }} />
          </section>
        )}

        <footer class="event-actions">
          <a href={`/ics/event/${event.id}`} class="btn" data-umami-event="event-ical-download">
            Download .ics
          </a>

          {isModerator && (
            <div class="admin-actions">
              <a href={`/edit/${event.id}`} class="btn btn-secondary">
                Edit
              </a>
              {event.status === 'pending' && (
                <button
                  class="btn btn-primary"
                  onClick={() => handleStatusChange('published')}
                  disabled={actionLoading}
                >
                  Approve
                </button>
              )}
              {event.status === 'published' && (
                <button
                  class="btn btn-secondary"
                  onClick={() => handleStatusChange('cancelled')}
                  disabled={actionLoading}
                >
                  Cancel Event
                </button>
              )}
              {event.status === 'cancelled' && (
                <button
                  class="btn btn-primary"
                  onClick={() => handleStatusChange('published')}
                  disabled={actionLoading}
                >
                  Republish
                </button>
              )}
              <button
                class="btn btn-danger"
                onClick={handleDelete}
                disabled={actionLoading}
              >
                Delete
              </button>
            </div>
          )}
        </footer>
      </div>
    </article>
  )
}
