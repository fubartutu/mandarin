import {describe, expect, it} from 'vitest'

import {homeWeek} from './homeWeek'

// Week of Monday 19 to Sunday 25 October 2026.
const entry = (id: number, day: string, meal = 'dinner', position = id) => ({id, day, meal, position})

describe('homeWeek', () => {
	it('splits tonight from the rest of the week, keeping only days with dinners', () => {
		const entries = [
			entry(1, '2026-10-19'), // Monday, already past
			entry(2, '2026-10-21', 'dinner', 2),
			entry(3, '2026-10-21', 'dinner', 1),
			entry(4, '2026-10-21', 'lunch'),
			entry(5, '2026-10-24'),
			entry(6, '2026-10-22', 'breakfast'), // Thursday, no dinner
		]
		const week = homeWeek(entries, '2026-10-21')
		expect(week.tonight.map(e => e.id)).toEqual([3, 2])
		expect(week.rest.map(d => [d.day, d.entries.map(e => e.id)])).toEqual([['2026-10-24', [5]]])
		expect(week.empty).toBe(false)
	})

	it('has nothing after tonight on Sunday', () => {
		const week = homeWeek([entry(1, '2026-10-24'), entry(2, '2026-10-25')], '2026-10-25')
		expect(week.tonight.map(e => e.id)).toEqual([2])
		expect(week.rest).toEqual([])
		expect(week.empty).toBe(false)
	})

	it('is not empty when only later days are planned', () => {
		const week = homeWeek([entry(1, '2026-10-23')], '2026-10-20')
		expect(week.tonight).toEqual([])
		expect(week.rest.map(d => d.day)).toEqual(['2026-10-23'])
		expect(week.empty).toBe(false)
	})

	it('is empty when nothing is planned from tonight to Sunday', () => {
		const week = homeWeek([entry(1, '2026-10-19'), entry(2, '2026-10-22', 'lunch')], '2026-10-21')
		expect(week).toEqual({tonight: [], rest: [], empty: true})
		expect(homeWeek([], '2026-10-21').empty).toBe(true)
	})
})
