import PocketBase from 'pocketbase'

export const pb = new PocketBase('/')

// Types for our collections
export interface Event {
  id: string
  title: string
  description: string
  start_datetime: string
  end_datetime?: string
  place?: string
  online_locations?: string[]
  tags?: string[]
  image?: string
  author?: string
  author_email?: string
  status: 'draft' | 'pending' | 'published' | 'cancelled'
  recurrence_rule?: string
  parent_event?: string
  ap_id?: string
  created: string
  updated: string
  expand?: {
    place?: Place
    tags?: Tag[]
    author?: User
  }
}

export interface Place {
  id: string
  osm_id?: number
  osm_type?: 'node' | 'way' | 'relation'
  name: string
  address?: string
  latitude: number
  longitude: number
  city?: string
  country_code?: string
  osm_data?: Record<string, unknown>
}

export interface Tag {
  id: string
  name: string
  color?: string
}

export interface User {
  id: string
  email: string
  role?: 'user' | 'editor' | 'admin'
  display_name?: string
}

export interface Settings {
  id: string
  instance_name?: string
  instance_description?: string
  allow_anonymous?: boolean
  require_moderation?: boolean
  custom_css?: string
  ap_enabled?: boolean
}

// Helper to get image URL
export function getImageUrl(record: Event, thumb?: string): string | undefined {
  if (!record.image) return undefined
  return pb.files.getUrl(record, record.image, { thumb })
}
