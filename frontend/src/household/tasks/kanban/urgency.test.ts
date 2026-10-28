import {describe, expect, it} from 'vitest'

import {urgencyOf} from './urgency'

// Tuesday 27 October 2026, 10:00 local time.
const now = new Date(2026, 9, 27, 10, 0)
const at = (day: number, hour = 17) => new Date(2026, 9, day, hour, 0).toISOString()

describe('urgencyOf', () => {
	it('follows the priority', () => {
		expect(urgencyOf({priority: 5}, now)).toBe('red') // Do now
		expect(urgencyOf({priority: 4}, now)).toBe('red') // Urgent
		expect(urgencyOf({priority: 3}, now)).toBe('orange')
		expect(urgencyOf({priority: 2}, now)).toBe('yellow')
		expect(urgencyOf({priority: 1}, now)).toBeNull()
		expect(urgencyOf({priority: 0}, now)).toBeNull()
	})

	it('follows the due date', () => {
		expect(urgencyOf({due_date: at(27, 9)}, now)).toBe('red') // an hour ago
		expect(urgencyOf({due_date: at(25)}, now)).toBe('red')
		expect(urgencyOf({due_date: at(27, 23)}, now)).toBe('orange') // later today
		expect(urgencyOf({due_date: at(28, 8)}, now)).toBe('yellow')
		expect(urgencyOf({due_date: at(30, 23)}, now)).toBe('yellow') // 3 days away
		expect(urgencyOf({due_date: at(31)}, now)).toBeNull()
		expect(urgencyOf({due_date: '0001-01-01T00:00:00Z'}, now)).toBeNull() // Vikunja's no date
		expect(urgencyOf({due_date: null}, now)).toBeNull()
	})

	it('takes the more urgent of the two', () => {
		expect(urgencyOf({priority: 1, due_date: at(25)}, now)).toBe('red')
		expect(urgencyOf({priority: 4, due_date: at(31)}, now)).toBe('red')
		expect(urgencyOf({priority: 2, due_date: at(27, 23)}, now)).toBe('orange')
	})

	it('is never urgent once done', () => {
		expect(urgencyOf({done: true, priority: 5, due_date: at(25)}, now)).toBeNull()
	})
})
