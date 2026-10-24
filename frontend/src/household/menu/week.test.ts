import {describe, expect, it} from 'vitest'

import {dayKey, dayStatus, dinnersByDay, mondayOf, shiftWeek, weekDays} from './week'

describe('dayKey', () => {
	it('is the local calendar day, not the UTC one', () => {
		// 11:30 pm local time is still that day, whatever the timezone.
		expect(dayKey(new Date(2026, 9, 24, 23, 30))).toBe('2026-10-24')
		expect(dayKey(new Date(2026, 0, 5, 0, 5))).toBe('2026-01-05')
	})
})

describe('mondayOf', () => {
	it.each([
		['Monday itself', new Date(2026, 9, 19), '2026-10-19'],
		['a Wednesday', new Date(2026, 9, 21, 18, 0), '2026-10-19'],
		['Sunday belongs to the week before it', new Date(2026, 9, 25, 20, 0), '2026-10-19'],
		['across a month end', new Date(2026, 10, 1), '2026-10-26'],
		['across a year end', new Date(2027, 0, 2), '2026-12-28'],
	])('%s', (_name, date, monday) => {
		expect(mondayOf(date)).toBe(monday)
	})
})

describe('weekDays', () => {
	it('runs Monday to Sunday', () => {
		expect(weekDays('2026-10-19')).toEqual([
			'2026-10-19', '2026-10-20', '2026-10-21', '2026-10-22', '2026-10-23', '2026-10-24', '2026-10-25',
		])
	})

	it('stays on whole days over a daylight-saving change', () => {
		// Europe and Australia both change their clocks around these weeks.
		expect(weekDays('2026-03-23').at(-1)).toBe('2026-03-29')
		expect(weekDays('2026-09-28').at(-1)).toBe('2026-10-04')
		expect(weekDays('2026-10-26').at(-1)).toBe('2026-11-01')
	})
})

describe('shiftWeek', () => {
	it('moves a week back or forward', () => {
		expect(shiftWeek('2026-10-19', -1)).toBe('2026-10-12')
		expect(shiftWeek('2026-10-19', 1)).toBe('2026-10-26')
		expect(shiftWeek('2026-12-28', 1)).toBe('2027-01-04')
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
