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
  recurrence_exceptions?: string
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
  subtitle: string
  instance_description: string
  allow_anonymous: boolean
  require_moderation: boolean
  custom_css: string
  custom_head: string
  ap_enabled: boolean
  ap_private_key: string
  ap_public_key: string
  favicon?: string
  umami_src: string
  umami_website_id: string
  umami_host_url: string
}

export interface PageRecord extends BaseModel {
  title: string
  slug: string
  content: string
  show_in_nav: boolean
  show_in_footer: boolean
  sort_order: number
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

export function recurrenceLabel(rule: string): string {
  const parts: Record<string, string> = {}
  rule.split(';').forEach(p => {
    const [k, v] = p.split('=')
    if (k && v) parts[k.toUpperCase()] = v.toUpperCase()
  })

  const dayNames: Record<string, string> = {
    MO: 'Monday', TU: 'Tuesday', WE: 'Wednesday', TH: 'Thursday',
    FR: 'Friday', SA: 'Saturday', SU: 'Sunday',
  }
  const ordinals: Record<string, string> = {
    '1': '1st', '2': '2nd', '3': '3rd', '4': '4th', '5': '5th', '-1': 'last',
  }

  switch (parts.FREQ) {
    case 'DAILY': return 'Repeats daily'
    case 'WEEKLY':
      return parts.INTERVAL === '2' ? 'Repeats fortnightly' : 'Repeats weekly'
    case 'MONTHLY':
      if (parts.BYDAY && parts.BYSETPOS) {
        return `Repeats monthly on the ${ordinals[parts.BYSETPOS] || parts.BYSETPOS + 'th'} ${dayNames[parts.BYDAY] || parts.BYDAY}`
      }
      return 'Repeats monthly'
    case 'YEARLY': return 'Repeats yearly'
    default: return 'Repeats'
  }
}

export function buildRecurrenceRule(opts: {
  freq: string
  interval?: number
  byDay?: string
  bySetPos?: number
  until?: string
  count?: number
}): string {
  const parts = [`FREQ=${opts.freq.toUpperCase()}`]
  if (opts.interval && opts.interval > 1) parts.push(`INTERVAL=${opts.interval}`)
  if (opts.byDay) parts.push(`BYDAY=${opts.byDay.toUpperCase()}`)
  if (opts.bySetPos) parts.push(`BYSETPOS=${opts.bySetPos}`)
  if (opts.until) {
    const d = new Date(opts.until)
    parts.push(`UNTIL=${d.toISOString().replace(/[-:]/g, '').split('.')[0]}Z`)
  }
  if (opts.count && opts.count > 0) parts.push(`COUNT=${opts.count}`)
  return parts.join(';')
}

export function parseVirtualId(id: string): { baseId: string; date: string } | null {
  const idx = id.indexOf('__')
  if (idx === -1) return null
  return { baseId: id.substring(0, idx), date: id.substring(idx + 2) }
}
