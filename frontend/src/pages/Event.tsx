import { useEffect, useState, useRef } from 'preact/hooks'
import { format } from 'date-fns'
import { tagStyle } from '../lib/color'
import { route } from 'preact-router'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { pb, Event as EventType, getImageUrl, canModerate, recurrenceLabel, parseVirtualId } from '../lib/pocketbase'
import { SkeletonEventDetailPage } from '../components/Skeleton'
import { ConfirmDialog } from '../components/ConfirmDialog'
import leafletCss from 'leaflet/dist/leaflet.css?inline'
import './Event.css'

interface Props {
  path?: string
  id?: string
}

export function Event({ id }: Props) {
  const [event, setEvent] = useState<EventType | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [cancelled, setCancelled] = useState(false)
  const [actionLoading, setActionLoading] = useState(false)
  const [confirmDialog, setConfirmDialog] = useState<{ message: string; onConfirm: () => void } | null>(null)
  const mapRef = useRef<HTMLDivElement>(null)
  const mapInstance = useRef<any>(null)
  const descriptionRef = useRef<HTMLDivElement>(null)
  const isModerator = canModerate()

  const handleDelete = () => {
    if (!event) return
    setConfirmDialog({
      message: 'Are you sure you want to delete this event?',
      onConfirm: async () => {
        setConfirmDialog(null)
        setActionLoading(true)
        const id = parseVirtualId(event.id)?.baseId || event.id
        try {
          await pb.collection('events').delete(id)
          route('/')
        } catch (err) {
          const msg = err instanceof Error ? err.message : String(err)
          alert(`Failed to delete event: ${msg}`)
          setActionLoading(false)
        }
      },
    })
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
            filter: pb.filter('slug = {:id}', { id }),
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

  // Realtime SSE subscription for the specific event record
  useEffect(() => {
    if (!event?.id) return
    // Virtual/recurring events have synthetic IDs — subscribe to the base record
    const baseId = parseVirtualId(event.id)?.baseId ?? event.id
    pb.collection('events').subscribe(baseId, ({ action, record }) => {
      if (action === 'delete') {
        setCancelled(true)
      } else if (action === 'update') {
        // Preserve expand — SSE payloads don't include relation data
        setEvent(prev => prev ? { ...prev, ...record, expand: prev.expand } : prev)
      }
    }).catch(console.error)
    return () => {
      pb.collection('events').unsubscribe(baseId)
    }
  }, [event?.id])

  useEffect(() => {
    if (!event?.expand?.place || !mapRef.current || mapInstance.current) return

    const place = event.expand.place
    if (!place.location) return
    const location = place.location

    if (!document.querySelector('style[data-leaflet-css]')) {
      const style = document.createElement('style')
      style.setAttribute('data-leaflet-css', '')
      style.textContent = leafletCss
      document.head.appendChild(style)
    }
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

      const lat = location.lat
      const lon = location.lon
      const map = L.default.map(mapRef.current).setView([lat, lon], 15)
      L.default.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '&copy; OpenStreetMap contributors',
      }).addTo(map)
      L.default.marker([lat, lon], { icon: markerIcon }).addTo(map)
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

  useEffect(() => {
    const el = descriptionRef.current
    if (!el || !event) return
    const handleClick = (e: MouseEvent) => {
      const anchor = (e.target as HTMLElement).closest('a')
      if (!anchor?.href) return
      try {
        const url = new URL(anchor.href)
        if (url.origin === window.location.origin) return
        ;(window as any).umami?.track('event-outbound-link', {
          url: anchor.href,
          domain: url.hostname,
          event_id: event.id,
          event_title: event.title,
        })
      } catch {}
    }
    el.addEventListener('click', handleClick)
    return () => el.removeEventListener('click', handleClick)
  }, [event])

  if (loading) {
    return <SkeletonEventDetailPage />
  }

  if (cancelled) {
    return (
      <div class="error">
        This event has been cancelled or deleted.{' '}
        <a href="/">Return to calendar</a>
      </div>
    )
  }

  if (error || !event) {
    return <div class="error">{error || 'Event not found'}</div>
  }

  const startDate = new Date(event.start_datetime)
  const endDate = event.end_datetime ? new Date(event.end_datetime) : null
  const imageUrl = getImageUrl(event, '800x600')
  const imageUrl400 = getImageUrl(event, '400x300')

  return (
    <article class="event-page">
      {imageUrl && (
        <div class="event-hero">
          <img
            src={imageUrl}
            srcset={`${imageUrl400} 400w, ${imageUrl} 800w`}
            sizes="(max-width: 600px) 400px, 800px"
            alt={event.title}
          />
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
            <time class="event-datetime" dateTime={event.start_datetime}>
              {format(startDate, 'EEEE, MMMM d, yyyy · h:mm a')}
              {endDate && ` - ${format(endDate, 'h:mm a')}`}
            </time>
            {event.recurrence_rule && (
              <p class="event-recurrence">
                {recurrenceLabel(event.recurrence_rule)}
              </p>
            )}
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
            <div ref={descriptionRef} dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(marked.parse(event.description) as string) }} />
          </section>
        )}

        <footer class="event-actions">
          <a href={`/ics/event/${event.id}`} class="btn" data-umami-event="event-ical-download">
            Download .ics
          </a>

          {isModerator && (
            <div class="admin-actions">
              <a href={`/edit/${parseVirtualId(event.id)?.baseId || event.id}`} class="btn btn-secondary" data-umami-event="event-edit-click">
                Edit
              </a>
              {parseVirtualId(event.id) && (
                <button
                  class="btn btn-danger"
                  data-umami-event="event-cancel-occurrence"
                  onClick={() => {
                    const parsed = parseVirtualId(event.id)
                    if (!parsed) return
                    setConfirmDialog({
                      message: 'Cancel this occurrence?',
                      onConfirm: async () => {
                        setConfirmDialog(null)
                        const dateStr = `${parsed.date.slice(0, 4)}-${parsed.date.slice(4, 6)}-${parsed.date.slice(6, 8)}`
                        try {
                          await fetch(`/api/events/${parsed.baseId}/cancel-occurrence`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ date: dateStr }),
                          })
                          route('/')
                        } catch {
                          alert('Failed to cancel occurrence')
                        }
                      },
                    })
                  }}
                >
                  Cancel This Occurrence
                </button>
              )}
              {event.status === 'pending' && (
                <button
                  class="btn btn-primary"
                  onClick={() => handleStatusChange('published')}
                  disabled={actionLoading}
                  data-umami-event="event-approve"
                >
                  Approve
                </button>
              )}
              {event.status === 'published' && (
                <button
                  class="btn btn-secondary"
                  onClick={() => handleStatusChange('cancelled')}
                  disabled={actionLoading}
                  data-umami-event="event-cancel"
                >
                  Cancel Event
                </button>
              )}
              {event.status === 'cancelled' && (
                <button
                  class="btn btn-primary"
                  onClick={() => handleStatusChange('published')}
                  disabled={actionLoading}
                  data-umami-event="event-republish"
                >
                  Republish
                </button>
              )}
              <button
                class="btn btn-danger"
                onClick={handleDelete}
                disabled={actionLoading}
                data-umami-event="event-delete"
              >
                Delete
              </button>
            </div>
          )}
        </footer>
      </div>
      {confirmDialog && (
        <ConfirmDialog
          message={confirmDialog.message}
          onConfirm={() => { confirmDialog.onConfirm(); setConfirmDialog(null) }}
          onCancel={() => setConfirmDialog(null)}
        />
      )}
    </article>
  )
}
