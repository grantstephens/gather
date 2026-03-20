import { useState } from 'preact/hooks'
import { route } from 'preact-router'
import { pb, Place, Tag } from '../lib/pocketbase'
import { PlaceSearch } from '../components/PlaceSearch'
import { TagPicker } from '../components/TagPicker'
import { MarkdownEditor } from '../components/MarkdownEditor'
import './Submit.css'

interface Props {
  path?: string
}

export function Submit(_props: Props) {
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
  const [email, setEmail] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const isLoggedIn = pb.authStore.isValid

  const handleImageChange = (e: Event) => {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (file) {
      setImage(file)
      const reader = new FileReader()
      reader.onload = () => setImagePreview(reader.result as string)
      reader.readAsDataURL(file)
    }
  }

  const removeImage = () => {
    setImage(null)
    setImagePreview(null)
  }

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    setError(null)

    // Validation
    if (!title.trim()) {
      setError('Title is required')
      return
    }
    if (!startDate || !startTime) {
      setError('Start date and time are required')
      return
    }
    if (!isLoggedIn && !email.trim()) {
      setError('Email is required for anonymous submissions')
      return
    }

    setSubmitting(true)

    try {
      const formData = new FormData()
      formData.append('title', title.trim())
      formData.append('description', description.trim())
      formData.append('start_datetime', new Date(`${startDate}T${startTime}`).toISOString())

      if (endDate && endTime) {
        formData.append('end_datetime', new Date(`${endDate}T${endTime}`).toISOString())
      }

      if (place) {
        formData.append('place', place.id)
      }

      tags.forEach((t) => formData.append('tags', t.id))

      if (image) {
        formData.append('image', image)
      }

      if (!isLoggedIn && email) {
        formData.append('author_email', email.trim())
      }

      // Set status based on auth state
      formData.append('status', isLoggedIn ? 'published' : 'pending')

      // If logged in, set author
      if (isLoggedIn) {
        formData.append('author', pb.authStore.model?.id)
      }

      const created = await pb.collection('events').create(formData)
      route(`/event/${created.id}`)
    } catch (err) {
      console.error('Failed to create event:', err)
      setError(err instanceof Error ? err.message : 'Failed to create event')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div class="submit-page">
      <h1>Submit an Event</h1>

      {!isLoggedIn && (
        <div class="submit-notice">
          You are submitting anonymously. Your event will be reviewed before being published.{' '}
          <a href="/login">Log in</a> to publish immediately.
        </div>
      )}

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

        {!isLoggedIn && (
          <div class="form-group">
            <label for="email">Your Email *</label>
            <input
              type="email"
              id="email"
              value={email}
              onInput={(e) => setEmail((e.target as HTMLInputElement).value)}
              placeholder="We'll notify you when your event is approved"
              required
            />
            <p class="form-hint">Your email won't be displayed publicly.</p>
          </div>
        )}

        <div class="form-actions">
          <button type="submit" class="btn btn-primary btn-large" disabled={submitting}>
            {submitting ? 'Submitting...' : isLoggedIn ? 'Publish Event' : 'Submit for Review'}
          </button>
        </div>
      </form>
    </div>
  )
}
