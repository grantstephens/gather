import { format } from 'date-fns'
import { Event, eventPath, getImageUrl } from '../lib/pocketbase'
import './EventCard.css'

interface Props {
  event: Event
}

export function EventCard({ event }: Props) {
  const startDate = new Date(event.start_datetime)
  const imageUrl = getImageUrl(event, '400x300')

  return (
    <a href={eventPath(event)} class="event-card">
      {imageUrl && (
        <div class="event-card-image">
          <img src={imageUrl} alt="" />
        </div>
      )}
      <div class="event-card-content">
        <time class="event-card-date">
          {format(startDate, 'EEE, MMM d · h:mm a')}
        </time>
        <h3 class="event-card-title">{event.title}</h3>
        {event.expand?.place && (
          <div class="event-card-place">
            {event.expand.place.name}
          </div>
        )}
        {event.expand?.tags && event.expand.tags.length > 0 && (
          <div class="event-card-tags">
            {event.expand.tags.map(tag => (
              <span
                key={tag.id}
                class="tag"
                style={tag.color ? { backgroundColor: tag.color } : undefined}
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
