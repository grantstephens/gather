import { useEffect, useState } from 'preact/hooks'
import { route } from 'preact-router'
import { pb, Event, Place, Tag, canModerate, eventPath, isAdmin } from '../lib/pocketbase'
import { tagStyle } from '../lib/color'
import type { PageRecord } from '../lib/pocketbase'
import { MarkdownEditor } from '../components/MarkdownEditor'
import { EventCard } from '../components/EventCard'
import { SettingsForm } from '../components/SettingsForm'
import { SkeletonBlock } from '../components/Skeleton'
import './Admin.css'

interface Props {
  path?: string
}

type TabType = 'pending-events' | 'pending-places' | 'pending-tags' | 'all-events' | 'settings' | 'pages'

const RESERVED_SLUGS = ['submit', 'login', 'admin', 'event', 'tag', 'place', 'edit']

function slugify(title: string): string {
  return title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
}

export function Admin(_props: Props) {
  const [pendingEvents, setPendingEvents] = useState<Event[]>([])
  const [allEvents, setAllEvents] = useState<Event[]>([])
  const [pendingPlaces, setPendingPlaces] = useState<Place[]>([])
  const [pendingTags, setPendingTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<TabType>('pending-events')
  const [pages, setPages] = useState<PageRecord[]>([])
  const [pagesLoaded, setPagesLoaded] = useState(false)
  const [showPageForm, setShowPageForm] = useState(false)
  const [editingPageId, setEditingPageId] = useState<string | null>(null)
  const [pageForm, setPageForm] = useState({
    title: '',
    slug: '',
    content: '',
    show_in_nav: true,
    show_in_footer: true,
  })
  const [pageFormError, setPageFormError] = useState<string | null>(null)
  const [pageSaving, setPageSaving] = useState(false)

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

  useEffect(() => {
    if (activeTab !== 'pages' || !isAdmin() || pagesLoaded) return
    async function loadPages() {
      try {
        const records = await pb.collection('pages').getFullList<PageRecord>({ sort: 'title' })
        setPages(records)
      } catch (err) {
        console.error('Failed to load pages:', err)
      } finally {
        setPagesLoaded(true)
      }
    }
    loadPages()
  }, [activeTab, pagesLoaded])

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

  const handlePageNew = () => {
    setEditingPageId(null)
    setPageForm({ title: '', slug: '', content: '', show_in_nav: true, show_in_footer: true })
    setPageFormError(null)
    setShowPageForm(true)
  }

  const handlePageEdit = (page: PageRecord) => {
    setEditingPageId(page.id)
    setPageForm({
      title: page.title,
      slug: page.slug,
      content: page.content,
      show_in_nav: page.show_in_nav,
      show_in_footer: page.show_in_footer,
    })
    setPageFormError(null)
    setShowPageForm(true)
  }

  const handlePageSave = async () => {
    if (!pageForm.title.trim() || !pageForm.slug.trim()) {
      setPageFormError('Title and slug are required.')
      return
    }
    if (RESERVED_SLUGS.includes(pageForm.slug)) {
      setPageFormError(`"${pageForm.slug}" is a reserved slug and cannot be used.`)
      return
    }
    setPageSaving(true)
    setPageFormError(null)
    try {
      if (editingPageId) {
        const updated = await pb.collection('pages').update<PageRecord>(editingPageId, pageForm)
        setPages(prev => prev.map(p => p.id === editingPageId ? updated : p))
      } else {
        const created = await pb.collection('pages').create<PageRecord>(pageForm)
        setPages(prev => [...prev, created])
      }
      setShowPageForm(false)
    } catch (err: any) {
      setPageFormError(err?.data?.data?.slug?.message || 'Failed to save page.')
    } finally {
      setPageSaving(false)
    }
  }

  const handlePageDelete = async (pageId: string) => {
    if (!confirm('Delete this page? This cannot be undone.')) return
    try {
      await pb.collection('pages').delete(pageId)
      setPages(prev => prev.filter(p => p.id !== pageId))
    } catch {
      alert('Failed to delete page.')
    }
  }

  if (loading) {
    return (
      <div class="admin-page">
        <SkeletonBlock width="200px" height="32px" borderRadius="var(--radius-sm)" />
        <div style={{ marginTop: 'var(--space-6)', display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
          <SkeletonBlock width="100%" height="60px" borderRadius="var(--radius-lg)" />
          <SkeletonBlock width="100%" height="60px" borderRadius="var(--radius-lg)" />
          <SkeletonBlock width="100%" height="60px" borderRadius="var(--radius-lg)" />
        </div>
      </div>
    )
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
        {isAdmin() && (
          <button
            class={`tab ${activeTab === 'pages' ? 'active' : ''}`}
            onClick={() => setActiveTab('pages')}
          >
            Pages
          </button>
        )}
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
                  <button class="btn btn-primary" onClick={() => handleApproveEvent(event.id)}>
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
                  <button class="btn btn-primary" onClick={() => handleApprovePlace(place.id)}>
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
                    style={tagStyle(tag.color)}
                  >
                    {tag.name}
                  </span>
                </div>
                <div class="admin-event-actions">
                  <button class="btn btn-primary" onClick={() => handleApproveTag(tag.id)}>
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

      {activeTab === 'pages' && (
        <div class="pages-admin">
          {!showPageForm ? (
            <>
              <div class="pages-list-header">
                <button class="btn btn-primary" onClick={handlePageNew}>New Page</button>
              </div>
              {pages.length === 0 ? (
                <p class="no-events">No pages yet. Create your first page above.</p>
              ) : (
                <div class="items-list">
                  {pages.map(page => (
                    <div key={page.id} class="admin-item-card">
                      <div class="item-info">
                        <h3>{page.title}</h3>
                        <p class="item-detail">
                          /{page.slug}
                          {' · '}
                          {[page.show_in_nav && 'Nav', page.show_in_footer && 'Footer']
                            .filter(Boolean).join(' · ') || 'Not linked'}
                        </p>
                      </div>
                      <div class="admin-event-actions">
                        <a href={`/${page.slug}`} target="_blank" class="btn btn-secondary">View</a>
                        <button class="btn btn-secondary" onClick={() => handlePageEdit(page)}>Edit</button>
                        <button class="btn btn-danger" onClick={() => handlePageDelete(page.id)}>Delete</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          ) : (
            <div class="page-form">
              <h2>{editingPageId ? 'Edit Page' : 'New Page'}</h2>
              {pageFormError && <div class="error">{pageFormError}</div>}
              <div class="form-group">
                <label for="page-title">Title</label>
                <input
                  type="text"
                  id="page-title"
                  value={pageForm.title}
                  onInput={(e) => {
                    const title = (e.target as HTMLInputElement).value
                    setPageForm(f => ({
                      ...f,
                      title,
                      slug: editingPageId ? f.slug : slugify(title),
                    }))
                  }}
                  disabled={pageSaving}
                  required
                />
              </div>
              <div class="form-group">
                <label for="page-slug">Slug (URL path)</label>
                <input
                  type="text"
                  id="page-slug"
                  value={pageForm.slug}
                  onInput={(e) => setPageForm(f => ({ ...f, slug: (e.target as HTMLInputElement).value }))}
                  disabled={pageSaving}
                  required
                />
                <small>Page will be accessible at /{pageForm.slug}</small>
              </div>
              <div class="form-group">
                <label>Content</label>
                <MarkdownEditor
                  value={pageForm.content}
                  onChange={(content) => setPageForm(f => ({ ...f, content }))}
                />
              </div>
              <div class="form-group">
                <label>
                  <input
                    type="checkbox"
                    checked={pageForm.show_in_nav}
                    onChange={(e) => setPageForm(f => ({ ...f, show_in_nav: (e.target as HTMLInputElement).checked }))}
                    disabled={pageSaving}
                  />
                  {' '}Show in navigation
                </label>
              </div>
              <div class="form-group">
                <label>
                  <input
                    type="checkbox"
                    checked={pageForm.show_in_footer}
                    onChange={(e) => setPageForm(f => ({ ...f, show_in_footer: (e.target as HTMLInputElement).checked }))}
                    disabled={pageSaving}
                  />
                  {' '}Show in footer
                </label>
              </div>
              <div class="form-actions">
                <button type="button" class="btn btn-secondary" onClick={() => setShowPageForm(false)} disabled={pageSaving}>
                  Cancel
                </button>
                <button type="button" class="btn btn-primary" onClick={handlePageSave} disabled={pageSaving}>
                  {pageSaving ? 'Saving...' : 'Save Page'}
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
