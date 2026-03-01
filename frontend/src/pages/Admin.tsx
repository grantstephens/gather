import { useEffect, useState } from 'preact/hooks'
import { route } from 'preact-router'
import { pb, Event, canModerate } from '../lib/pocketbase'
import { EventCard } from '../components/EventCard'
import './Admin.css'

interface Props {
  path?: string
}

export function Admin(_props: Props) {
  const [pendingEvents, setPendingEvents] = useState<Event[]>([])
  const [allEvents, setAllEvents] = useState<Event[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'pending' | 'all'>('pending')

  useEffect(() => {
    if (!canModerate()) {
      route('/login')
      return
    }

    async function load() {
      try {
        const [pending, all] = await Promise.all([
          pb.collection('events').getList<Event>(1, 50, {
            filter: 'status = "pending"',
            sort: '-created',
            expand: 'place,tags',
          }),
          pb.collection('events').getList<Event>(1, 50, {
            sort: '-created',
            expand: 'place,tags',
          }),
        ])
        setPendingEvents(pending.items)
        setAllEvents(all.items)
      } catch (err) {
        console.error('Failed to load events:', err)
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [])

  const handleApprove = async (eventId: string) => {
    try {
      await pb.collection('events').update(eventId, { status: 'published' })
      setPendingEvents(prev => prev.filter(e => e.id !== eventId))
      setAllEvents(prev => prev.map(e => e.id === eventId ? { ...e, status: 'published' } : e))
    } catch (err) {
      alert('Failed to approve event')
    }
  }

  const handleReject = async (eventId: string) => {
    if (!confirm('Are you sure you want to delete this event?')) return
    try {
      await pb.collection('events').delete(eventId)
      setPendingEvents(prev => prev.filter(e => e.id !== eventId))
      setAllEvents(prev => prev.filter(e => e.id !== eventId))
    } catch (err) {
      alert('Failed to delete event')
    }
  }

  if (loading) {
    return <div class="loading">Loading...</div>
  }

  return (
    <div class="admin-page">
      <h1>Event Management</h1>

      <div class="admin-tabs">
        <button
          class={`tab ${activeTab === 'pending' ? 'active' : ''}`}
          onClick={() => setActiveTab('pending')}
        >
          Pending ({pendingEvents.length})
        </button>
        <button
          class={`tab ${activeTab === 'all' ? 'active' : ''}`}
          onClick={() => setActiveTab('all')}
        >
          All Events ({allEvents.length})
        </button>
      </div>

      {activeTab === 'pending' && (
        <div class="events-list">
          {pendingEvents.length === 0 ? (
            <p class="no-events">No pending events to review.</p>
          ) : (
            pendingEvents.map(event => (
              <div key={event.id} class="admin-event-card">
                <EventCard event={event} />
                <div class="admin-event-actions">
                  <button class="btn btn-success" onClick={() => handleApprove(event.id)}>
                    Approve
                  </button>
                  <a href={`/event/${event.id}`} class="btn btn-secondary">
                    View
                  </a>
                  <button class="btn btn-danger" onClick={() => handleReject(event.id)}>
                    Delete
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {activeTab === 'all' && (
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
                  <a href={`/event/${event.id}`} class="btn btn-secondary">
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
    </div>
  )
}
