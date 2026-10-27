import {describe, expect, it} from 'vitest'

import {atTaskTime, quickDays} from './reschedule'

const days = (now: Date) => Object.fromEntries(quickDays(now).map(({key, day}) => [key, day]))

describe('quickDays', () => {
	it('offers today, tomorrow, the coming weekend and next Monday midweek', () => {
		// Tuesday 27 October 2026, evening.
		expect(days(new Date(2026, 9, 27, 20, 30))).toEqual({
			today: new Date(2026, 9, 27),
			tomorrow: new Date(2026, 9, 28),
			weekend: new Date(2026, 9, 31),
			nextWeek: new Date(2026, 10, 2),
		})
	})

	it('drops "weekend" from Friday, when Saturday is tomorrow or today', () => {
		expect(Object.keys(days(new Date(2026, 9, 30)))).toEqual(['today', 'tomorrow', 'nextWeek'])
		expect(Object.keys(days(new Date(2026, 9, 31)))).toEqual(['today', 'tomorrow', 'nextWeek'])
	})

	it('makes "next week" a week away on a Monday, and tomorrow on a Sunday', () => {
		expect(days(new Date(2026, 10, 2)).nextWeek).toEqual(new Date(2026, 10, 9))
		expect(days(new Date(2026, 10, 1)).nextWeek).toEqual(new Date(2026, 10, 2))
	})
})

describe('atTaskTime', () => {
	it('keeps the task\'s time of day', () => {
		const overdue = new Date(2026, 9, 20, 19, 15)
		expect(atTaskTime(new Date(2026, 9, 28), overdue)).toEqual(new Date(2026, 9, 28, 19, 15))
	})
})
