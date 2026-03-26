import { useState, useEffect, useRef, useCallback } from 'preact/hooks'
import { route } from 'preact-router'
import { Event, Place, Tag, eventPath } from '../lib/pocketbase'
import { tagStyle } from '../lib/color'
import './SearchBar.css'

interface SearchResults {
  events: Event[]
  places: Place[]
  tags: Tag[]
}

const EMPTY: SearchResults = { events: [], places: [], tags: [] }

export function SearchBar() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResults>(EMPTY)
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const wrapperRef = useRef<HTMLDivElement>(null)

  // Debounced search
  useEffect(() => {
    const q = query.trim()
    if (q.length < 2) {
      setResults(EMPTY)
      setOpen(false)
      return
    }

    setLoading(true)
    const controller = new AbortController()
    const timer = setTimeout(() => {
      fetch(`/api/search?q=${encodeURIComponent(q)}&limit=3`, { signal: controller.signal })
        .then(r => r.json())
        .then((data: SearchResults) => {
          setResults(data)
          setOpen(true)
        })
        .catch(() => setResults(EMPTY))
        .finally(() => setLoading(false))
    }, 250)

    return () => { clearTimeout(timer); controller.abort() }
  }, [query])

  // Close on click outside
  useEffect(() => {
    if (!open) return
    const handleClick = (e: MouseEvent) => {
      if (!wrapperRef.current?.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('click', handleClick)
    return () => document.removeEventListener('click', handleClick)
  }, [open])

  // "/" keyboard shortcut to focus
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === '/' && !e.ctrlKey && !e.metaKey && !e.altKey) {
        const tag = (e.target as HTMLElement).tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
        e.preventDefault()
        inputRef.current?.focus()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  const handleSubmit = useCallback((e: globalThis.Event) => {
    e.preventDefault()
    const q = query.trim()
    if (q) {
      route(`/search?q=${encodeURIComponent(q)}`)
      setOpen(false)
      inputRef.current?.blur()
    }
  }, [query])

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      setOpen(false)
      inputRef.current?.blur()
    }
  }

  const navigate = (path: string) => {
    setOpen(false)
    setQuery('')
    route(path)
  }

  const total = results.events.length + results.places.length + results.tags.length
  const showDropdown = open && total > 0

  return (
    <div class="nav-search" ref={wrapperRef}>
      <form onSubmit={handleSubmit}>
        <svg class="nav-search-icon" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <circle cx="7" cy="7" r="4.5" />
          <line x1="10.5" y1="10.5" x2="14" y2="14" />
        </svg>
        <input
          ref={inputRef}
          type="search"
          class="nav-search-input"
          placeholder="Search..."
          value={query}
          onInput={(e) => setQuery((e.target as HTMLInputElement).value)}
          onFocus={() => { if (total > 0) setOpen(true) }}
          onKeyDown={handleKeyDown}
        />
        {loading && <span class="nav-search-spinner" />}
      </form>

      {showDropdown && (
        <div class="search-dropdown">
          {results.tags.map(tag => (
            <button
              key={tag.id}
              class="search-dropdown-item search-dropdown-tag"
              onClick={() => navigate(`/tag/${tag.name}`)}
            >
              <span class="search-dropdown-type">Tag</span>
              <span class="tag tag--small" style={tagStyle(tag.color)}>{tag.name}</span>
            </button>
          ))}

          {results.places.map(place => (
            <button
              key={place.id}
              class="search-dropdown-item"
              onClick={() => navigate(`/place/${place.id}`)}
            >
              <span class="search-dropdown-type">Place</span>
              <span class="search-dropdown-label">{place.name}</span>
              {place.city && <span class="search-dropdown-meta">{place.city}</span>}
            </button>
          ))}

          {results.events.map(event => (
            <button
              key={event.id}
              class="search-dropdown-item"
              onClick={() => navigate(eventPath(event))}
            >
              <span class="search-dropdown-type">Event</span>
              <span class="search-dropdown-label">{event.title}</span>
            </button>
          ))}

          <button
            class="search-dropdown-item search-dropdown-viewall"
            onClick={() => { setOpen(false); route(`/search?q=${encodeURIComponent(query.trim())}`) }}
          >
            View all results
          </button>
        </div>
      )}
    </div>
  )
}
