import { describe, it, expect } from 'vitest'
import { leadingBlanks, weekdayLabels } from './week'

describe('weekdayLabels', () => {
  it('starts on Sunday by default layout', () => {
    expect(weekdayLabels('sunday')).toEqual(['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'])
  })
  it('starts on Monday and ends on Sunday', () => {
    expect(weekdayLabels('monday')).toEqual(['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'])
  })
})

describe('leadingBlanks', () => {
  it('Sunday-first: a month starting on Sunday has no blanks, Saturday has six', () => {
    expect(leadingBlanks(0, 'sunday')).toBe(0)
    expect(leadingBlanks(6, 'sunday')).toBe(6)
  })
  it('Monday-first: a month starting on Monday has no blanks, Sunday has six', () => {
    expect(leadingBlanks(1, 'monday')).toBe(0)
    expect(leadingBlanks(0, 'monday')).toBe(6)
    expect(leadingBlanks(6, 'monday')).toBe(5)
  })
})
