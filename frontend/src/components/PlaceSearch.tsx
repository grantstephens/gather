import { useState, useCallback, useRef, useEffect } from 'preact/hooks'
import { pb, Place } from '../lib/pocketbase'
import './PlaceSearch.css'

interface NominatimResult {
  place_id: number
  osm_id: number
  osm_type: string
  display_name: string
  name?: string
  lat: string
  lon: string
  address?: {
    house_number?: string
    road?: string
    city?: string
    town?: string
    village?: string
    state?: string
    postcode?: string
    country?: string
    country_code?: string
  }
}

interface Props {
  value: Place | null
  onChange: (place: Place | null) => void
}

export function PlaceSearch({ value, onChange }: Props) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<NominatimResult[]>([])
  const [loading, setLoading] = useState(false)
  const [showResults, setShowResults] = useState(false)
  const [showManual, setShowManual] = useState(false)
  const [manualName, setManualName] = useState('')
  const [manualAddress, setManualAddress] = useState('')
  const [manualLat, setManualLat] = useState('')
  const [manualLon, setManualLon] = useState('')
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()
  const containerRef = useRef<HTMLDivElement>(null)

  const search = useCallback(async (q: string) => {
    if (q.length < 3) {
      setResults([])
      return
    }
    setLoading(true)
    try {
      const response = await fetch(
        `https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(q)}&addressdetails=1&limit=5`,
        { headers: { 'User-Agent': 'Gather Community Calendar' } }
      )
      const data = await response.json()
      setResults(data)
    } catch {
      setResults([])
    } finally {
      setLoading(false)
    }
  }, [])

  const handleInput = (e: Event) => {
    const val = (e.target as HTMLInputElement).value
    setQuery(val)
    setShowResults(true)

    if (debounceRef.current) {
      clearTimeout(debounceRef.current)
    }
    debounceRef.current = setTimeout(() => search(val), 300)
  }

  const selectResult = async (result: NominatimResult) => {
    setShowResults(false)
    setLoading(true)

    try {
      // Check if place exists in our DB by osm_id
      const existing = await pb.collection('places').getList<Place>(1, 1, {
        filter: `osm_id = ${result.osm_id}`,
      })

      if (existing.items.length > 0) {
        onChange(existing.items[0])
        setQuery(existing.items[0].name)
      } else {
        // Create new place from OSM data
        const address = result.address
        const addressParts: string[] = []
        if (address?.house_number && address?.road) {
          addressParts.push(`${address.house_number} ${address.road}`)
        } else if (address?.road) {
          addressParts.push(address.road)
        }
        const city = address?.city || address?.town || address?.village
        if (city) addressParts.push(city)
        if (address?.state) addressParts.push(address.state)
        if (address?.postcode) addressParts.push(address.postcode)

        const newPlace = await pb.collection('places').create<Place>({
          osm_id: result.osm_id,
          osm_type: result.osm_type,
          name: result.name || result.display_name.split(',')[0],
          address: addressParts.join(', '),
          latitude: parseFloat(result.lat),
          longitude: parseFloat(result.lon),
          city: city,
          country_code: address?.country_code?.toUpperCase(),
          osm_data: result,
        })

        onChange(newPlace)
        setQuery(newPlace.name)
      }
    } catch (err) {
      console.error('Failed to create/find place:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleManualSubmit = async (e: Event) => {
    e.preventDefault()
    if (!manualName || !manualLat || !manualLon) return

    setLoading(true)
    try {
      const newPlace = await pb.collection('places').create<Place>({
        name: manualName,
        address: manualAddress || undefined,
        latitude: parseFloat(manualLat),
        longitude: parseFloat(manualLon),
      })

      onChange(newPlace)
      setQuery(newPlace.name)
      setShowManual(false)
      setManualName('')
      setManualAddress('')
      setManualLat('')
      setManualLon('')
    } catch (err) {
      console.error('Failed to create place:', err)
    } finally {
      setLoading(false)
    }
  }

  const clearSelection = () => {
    onChange(null)
    setQuery('')
    setResults([])
  }

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setShowResults(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  if (showManual) {
    return (
      <div class="place-search">
        <form class="manual-place-form" onSubmit={handleManualSubmit}>
          <h4>Add Place Manually</h4>
          <div class="form-group">
            <label>Name *</label>
            <input
              type="text"
              value={manualName}
              onInput={(e) => setManualName((e.target as HTMLInputElement).value)}
              required
            />
          </div>
          <div class="form-group">
            <label>Address</label>
            <input
              type="text"
              value={manualAddress}
              onInput={(e) => setManualAddress((e.target as HTMLInputElement).value)}
            />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label>Latitude *</label>
              <input
                type="number"
                step="any"
                value={manualLat}
                onInput={(e) => setManualLat((e.target as HTMLInputElement).value)}
                required
              />
            </div>
            <div class="form-group">
              <label>Longitude *</label>
              <input
                type="number"
                step="any"
                value={manualLon}
                onInput={(e) => setManualLon((e.target as HTMLInputElement).value)}
                required
              />
            </div>
          </div>
          <div class="manual-actions">
            <button type="button" class="btn btn-secondary" onClick={() => setShowManual(false)}>
              Cancel
            </button>
            <button type="submit" class="btn btn-primary" disabled={loading}>
              {loading ? 'Saving...' : 'Add Place'}
            </button>
          </div>
        </form>
      </div>
    )
  }

  return (
    <div class="place-search" ref={containerRef}>
      {value ? (
        <div class="place-selected">
          <div class="place-info">
            <span class="place-name">{value.name}</span>
            {value.address && <span class="place-address">{value.address}</span>}
          </div>
          <button type="button" class="place-clear" onClick={clearSelection}>
            &times;
          </button>
        </div>
      ) : (
        <>
          <input
            type="text"
            class="place-input"
            placeholder="Search for a place..."
            value={query}
            onInput={handleInput}
            onFocus={() => setShowResults(true)}
          />
          {loading && <div class="place-loading">Searching...</div>}
          {showResults && results.length > 0 && (
            <ul class="place-results">
              {results.map((result) => (
                <li key={result.place_id}>
                  <button type="button" onClick={() => selectResult(result)}>
                    <span class="result-name">
                      {result.name || result.display_name.split(',')[0]}
                    </span>
                    <span class="result-address">{result.display_name}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
          {showResults && query.length >= 3 && !loading && results.length === 0 && (
            <div class="place-no-results">
              No places found.{' '}
              <button type="button" class="link" onClick={() => setShowManual(true)}>
                Add manually
              </button>
            </div>
          )}
          {!value && (
            <button type="button" class="place-manual-link" onClick={() => setShowManual(true)}>
              Can't find your place? Add it manually
            </button>
          )}
        </>
      )}
    </div>
  )
}
