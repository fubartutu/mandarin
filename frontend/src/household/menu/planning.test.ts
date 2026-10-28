import {describe, expect, it} from 'vitest'

import {cookRoute, entryFactor, menuWeekRoute, plannedDayLabel} from './planning'

describe('menuWeekRoute', () => {
	const now = new Date(2026, 9, 28, 12, 0) // Wednesday 28 Oct 2026

	it('is the plain menu page for the next 7 days', () => {
		expect(menuWeekRoute('2026-10-28', now)).toEqual({name: 'menu'})
		expect(menuWeekRoute('2026-11-03', now)).toEqual({name: 'menu'})
	})

	it('names the first day of the page that day is on', () => {
		expect(menuWeekRoute('2026-11-04', now)).toEqual({name: 'menu', params: {start: '2026-11-04'}})
		expect(menuWeekRoute('2026-11-06', now)).toEqual({name: 'menu', params: {start: '2026-11-04'}})
		expect(menuWeekRoute('2026-10-26', now)).toEqual({name: 'menu', params: {start: '2026-10-21'}})
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

describe('entryFactor', () => {
	it('scales the recipe to the servings planned', () => {
		expect(entryFactor(6, 4)).toBe(1.5)
		expect(entryFactor(2, 4)).toBe(0.5)
		expect(entryFactor(4, 4)).toBe(1)
	})

	it('keeps the recipe as written when either serving count is unknown', () => {
		expect(entryFactor(0, 4)).toBe(1)
		expect(entryFactor(6, 0)).toBe(1)
		expect(entryFactor(0, 0)).toBe(1)
	})
})

describe('cookRoute', () => {
	it('passes the entry servings', () => {
		expect(cookRoute({
			recipeId: 7,
			servings: 6,
		})).toEqual({
			name: 'recipe.cook',
			params: {recipeId: 7},
			query: {servings: '6'},
		})
	})
	it('leaves servings out when the entry has none', () => {
		expect(cookRoute({recipeId: 7, servings: 0})).toEqual({name: 'recipe.cook', params: {recipeId: 7}})
	})
})
