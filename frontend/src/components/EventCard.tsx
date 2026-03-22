import { format } from 'date-fns'
import { Event, eventPath, getImageUrl } from '../lib/pocketbase'
import { tagStyle } from '../lib/color'
import './EventCard.css'

interface Props {
  event: Event
  variant?: 'featured' | 'compact'
}

export function EventCard({ event, variant = 'featured' }: Props) {
  const startDate = new Date(event.start_datetime)
  const imageUrl = getImageUrl(event, '400x300')

  if (variant === 'compact') {
    return (
      <a href={eventPath(event)} class="event-card event-card--compact">
        <div class="event-card-compact-inner">
          <time class="event-card-date">
            {format(startDate, 'EEE, MMM d · h:mm a')}
          </time>
          <h3 class="event-card-title">{event.title}</h3>
          {event.expand?.place && (
            <div class="event-card-place">
              📍 {event.expand.place.name}
            </div>
          )}
        </div>
        {imageUrl && (
          <div class="event-card-compact-thumb">
            <img src={imageUrl} alt="" width="80" height="60" loading="lazy" />
          </div>
        )}
      </a>
    )
  }

  // Featured variant (default)
  const cardClass = `event-card event-card--featured${!imageUrl ? ' event-card--no-image' : ''}`

  return (
    <a href={eventPath(event)} class={cardClass}>
      {imageUrl ? (
        <div class="event-card-thumb">
          <img src={imageUrl} alt="" width="400" height="300" loading="lazy" />
        </div>
      ) : (
        <div class="event-card-thumb">
          <div class="event-card-thumb-fallback" />
        </div>
      )}
      <div class="event-card-body">
        <time class="event-card-date">
          {format(startDate, 'EEE, MMM d · h:mm a')}
        </time>
        <h3 class="event-card-title">{event.title}</h3>
        {event.expand?.place && (
          <div class="event-card-place">
            📍 {event.expand.place.name}
          </div>
        )}
        {event.expand?.tags && event.expand.tags.length > 0 && (
          <div class="event-card-tags">
            {event.expand.tags.map(tag => (
              <span
                key={tag.id}
                class="tag"
                style={tagStyle(tag.color)}
              >
                {tag.name}
              </span>
            ))}
          </div>
        )}
      </div>
    </a>
  )
}
