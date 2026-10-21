import { useRef, useEffect } from 'preact/hooks'
import flatpickr from 'flatpickr'
import 'flatpickr/dist/flatpickr.min.css'
import './DateTimePicker.css'

interface Props {
  value: string
  onChange: (value: string) => void
  id?: string
  required?: boolean
  placeholder?: string
}

export function DateTimePicker({ value, onChange, id, required, placeholder }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const fpRef = useRef<flatpickr.Instance | null>(null)

  useEffect(() => {
    if (!inputRef.current) return

    fpRef.current = flatpickr(inputRef.current, {
      enableTime: true,
      dateFormat: 'Y-m-d H:i',
      time_24hr: true,
      defaultDate: value || undefined,
      onChange: (_dates, dateStr) => {
        onChange(dateStr)
      },
    })

    return () => {
      fpRef.current?.destroy()
    }
  }, [])

  // Sync external value changes
  useEffect(() => {
    if (fpRef.current && value) {
      fpRef.current.setDate(value, false)
    } else if (fpRef.current && !value) {
      fpRef.current.clear()
    }
  }, [value])

  return (
    <input
      ref={inputRef}
      type="text"
      id={id}
      required={required}
      placeholder={placeholder || 'Select date and time...'}
      class="flatpickr-input"
    />
  )
}
