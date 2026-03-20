import { useEffect, useState } from 'preact/hooks'
import { route } from 'preact-router'
import { pb, Event, Place, Tag, canModerate, eventPath } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import { SettingsForm } from '../components/SettingsForm'
import './Admin.css'

interface Props {
  path?: string
}

type TabType = 'pending-events' | 'pending-places' | 'pending-tags' | 'all-events' | 'settings'

export function Admin(_props: Props) {
  const [pendingEvents, setPendingEvents] = useState<Event[]>([])
  const [allEvents, setAllEvents] = useState<Event[]>([])
  const [pendingPlaces, setPendingPlaces] = useState<Place[]>([])
  const [pendingTags, setPendingTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<TabType>('pending-events')

  useEffect(() => {
    if (!canModerate()) {
      route('/login')
      return
    }

    let cancelled = false

    async function load() {
      try {
        // Fetch sequentially to avoid race conditions
        const pending = await pb.collection('events').getList<Event>(1, 50, {
          filter: 'status = "pending"',
          sort: '-start_datetime',
          expand: 'place,tags',
        })
        if (cancelled) return

        const all = await pb.collection('events').getList<Event>(1, 50, {
          sort: '-start_datetime',
          expand: 'place,tags',
        })
        if (cancelled) return

        const places = await pb.collection('places').getList<Place>(1, 50, {
          filter: 'status = "pending"',
        })
        if (cancelled) return

        const tags = await pb.collection('tags').getList<Tag>(1, 50, {
          filter: 'status = "pending"',
        })
        if (cancelled) return

        setPendingEvents(pending.items)
        setAllEvents(all.items)
        setPendingPlaces(places.items)
        setPendingTags(tags.items)
      } catch (err) {
        if (!cancelled) {
          console.error('Failed to load data:', err)
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    load()

    return () => {
      cancelled = true
    }
  }, [])

  const checkEventDependencies = async (event: Event): Promise<string | null> => {
    // Check if place is pending
    if (event.place) {
      try {
        const place = await pb.collection('places').getOne<Place>(event.place)
        if (place.status === 'pending') {
          return `Cannot approve: Place "${place.name}" is still pending approval`
        }
      } catch {
        // Place might not exist or we don't have access
      }
    }

    // Check if any tags are pending
    if (event.tags && event.tags.length > 0) {
      for (const tagId of event.tags) {
        try {
          const tag = await pb.collection('tags').getOne<Tag>(tagId)
          if (tag.status === 'pending') {
            return `Cannot approve: Tag "${tag.name}" is still pending approval`
          }
        } catch {
          // Tag might not exist or we don't have access
        }
      }
    }

    return null
  }

  const handleApproveEvent = async (eventId: string) => {
    const event = pendingEvents.find(e => e.id === eventId)
    if (!event) return

    // Check dependencies first
    const dependencyError = await checkEventDependencies(event)
    if (dependencyError) {
      alert(dependencyError)
      return
    }

    try {
      await pb.collection('events').update(eventId, { status: 'published' })
      setPendingEvents(prev => prev.filter(e => e.id !== eventId))
      setAllEvents(prev => prev.map(e => e.id === eventId ? { ...e, status: 'published' } : e))
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to approve event'
      alert(message)
    }
  }

  const handleRejectEvent = async (eventId: string) => {
    if (!confirm('Are you sure you want to delete this event?')) return
    try {
      await pb.collection('events').delete(eventId)
      setPendingEvents(prev => prev.filter(e => e.id !== eventId))
      setAllEvents(prev => prev.filter(e => e.id !== eventId))
    } catch (err) {
      alert('Failed to delete event')
    }
  }

  const handleApprovePlace = async (placeId: string) => {
    try {
      await pb.collection('places').update(placeId, { status: 'approved' })
      setPendingPlaces(prev => prev.filter(p => p.id !== placeId))
    } catch (err) {
      alert('Failed to approve place')
    }
  }

  const handleRejectPlace = async (placeId: string) => {
    if (!confirm('Are you sure you want to delete this place? Events using it will need to be updated.')) return
    try {
      await pb.collection('places').delete(placeId)
      setPendingPlaces(prev => prev.filter(p => p.id !== placeId))
    } catch (err) {
      alert('Failed to delete place')
    }
  }

  const handleApproveTag = async (tagId: string) => {
    try {
      await pb.collection('tags').update(tagId, { status: 'approved' })
      setPendingTags(prev => prev.filter(t => t.id !== tagId))
    } catch (err) {
      alert('Failed to approve tag')
    }
  }

  const handleRejectTag = async (tagId: string) => {
    if (!confirm('Are you sure you want to delete this tag? Events using it will need to be updated.')) return
    try {
      await pb.collection('tags').delete(tagId)
      setPendingTags(prev => prev.filter(t => t.id !== tagId))
    } catch (err) {
      alert('Failed to delete tag')
    }
  }

  if (loading) {
    return <div class="loading">Loading...</div>
  }

  const totalPending = pendingEvents.length + pendingPlaces.length + pendingTags.length

  return (
    <div class="admin-page">
      <h1>Content Moderation</h1>

      <div class="admin-tabs">
        <button
          class={`tab ${activeTab === 'pending-events' ? 'active' : ''}`}
          onClick={() => setActiveTab('pending-events')}
        >
          Events ({pendingEvents.length})
        </button>
        <button
          class={`tab ${activeTab === 'pending-places' ? 'active' : ''}`}
          onClick={() => setActiveTab('pending-places')}
        >
          Places ({pendingPlaces.length})
        </button>
        <button
          class={`tab ${activeTab === 'pending-tags' ? 'active' : ''}`}
          onClick={() => setActiveTab('pending-tags')}
        >
          Tags ({pendingTags.length})
        </button>
        <button
          class={`tab ${activeTab === 'all-events' ? 'active' : ''}`}
          onClick={() => setActiveTab('all-events')}
        >
          All Events ({allEvents.length})
        </button>
        <button
          class={`tab ${activeTab === 'settings' ? 'active' : ''}`}
          onClick={() => setActiveTab('settings')}
        >
          Settings
        </button>
      </div>

      {totalPending > 0 && activeTab.startsWith('pending') && (
        <p class="moderation-tip">
          Approve places and tags before approving events that use them.
        </p>
      )}

      {activeTab === 'pending-events' && (
        <div class="events-list">
          {pendingEvents.length === 0 ? (
            <p class="no-events">No pending events to review.</p>
          ) : (
            pendingEvents.map(event => (
              <div key={event.id} class="admin-event-card">
                <EventCard event={event} />
                <div class="admin-event-actions">
                  <button class="btn btn-success" onClick={() => handleApproveEvent(event.id)}>
                    Approve
                  </button>
                  <a href={eventPath(event)} class="btn btn-secondary">
                    View
                  </a>
                  <button class="btn btn-danger" onClick={() => handleRejectEvent(event.id)}>
                    Delete
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {activeTab === 'pending-places' && (
        <div class="items-list">
          {pendingPlaces.length === 0 ? (
            <p class="no-events">No pending places to review.</p>
          ) : (
            pendingPlaces.map(place => (
              <div key={place.id} class="admin-item-card">
                <div class="item-info">
                  <h3>{place.name}</h3>
                  {place.address && <p class="item-detail">{place.address}</p>}
                  {place.city && <p class="item-detail">{place.city}</p>}
                </div>
                <div class="admin-event-actions">
                  <button class="btn btn-success" onClick={() => handleApprovePlace(place.id)}>
                    Approve
                  </button>
                  <button class="btn btn-danger" onClick={() => handleRejectPlace(place.id)}>
                    Delete
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {activeTab === 'pending-tags' && (
        <div class="items-list">
          {pendingTags.length === 0 ? (
            <p class="no-events">No pending tags to review.</p>
          ) : (
            pendingTags.map(tag => (
              <div key={tag.id} class="admin-item-card">
                <div class="item-info">
                  <span
                    class="tag-preview"
                    style={tag.color ? { backgroundColor: tag.color, color: 'white' } : undefined}
                  >
                    {tag.name}
                  </span>
                </div>
                <div class="admin-event-actions">
                  <button class="btn btn-success" onClick={() => handleApproveTag(tag.id)}>
                    Approve
                  </button>
                  <button class="btn btn-danger" onClick={() => handleRejectTag(tag.id)}>
                    Delete
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {activeTab === 'all-events' && (
        <div class="events-list">
          {allEvents.length === 0 ? (
            <p class="no-events">No events found.</p>
          ) : (
            allEvents.map(event => (
              <div key={event.id} class="admin-event-card">
                <div class="event-status-row">
                  <span class={`status-badge status-${event.status}`}>{event.status}</span>
                </div>
                <EventCard event={event} />
                <div class="admin-event-actions">
                  <a href={eventPath(event)} class="btn btn-secondary">
                    View
                  </a>
                  <a href={`/edit/${event.id}`} class="btn btn-secondary">
                    Edit
                  </a>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {activeTab === 'settings' && <SettingsForm />}
    </div>
  )
}
