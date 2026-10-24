import {describe, expect, it} from 'vitest'

import {menuWeekRoute, plannedDayLabel} from './planning'

describe('menuWeekRoute', () => {
	const now = new Date(2026, 9, 24, 12, 0) // Saturday 24 Oct 2026, week of Mon 19 Oct

	it('is the plain menu page for a day in this week', () => {
		expect(menuWeekRoute('2026-10-25', now)).toEqual({name: 'menu'})
		expect(menuWeekRoute('2026-10-19', now)).toEqual({name: 'menu'})
	})

	it('names the Monday of another week', () => {
		expect(menuWeekRoute('2026-10-29', now)).toEqual({name: 'menu', params: {week: '2026-10-26'}})
		expect(menuWeekRoute('2026-10-18', now)).toEqual({name: 'menu', params: {week: '2026-10-12'}})
	})
})

describe('plannedDayLabel', () => {
	it('writes the weekday, day and short month', () => {
		// The comma after the weekday depends on the ICU data version.
		expect(plannedDayLabel('2026-10-27', 'en-GB')).toMatch(/^Tue,? 27 Oct$/)
		expect(plannedDayLabel('2027-01-03', 'en-GB')).toMatch(/^Sun,? 3 Jan$/)
		expect(plannedDayLabel('2026-10-27', 'en-US')).toMatch(/^Tue,? Oct 27$/)
	})
})
