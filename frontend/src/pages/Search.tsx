import { useEffect, useState } from 'preact/hooks'
import { Event, Place, Tag } from '../lib/pocketbase'
import { EventTimeline } from '../components/EventTimeline'
import { tagStyle } from '../lib/color'
import { SkeletonTimeline } from '../components/Skeleton'
import './Search.css'

interface Props {
  path?: string
}

export function Search(_props: Props) {
  const [query, setQuery] = useState('')
  const [events, setEvents] = useState<Event[]>([])
  const [places, setPlaces] = useState<Place[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(false)
  const [searched, setSearched] = useState(false)

  // Read query from URL
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const q = params.get('q') || ''
    setQuery(q)
  }, [])

  // Search when query changes
  useEffect(() => {
    if (!query.trim()) {
      setEvents([])
      setPlaces([])
      setTags([])
      setSearched(false)
      return
    }

    setLoading(true)
    setSearched(true)

    fetch(`/api/search?q=${encodeURIComponent(query.trim())}&limit=20`)
      .then(r => r.json())
      .then((data: { events: Event[]; places: Place[]; tags: Tag[] }) => {
        setEvents(data.events)
        setPlaces(data.places)
        setTags(data.tags)
      })
      .catch(() => {
        setEvents([])
        setPlaces([])
        setTags([])
      })
      .finally(() => setLoading(false))
  }, [query])

  const totalResults = events.length + places.length + tags.length

  if (!searched && !loading) {
    return (
      <div class="search-page">
        <header class="search-header">
          <h1>Search</h1>
          <p class="search-query">Type in the search bar above to find events, places, and tags.</p>
        </header>
      </div>
    )
  }

  if (loading) {
    return (
      <div class="search-page">
        <header class="search-header">
          <h1>Search results</h1>
          <p class="search-query">for "{query}"</p>
        </header>
        <SkeletonTimeline />
      </div>
    )
  }

  return (
    <div class="search-page">
      <header class="search-header">
        <h1>Search results</h1>
        <p class="search-query">
          {totalResults === 0
            ? `No results for "${query}"`
            : `${totalResults} result${totalResults !== 1 ? 's' : ''} for "${query}"`}
        </p>
      </header>

      {tags.length > 0 && (
        <section class="search-section">
          <h2 class="search-section-title">Tags ({tags.length})</h2>
          <div class="search-tags">
            {tags.map(tag => (
              <a key={tag.id} href={`/tag/${tag.name}`} class="tag" style={tagStyle(tag.color)}>
                {tag.name}
              </a>
            ))}
          </div>
        </section>
      )}

      {places.length > 0 && (
        <section class="search-section">
          <h2 class="search-section-title">Places ({places.length})</h2>
          <div class="search-places">
            {places.map(place => (
              <a key={place.id} href={`/place/${place.id}`} class="search-place-link">
                <span class="search-place-name">{place.name}</span>
                {place.city && <span class="search-place-city">{place.city}</span>}
              </a>
            ))}
          </div>
        </section>
      )}

      {events.length > 0 && (
        <section class="search-section">
          <h2 class="search-section-title">Events ({events.length})</h2>
          <EventTimeline events={events} />
        </section>
      )}
    </div>
  )
}
