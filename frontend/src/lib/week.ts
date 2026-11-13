export type WeekStart = 'sunday' | 'monday'

const SUNDAY_FIRST = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']
const MONDAY_FIRST = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su']

/** Column headings for the month grid, starting on the configured day. */
export function weekdayLabels(weekStartsOn: WeekStart): string[] {
  return weekStartsOn === 'monday' ? MONDAY_FIRST : SUNDAY_FIRST
}

/** Empty cells before the 1st of a month, given that day's Date#getDay() (0 = Sunday). */
export function leadingBlanks(firstDayOfMonthDow: number, weekStartsOn: WeekStart): number {
  const offset = weekStartsOn === 'monday' ? 1 : 0
  return (firstDayOfMonthDow - offset + 7) % 7
}
