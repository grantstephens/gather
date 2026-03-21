import { useState } from 'preact/hooks'
import './MiniCalendar.css'

interface Props {
  eventDates?: Map<string, string[]>  // date → unique tag colors ('' = default)
  selectedDate?: string               // Currently selected date in YYYY-MM-DD format
  onDateSelect?: (date: string) => void
}

const DAYS = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']
const MONTHS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December'
]

function formatDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function MiniCalendar({ eventDates, selectedDate, onDateSelect }: Props) {
  const today = new Date()
  const [viewDate, setViewDate] = useState(new Date(today.getFullYear(), today.getMonth(), 1))

  const year = viewDate.getFullYear()
  const month = viewDate.getMonth()

  // Get first day of month and number of days
  const firstDayOfMonth = new Date(year, month, 1)
  const lastDayOfMonth = new Date(year, month + 1, 0)
  const daysInMonth = lastDayOfMonth.getDate()
  const startingDayOfWeek = firstDayOfMonth.getDay()

  // Generate calendar days
  const days: (number | null)[] = []

  // Add empty slots for days before the first of the month
  for (let i = 0; i < startingDayOfWeek; i++) {
    days.push(null)
  }

  // Add days of the month
  for (let day = 1; day <= daysInMonth; day++) {
    days.push(day)
  }

  const prevMonth = () => {
    setViewDate(new Date(year, month - 1, 1))
  }

  const nextMonth = () => {
    setViewDate(new Date(year, month + 1, 1))
  }

  const goToToday = () => {
    setViewDate(new Date(today.getFullYear(), today.getMonth(), 1))
    if (onDateSelect) {
      onDateSelect(formatDate(today))
    }
  }

  const handleDateClick = (day: number) => {
    const date = new Date(year, month, day)
    if (onDateSelect) {
      onDateSelect(formatDate(date))
    }
  }

  const todayStr = formatDate(today)

  return (
    <div class="mini-calendar">
      <div class="calendar-header">
        <button class="calendar-nav" onClick={prevMonth} aria-label="Previous month">
          &lt;
        </button>
        <button class="calendar-title" onClick={goToToday}>
          {MONTHS[month]} {year}
        </button>
        <button class="calendar-nav" onClick={nextMonth} aria-label="Next month">
          &gt;
        </button>
      </div>

      <div class="calendar-grid">
        {DAYS.map(day => (
          <div key={day} class="calendar-day-name">{day}</div>
        ))}

        {days.map((day, index) => {
          if (day === null) {
            return <div key={`empty-${index}`} class="calendar-day empty" />
          }

          const dateStr = formatDate(new Date(year, month, day))
          const isToday = dateStr === todayStr
          const isSelected = dateStr === selectedDate
          const dotColors = eventDates?.get(dateStr)
          const hasEvents = dotColors !== undefined
          const isPast = new Date(year, month, day) < new Date(today.getFullYear(), today.getMonth(), today.getDate())

          return (
            <button
              key={day}
              class={`calendar-day ${isToday ? 'today' : ''} ${isSelected ? 'selected' : ''} ${hasEvents ? 'has-events' : ''} ${isPast ? 'past' : ''}`}
              onClick={() => handleDateClick(day)}
            >
              {day}
              {hasEvents && (
                <div class="event-dots">
                  {(dotColors!.length > 0 ? dotColors!.slice(0, 3) : ['']).map((color, i) => (
                    <span
                      key={i}
                      class="event-dot"
                      style={color && !isSelected && !isPast ? { backgroundColor: color } : undefined}
                    />
                  ))}
                </div>
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}
