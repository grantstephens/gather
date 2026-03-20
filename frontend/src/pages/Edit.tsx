import { useState, useEffect } from 'preact/hooks'
import { route } from 'preact-router'
import { pb, Place, Tag, Event as EventType, canModerate, getImageUrl } from '../lib/pocketbase'
import { PlaceSearch } from '../components/PlaceSearch'
import { TagPicker } from '../components/TagPicker'
import { MarkdownEditor } from '../components/MarkdownEditor'
import './Submit.css'

interface Props {
  path?: string
  id?: string
}

export function Edit({ id }: Props) {
  const [eventData, setEventData] = useState<EventType | null>(null)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [startDate, setStartDate] = useState('')
  const [startTime, setStartTime] = useState('')
  const [endDate, setEndDate] = useState('')
  const [endTime, setEndTime] = useState('')
  const [place, setPlace] = useState<Place | null>(null)
  const [tags, setTags] = useState<Tag[]>([])
  const [image, setImage] = useState<File | null>(null)
  const [imagePreview, setImagePreview] = useState<string | null>(null)
  const [existingImage, setExistingImage] = useState<string | null>(null)
  const [removeExistingImage, setRemoveExistingImage] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!canModerate()) {
      route('/login')
      return
    }

    if (!id) return

    async function load() {
      try {
        const loadedEvent = await pb.collection('events').getOne<EventType>(id!, {
          expand: 'place,tags',
        })
        setEventData(loadedEvent)
        setTitle(loadedEvent.title)
        setDescription(loadedEvent.description || '')

        // Parse dates
        const start = new Date(loadedEvent.start_datetime)
        setStartDate(start.toISOString().split('T')[0])
        setStartTime(start.toTimeString().slice(0, 5))

        if (loadedEvent.end_datetime) {
          const end = new Date(loadedEvent.end_datetime)
          setEndDate(end.toISOString().split('T')[0])
          setEndTime(end.toTimeString().slice(0, 5))
        }

        if (loadedEvent.expand?.place) {
          setPlace(loadedEvent.expand.place)
        }

        if (loadedEvent.expand?.tags) {
          setTags(loadedEvent.expand.tags)
        }

        if (loadedEvent.image) {
          setExistingImage(getImageUrl(loadedEvent, '400x300') || null)
        }
      } catch (err) {
        setError('Failed to load event')
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [id])

  const handleImageChange = (e: globalThis.Event) => {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (file) {
      setImage(file)
      setRemoveExistingImage(true)
      const reader = new FileReader()
      reader.onload = () => setImagePreview(reader.result as string)
      reader.readAsDataURL(file)
    }
  }

  const removeImage = () => {
    setImage(null)
    setImagePreview(null)
    setExistingImage(null)
    setRemoveExistingImage(true)
  }

  const handleSubmit = async (e: globalThis.Event) => {
    e.preventDefault()
    if (!eventData) return
    setError(null)

    if (!title.trim()) {
      setError('Title is required')
      return
    }
    if (!startDate || !startTime) {
      setError('Start date and time are required')
      return
    }

    setSubmitting(true)

    try {
      // Use FormData only if we have a new image to upload
      if (image) {
        const formData = new FormData()
        formData.append('title', title.trim())
        formData.append('description', description.trim())
        formData.append('start_datetime', new Date(`${startDate}T${startTime}`).toISOString())
        formData.append('end_datetime', endDate && endTime ? new Date(`${endDate}T${endTime}`).toISOString() : '')
        formData.append('place', place?.id || '')
        tags.forEach((t) => formData.append('tags', t.id))
        formData.append('image', image)
        await pb.collection('events').update(eventData.id, formData)
      } else {
        // Use JSON for updates without new images (more reliable)
        const updateData: Record<string, unknown> = {
          title: title.trim(),
          description: description.trim(),
          start_datetime: new Date(`${startDate}T${startTime}`).toISOString(),
          end_datetime: endDate && endTime ? new Date(`${endDate}T${endTime}`).toISOString() : '',
          place: place?.id || '',
          tags: tags.map(t => t.id),
        }
        // Clear image if user removed it
        if (removeExistingImage) {
          updateData.image = null
        }
        await pb.collection('events').update(eventData.id, updateData)
      }
      route(`/event/${eventData.id}`)
    } catch (err) {
      console.error('Failed to update event:', err)
      setError(err instanceof Error ? err.message : 'Failed to update event')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return <div class="loading">Loading...</div>
  }

  if (!eventData) {
    return <div class="error">{error || 'Event not found'}</div>
  }

  return (
    <div class="submit-page">
      <h1>Edit Event</h1>

      {error && <div class="submit-error">{error}</div>}

      <form class="submit-form" onSubmit={handleSubmit}>
        <div class="form-group">
          <label for="title">Event Title *</label>
          <input
            type="text"
            id="title"
            value={title}
            onInput={(e) => setTitle((e.target as HTMLInputElement).value)}
            placeholder="What's happening?"
            required
          />
        </div>

        <div class="form-group">
          <label>Description</label>
          <MarkdownEditor
            value={description}
            onChange={setDescription}
            placeholder="Tell people more about your event..."
          />
        </div>

        <div class="form-row">
          <div class="form-group">
            <label for="startDate">Start Date *</label>
            <input
              type="date"
              id="startDate"
              value={startDate}
              onInput={(e) => setStartDate((e.target as HTMLInputElement).value)}
              required
            />
          </div>
          <div class="form-group">
            <label for="startTime">Start Time *</label>
            <input
              type="time"
              id="startTime"
              value={startTime}
              onInput={(e) => setStartTime((e.target as HTMLInputElement).value)}
              required
            />
          </div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label for="endDate">End Date</label>
            <input
              type="date"
              id="endDate"
              value={endDate}
              onInput={(e) => setEndDate((e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="form-group">
            <label for="endTime">End Time</label>
            <input
              type="time"
              id="endTime"
              value={endTime}
              onInput={(e) => setEndTime((e.target as HTMLInputElement).value)}
            />
          </div>
        </div>

        <div class="form-group">
          <label>Location</label>
          <PlaceSearch value={place} onChange={setPlace} />
        </div>

        <div class="form-group">
          <label>Tags</label>
          <TagPicker value={tags} onChange={setTags} />
        </div>

        <div class="form-group">
          <label>Event Image</label>
          {imagePreview ? (
            <div class="image-preview">
              <img src={imagePreview} alt="Preview" />
              <button type="button" class="image-remove" onClick={removeImage}>
                Remove
              </button>
            </div>
          ) : existingImage ? (
            <div class="image-preview">
              <img src={existingImage} alt="Current" />
              <button type="button" class="image-remove" onClick={removeImage}>
                Remove
              </button>
            </div>
          ) : (
            <div class="image-upload">
              <input
                type="file"
                id="image"
                accept="image/*"
                onChange={handleImageChange}
              />
              <label for="image" class="image-upload-label">
                <span>Choose an image</span>
                <span class="image-upload-hint">Images auto-converted to WebP (max 20MB, no animated GIFs)</span>
              </label>
            </div>
          )}
        </div>

        <div class="form-actions">
          <button type="submit" class="btn btn-primary btn-large" disabled={submitting}>
            {submitting ? 'Saving...' : 'Save Changes'}
          </button>
          <a href={`/event/${eventData.id}`} class="btn btn-secondary btn-large">
            Cancel
          </a>
        </div>
      </form>
    </div>
  )
}
