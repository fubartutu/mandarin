import {describe, expect, it} from 'vitest'

import {addDays, dayKey, daysBetween, dayStatus, dinnersByDay, pageStart, weekDays} from './week'

describe('dayKey', () => {
	it('is the local calendar day, not the UTC one', () => {
		// 11:30 pm local time is still that day, whatever the timezone.
		expect(dayKey(new Date(2026, 9, 24, 23, 30))).toBe('2026-10-24')
		expect(dayKey(new Date(2026, 0, 5, 0, 5))).toBe('2026-01-05')
	})
})

describe('weekDays', () => {
	it('is seven days from any day', () => {
		expect(weekDays('2026-10-19')).toEqual([
			'2026-10-19', '2026-10-20', '2026-10-21', '2026-10-22', '2026-10-23', '2026-10-24', '2026-10-25',
		])
		// A Wednesday start runs to the next Tuesday.
		expect(weekDays('2026-10-28').at(-1)).toBe('2026-11-03')
	})

	it('stays on whole days over a daylight-saving change', () => {
		// Europe and Australia both change their clocks around these weeks.
		expect(weekDays('2026-03-23').at(-1)).toBe('2026-03-29')
		expect(weekDays('2026-09-28').at(-1)).toBe('2026-10-04')
		expect(weekDays('2026-10-26').at(-1)).toBe('2026-11-01')
	})
})

describe('addDays', () => {
	it('moves back or forward across months and years', () => {
		expect(addDays('2026-10-28', -7)).toBe('2026-10-21')
		expect(addDays('2026-10-28', 7)).toBe('2026-11-04')
		expect(addDays('2026-12-28', 7)).toBe('2027-01-04')
	})
})

describe('daysBetween', () => {
	it('counts whole days, even over a daylight-saving change', () => {
		expect(daysBetween('2026-10-28', '2026-10-28')).toBe(0)
		expect(daysBetween('2026-10-28', '2026-11-04')).toBe(7)
		expect(daysBetween('2026-11-04', '2026-10-28')).toBe(-7)
		expect(daysBetween('2026-09-30', '2026-10-07')).toBe(7) // Sydney's clocks go forward on 4 Oct
		expect(daysBetween('2026-12-31', '2027-01-01')).toBe(1)
	})
})

describe('pageStart', () => {
	// Pages are 7 days each, starting from today (Wednesday 28 Oct 2026).
	const today = '2026-10-28'

	it('is today for the next 7 days', () => {
		expect(pageStart('2026-10-28', today)).toBe(today)
		expect(pageStart('2026-11-03', today)).toBe(today)
	})

	it('is a whole number of weeks from today otherwise', () => {
		expect(pageStart('2026-11-04', today)).toBe('2026-11-04')
		expect(pageStart('2026-11-12', today)).toBe('2026-11-11')
		expect(pageStart('2026-10-27', today)).toBe('2026-10-21')
		expect(pageStart('2026-10-21', today)).toBe('2026-10-21')
		expect(pageStart('2026-10-20', today)).toBe('2026-10-14')
	})
})

describe('dayStatus', () => {
	it('says whether a day is past, today or still to come', () => {
		expect(dayStatus('2026-10-23', '2026-10-24')).toBe('past')
		expect(dayStatus('2026-10-24', '2026-10-24')).toBe('today')
		expect(dayStatus('2026-10-25', '2026-10-24')).toBe('future')
		expect(dayStatus('2025-12-31', '2026-01-01')).toBe('past')
	})
})

describe('dinnersByDay', () => {
	const entry = (id: number, day: string, meal: string, position: number) => ({id, day, meal, position, note: `#${id}`})

	it('gives every day of the week its dinners in order, and leaves other meals out', () => {
		const days = weekDays('2026-10-19')
		const byDay = dinnersByDay([
			entry(1, '2026-10-20', 'dinner', 1),
			entry(2, '2026-10-20', 'breakfast', 0),
			entry(3, '2026-10-20', 'dinner', 0),
			entry(4, '2026-10-25', 'dinner', 0),
		], days)

		expect([...byDay.keys()]).toEqual(days)
		expect(byDay.get('2026-10-20')?.map(e => e.id)).toEqual([3, 1])
		expect(byDay.get('2026-10-25')?.map(e => e.id)).toEqual([4])
		expect(byDay.get('2026-10-19')).toEqual([])
	})
})
