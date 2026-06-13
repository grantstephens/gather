import { useState, useEffect } from 'preact/hooks'
import { buildRecurrenceRule } from '../lib/pocketbase'
import './RecurrencePicker.css'

interface Props {
  value: string
  onChange: (rule: string) => void
  startDate?: string
}

type Freq = '' | 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'YEARLY'
type MonthlyMode = 'date' | 'weekday'
type EndMode = 'never' | 'until' | 'count'

function getWeekdayInfo(dateStr: string): { day: string; pos: number; label: string } | null {
  if (!dateStr) return null
  const d = new Date(dateStr.replace(' ', 'T'))
  if (isNaN(d.getTime())) return null

  const days = ['SU', 'MO', 'TU', 'WE', 'TH', 'FR', 'SA']
  const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
  const day = days[d.getDay()]
  const dayName = dayNames[d.getDay()]
  const pos = Math.ceil(d.getDate() / 7)
  const ordinals = ['', '1st', '2nd', '3rd', '4th', '5th']

  return { day, pos, label: `${ordinals[pos]} ${dayName}` }
}

function parseExistingRule(rule: string): {
  freq: Freq; interval: number; monthlyMode: MonthlyMode;
  byDay: string; bySetPos: number; endMode: EndMode;
  until: string; count: number
} {
  const defaults = {
    freq: '' as Freq, interval: 1, monthlyMode: 'date' as MonthlyMode,
    byDay: '', bySetPos: 0, endMode: 'never' as EndMode, until: '', count: 4,
  }
  if (!rule) return defaults

  const parts: Record<string, string> = {}
  rule.split(';').forEach(p => {
    const [k, v] = p.split('=')
    if (k && v) parts[k.toUpperCase()] = v
  })

  const result = { ...defaults }
  result.freq = (parts.FREQ || '') as Freq
  if (parts.INTERVAL) result.interval = parseInt(parts.INTERVAL) || 1
  if (parts.BYDAY) { result.byDay = parts.BYDAY; result.monthlyMode = 'weekday' }
  if (parts.BYSETPOS) result.bySetPos = parseInt(parts.BYSETPOS) || 0
  if (parts.UNTIL) {
    result.endMode = 'until'
    const u = parts.UNTIL.replace(/Z$/, '')
    if (u.length >= 8) result.until = `${u.slice(0, 4)}-${u.slice(4, 6)}-${u.slice(6, 8)}`
  }
  if (parts.COUNT) { result.endMode = 'count'; result.count = parseInt(parts.COUNT) || 4 }

  return result
}

export function RecurrencePicker({ value, onChange, startDate }: Props) {
  const parsed = parseExistingRule(value)
  const [freq, setFreq] = useState<Freq>(parsed.freq)
  const [interval, setInterval] = useState(parsed.interval)
  const [monthlyMode, setMonthlyMode] = useState<MonthlyMode>(parsed.monthlyMode)
  const [endMode, setEndMode] = useState<EndMode>(parsed.endMode)
  const [until, setUntil] = useState(parsed.until)
  const [count, setCount] = useState(parsed.count)

  const weekdayInfo = startDate ? getWeekdayInfo(startDate) : null

  useEffect(() => {
    if (!freq) { if (value) onChange(''); return }

    const opts: Parameters<typeof buildRecurrenceRule>[0] = { freq }

    if (freq === 'WEEKLY' && interval === 2) opts.interval = 2

    if (freq === 'MONTHLY' && monthlyMode === 'weekday' && weekdayInfo) {
      opts.byDay = weekdayInfo.day
      opts.bySetPos = weekdayInfo.pos
    }

    if (endMode === 'until' && until) opts.until = until
    else if (endMode === 'count' && count > 0) opts.count = count

    const rule = buildRecurrenceRule(opts)
    if (rule !== value) onChange(rule)
  }, [freq, interval, monthlyMode, endMode, until, count, startDate])

  const handleFreqChange = (val: string) => {
    if (val === 'FORTNIGHTLY') { setFreq('WEEKLY'); setInterval(2) }
    else { setFreq(val as Freq); setInterval(1) }
  }

  const currentFreqValue = freq === 'WEEKLY' && interval === 2 ? 'FORTNIGHTLY' : freq

  return (
    <div class="recurrence-picker">
      <select
        value={currentFreqValue}
        onChange={(e) => handleFreqChange((e.target as HTMLSelectElement).value)}
        class="recurrence-freq"
      >
        <option value="">Does not repeat</option>
        <option value="DAILY">Daily</option>
        <option value="WEEKLY">Weekly</option>
        <option value="FORTNIGHTLY">Fortnightly</option>
        <option value="MONTHLY">Monthly</option>
        <option value="YEARLY">Yearly</option>
      </select>

      {freq === 'MONTHLY' && weekdayInfo && (
        <div class="recurrence-monthly-mode">
          <label>
            <input type="radio" name="monthlyMode" checked={monthlyMode === 'date'} onChange={() => setMonthlyMode('date')} />
            Same date each month
          </label>
          <label>
            <input type="radio" name="monthlyMode" checked={monthlyMode === 'weekday'} onChange={() => setMonthlyMode('weekday')} />
            {weekdayInfo.label} each month
          </label>
        </div>
      )}

      {freq && (
        <div class="recurrence-end">
          <span class="recurrence-end-label">Ends</span>
          <div class="recurrence-end-options">
            <label>
              <input type="radio" name="endMode" checked={endMode === 'never'} onChange={() => setEndMode('never')} />
              Never
            </label>
            <label>
              <input type="radio" name="endMode" checked={endMode === 'until'} onChange={() => setEndMode('until')} />
              Until
              {endMode === 'until' && (
                <input type="date" value={until} onInput={(e) => setUntil((e.target as HTMLInputElement).value)} class="recurrence-until-date" />
              )}
            </label>
            <label>
              <input type="radio" name="endMode" checked={endMode === 'count'} onChange={() => setEndMode('count')} />
              After
              {endMode === 'count' && (
                <>
                  <input type="number" min="1" max="365" value={count} onInput={(e) => setCount(parseInt((e.target as HTMLInputElement).value) || 1)} class="recurrence-count-input" />
                  occurrences
                </>
              )}
            </label>
          </div>
        </div>
      )}
    </div>
  )
}
