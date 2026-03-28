import { useEffect, useState } from 'preact/hooks'
import { route } from 'preact-router'
import { pb, Event, Place, Tag, canModerate, eventPath, isAdmin } from '../lib/pocketbase'
import { tagStyle } from '../lib/color'
import type { PageRecord } from '../lib/pocketbase'
import { MarkdownEditor } from '../components/MarkdownEditor'
import { EventCard } from '../components/EventCard'
import { PlaceSearch } from '../components/PlaceSearch'
import type { NominatimPlaceData } from '../components/PlaceSearch'
import { SettingsForm } from '../components/SettingsForm'
import { SkeletonBlock } from '../components/Skeleton'
import './Admin.css'

interface Props {
  path?: string
}

type TabType = 'events' | 'places' | 'tags' | 'settings' | 'pages'

const RESERVED_SLUGS = ['submit', 'login', 'admin', 'event', 'tag', 'place', 'edit']

function slugify(title: string): string {
  return title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
}

export function Admin(_props: Props) {
  const [allEvents, setAllEvents] = useState<Event[]>([])
  const [allPlaces, setAllPlaces] = useState<Place[]>([])
  const [allTags, setAllTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<TabType>('events')
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
  const [editingPlaceId, setEditingPlaceId] = useState<string | null>(null)
  const [editingTagId, setEditingTagId] = useState<string | null>(null)
  const [tagForm, setTagForm] = useState({ name: '', color: '' })

  useEffect(() => {
    if (!canModerate()) {
      route('/login')
      return
    }

    let cancelled = false

    async function load() {
      try {
        // Fetch sequentially to avoid race conditions
        const events = await pb.collection('events').getFullList<Event>({
          sort: '-start_datetime',
          expand: 'place,tags',
        })
        if (cancelled) return

        const places = await pb.collection('places').getFullList<Place>({
          sort: 'status,name',
        })
        if (cancelled) return

        const tags = await pb.collection('tags').getFullList<Tag>({
          sort: 'status,name',
        })
        if (cancelled) return

        setAllEvents(events)
        setAllPlaces(places)
        setAllTags(tags)
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
        const records = await pb.collection('pages').getFullList<PageRecord>({ sort: 'sort_order,title' })
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
    const event = allEvents.find(e => e.id === eventId)
    if (!event) return

    // Check dependencies first
    const dependencyError = await checkEventDependencies(event)
    if (dependencyError) {
      alert(dependencyError)
      return
    }

    try {
      await pb.collection('events').update(eventId, { status: 'published' })
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
      setAllEvents(prev => prev.filter(e => e.id !== eventId))
    } catch (err) {
      alert('Failed to delete event')
    }
  }

  const handleApprovePlace = async (placeId: string) => {
    try {
      await pb.collection('places').update(placeId, { status: 'approved' })
      setAllPlaces(prev => prev.map(p => p.id === placeId ? { ...p, status: 'approved' } : p))
    } catch (err) {
      alert('Failed to approve place')
    }
  }

  const handleRejectPlace = async (placeId: string) => {
    if (!confirm('Are you sure you want to delete this place? Events using it will need to be updated.')) return
    try {
      await pb.collection('places').delete(placeId)
      setAllPlaces(prev => prev.filter(p => p.id !== placeId))
    } catch (err) {
      alert('Failed to delete place')
    }
  }

  const handleApproveTag = async (tagId: string) => {
    try {
      await pb.collection('tags').update(tagId, { status: 'approved' })
      setAllTags(prev => prev.map(t => t.id === tagId ? { ...t, status: 'approved' } : t))
    } catch (err) {
      alert('Failed to approve tag')
    }
  }

  const handleRejectTag = async (tagId: string) => {
    if (!confirm('Are you sure you want to delete this tag? Events using it will need to be updated.')) return
    try {
      await pb.collection('tags').delete(tagId)
      setAllTags(prev => prev.filter(t => t.id !== tagId))
    } catch (err) {
      alert('Failed to delete tag')
    }
  }

  const handleReplacePlaceWithOSM = async (data: NominatimPlaceData, placeId: string) => {
    try {
      await pb.collection('places').update(placeId, data)
      setAllPlaces(prev => prev.map(p => p.id === placeId ? { ...p, ...data } as Place : p))
      setEditingPlaceId(null)
    } catch {
      alert('Failed to update place')
    }
  }

  const handleAddPlace = (place: Place | null) => {
    if (place) {
      setAllPlaces(prev => [place, ...prev])
    }
  }

  const handleEditTag = (tag: Tag) => {
    setEditingTagId(tag.id)
    setTagForm({ name: tag.name, color: tag.color || '' })
  }

  const handleSaveTag = async (tagId: string) => {
    if (!tagForm.name.trim()) return
    try {
      await pb.collection('tags').update(tagId, tagForm)
      setAllTags(prev => prev.map(t => t.id === tagId ? { ...t, ...tagForm } : t))
      setEditingTagId(null)
    } catch {
      alert('Failed to update tag')
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
        const created = await pb.collection('pages').create<PageRecord>({ ...pageForm, sort_order: pages.length })
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

  const handlePageMove = async (index: number, direction: -1 | 1) => {
    const targetIndex = index + direction
    if (targetIndex < 0 || targetIndex >= pages.length) return

    const newPages = [...pages]
    const [moved] = newPages.splice(index, 1)
    newPages.splice(targetIndex, 0, moved)

    setPages(newPages)

    try {
      await Promise.all(
        newPages.map((p, i) => pb.collection('pages').update(p.id, { sort_order: i }))
      )
      setPages(prev => prev.map((p, i) => ({ ...p, sort_order: i })))
    } catch {
      setPagesLoaded(false) // reload on failure
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

  const pendingEventsCount = allEvents.filter(e => e.status === 'pending').length
  const pendingPlacesCount = allPlaces.filter(p => p.status === 'pending').length
  const pendingTagsCount = allTags.filter(t => t.status === 'pending').length
  const totalPending = pendingEventsCount + pendingPlacesCount + pendingTagsCount

  return (
    <div class="admin-page">
      <h1>Content Moderation</h1>

      <div class="admin-tabs">
        <button
          class={`tab ${activeTab === 'events' ? 'active' : ''}`}
          onClick={() => setActiveTab('events')}
        >
          Events {pendingEventsCount > 0 && `(${pendingEventsCount})`}
        </button>
        <button
          class={`tab ${activeTab === 'places' ? 'active' : ''}`}
          onClick={() => setActiveTab('places')}
        >
          Places {pendingPlacesCount > 0 && `(${pendingPlacesCount})`}
        </button>
        <button
          class={`tab ${activeTab === 'tags' ? 'active' : ''}`}
          onClick={() => setActiveTab('tags')}
        >
          Tags {pendingTagsCount > 0 && `(${pendingTagsCount})`}
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

      {totalPending > 0 && (activeTab === 'events' || activeTab === 'places' || activeTab === 'tags') && (
        <p class="moderation-tip">
          Approve places and tags before approving events that use them.
        </p>
      )}

      {activeTab === 'events' && (
        <div class="events-list">
          {allEvents.length === 0 ? (
            <p class="no-events">No events found.</p>
          ) : (
            [...allEvents].sort((a, b) => {
              if (a.status === 'pending' && b.status !== 'pending') return -1
              if (a.status !== 'pending' && b.status === 'pending') return 1
              return 0
            }).map(event => (
              <div key={event.id} class="admin-event-card">
                <div class="event-status-row">
                  <span class={`status-badge status-${event.status}`}>{event.status}</span>
                </div>
                <EventCard event={event} />
                <div class="admin-event-actions">
                  {event.status === 'pending' && (
                    <button class="btn btn-primary" onClick={() => handleApproveEvent(event.id)}>
                      Approve
                    </button>
                  )}
                  <a href={eventPath(event)} class="btn btn-secondary">
                    View
                  </a>
                  <a href={`/edit/${event.id}`} class="btn btn-secondary">
                    Edit
                  </a>
                  {event.status === 'pending' && (
                    <button class="btn btn-danger" onClick={() => handleRejectEvent(event.id)}>
                      Delete
                    </button>
                  )}
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {activeTab === 'places' && (
        <div>
          <div style={{ marginBottom: 'var(--space-4)' }}>
            <label style={{ display: 'block', marginBottom: 'var(--space-2)', fontWeight: 500 }}>Add a place</label>
            <PlaceSearch value={null} onChange={handleAddPlace} />
          </div>
          <div class="items-list">
            {allPlaces.map(place => (
              <div key={place.id} class="admin-item-card">
                {editingPlaceId === place.id ? (
                  <div style={{ flex: 1 }}>
                    <p class="item-detail" style={{ marginBottom: 'var(--space-2)' }}>
                      Search for a new location to replace <strong>{place.name}</strong>:
                    </p>
                    <PlaceSearch
                      value={null}
                      onChange={() => {}}
                      onRawSelect={(data) => handleReplacePlaceWithOSM(data, place.id)}
                    />
                    <div class="admin-event-actions" style={{ marginTop: 'var(--space-3)' }}>
                      <button class="btn btn-secondary" onClick={() => setEditingPlaceId(null)}>
                        Cancel
                      </button>
                    </div>
                  </div>
                ) : (
                  <>
                    <div class="item-info">
                      <h3>
                        {place.status === 'pending' && <span class="status-badge status-pending">pending</span>}
                        {place.name}
                      </h3>
                      {place.address && <p class="item-detail">{place.address}</p>}
                      {place.city && <p class="item-detail">{place.city}</p>}
                    </div>
                    <div class="admin-event-actions">
                      {place.status === 'pending' && (
                        <button class="btn btn-primary" onClick={() => handleApprovePlace(place.id)}>
                          Approve
                        </button>
                      )}
                      <button class="btn btn-secondary" onClick={() => setEditingPlaceId(place.id)}>
                        Edit
                      </button>
                      <button class="btn btn-danger" onClick={() => handleRejectPlace(place.id)}>
                        Delete
                      </button>
                    </div>
                  </>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'tags' && (
        <div class="items-list">
          {allTags.length === 0 ? (
            <p class="no-events">No tags found.</p>
          ) : (
            allTags.map(tag => (
              <div key={tag.id} class="admin-item-card">
                {editingTagId === tag.id ? (
                  <>
                    <div class="item-info" style={{ flex: 1 }}>
                      <div class="inline-edit-fields">
                        <input
                          type="text"
                          value={tagForm.name}
                          onInput={(e) => setTagForm(f => ({ ...f, name: (e.target as HTMLInputElement).value }))}
                          placeholder="Name"
                        />
                        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--space-2)' }}>
                          <input
                            type="color"
                            value={tagForm.color || '#808080'}
                            onInput={(e) => setTagForm(f => ({ ...f, color: (e.target as HTMLInputElement).value }))}
                            style={{ width: '40px', height: '32px', padding: '2px', cursor: 'pointer' }}
                          />
                          <span class="tag-preview" style={tagStyle(tagForm.color)}>{tagForm.name || 'preview'}</span>
                        </div>
                      </div>
                    </div>
                    <div class="admin-event-actions">
                      <button class="btn btn-primary" onClick={() => handleSaveTag(tag.id)}>Save</button>
                      <button class="btn btn-secondary" onClick={() => setEditingTagId(null)}>Cancel</button>
                    </div>
                  </>
                ) : (
                  <>
                    <div class="item-info">
                      {tag.status === 'pending' && <span class="status-badge status-pending">pending</span>}
                      <span
                        class="tag-preview"
                        style={tagStyle(tag.color)}
                      >
                        {tag.name}
                      </span>
                    </div>
                    <div class="admin-event-actions">
                      {tag.status === 'pending' && (
                        <button class="btn btn-primary" onClick={() => handleApproveTag(tag.id)}>
                          Approve
                        </button>
                      )}
                      <button class="btn btn-secondary" onClick={() => handleEditTag(tag)}>
                        Edit
                      </button>
                      <button class="btn btn-danger" onClick={() => handleRejectTag(tag.id)}>
                        Delete
                      </button>
                    </div>
                  </>
                )}
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
                  {pages.map((page, i) => (
                    <div key={page.id} class="admin-item-card">
                      <div class="page-reorder">
                        <button
                          class="reorder-btn"
                          onClick={() => handlePageMove(i, -1)}
                          disabled={i === 0}
                          aria-label="Move up"
                        >
                          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 2v10"/><path d="M3 6l4-4 4 4"/></svg>
                        </button>
                        <button
                          class="reorder-btn"
                          onClick={() => handlePageMove(i, 1)}
                          disabled={i === pages.length - 1}
                          aria-label="Move down"
                        >
                          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 2v10"/><path d="M3 8l4 4 4-4"/></svg>
                        </button>
                      </div>
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
