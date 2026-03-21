import PocketBase, { type BaseModel } from 'pocketbase'

export const pb = new PocketBase('/')

// Types for our collections
export interface Event {
  id: string
  slug?: string
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
  edit_token?: string
  expand?: {
    place?: Place
    tags?: Tag[]
    author?: User
  }
}

// Returns the URL path for an event, preferring slug over ID
export function eventPath(event: Pick<Event, 'id' | 'slug'>): string {
  return `/event/${event.slug || event.id}`
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
  status: 'pending' | 'approved'
}

export interface Tag {
  id: string
  name: string
  color?: string
  status: 'pending' | 'approved'
}

export interface User {
  id: string
  email: string
  role?: 'user' | 'editor' | 'admin'
  display_name?: string
}

export interface Settings extends BaseModel {
  instance_name: string
  instance_description: string
  allow_anonymous: boolean
  require_moderation: boolean
  custom_css: string
  custom_head: string
  ap_enabled: boolean
  ap_private_key: string
  ap_public_key: string
  logo?: string
}

export interface PageRecord extends BaseModel {
  title: string
  slug: string
  content: string
  show_in_nav: boolean
  show_in_footer: boolean
}

// Helper to get image URL
export function getImageUrl(record: Event, thumb?: string): string | undefined {
  if (!record.image) return undefined
  return pb.files.getUrl(record, record.image, { thumb })
}

// Auth helpers
export function getCurrentUser(): User | null {
  return pb.authStore.model as User | null
}

export function isAdmin(): boolean {
  const user = getCurrentUser()
  return user?.role === 'admin'
}

export function isEditor(): boolean {
  const user = getCurrentUser()
  return user?.role === 'editor' || user?.role === 'admin'
}

export function canModerate(): boolean {
  return isEditor()
}
