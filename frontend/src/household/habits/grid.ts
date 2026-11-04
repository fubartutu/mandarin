// The history grid under an opened habit: for day habits, one column per
// week (Monday at the top), like a calendar turned on its side; for weekly
// habits, one square per week. The server sends one period per day (week);
// this only places them.

import {addDays, fromKey} from '../menu/week'
import type {HabitKind, PeriodResponse} from './queries'

/** Weeks shown for day habits, and weeks shown for weekly ones. */
export const GRID_WEEKS = 12
export const GRID_WEEKLY = 26

/** The Monday of a day's week. */
export function mondayOf(day: string): string {
	// getDay() is 0 for Sunday; (n + 6) % 7 makes Monday 0.
	return addDays(day, -((fromKey(day).getDay() + 6) % 7))
}

/** The days to ask the server for: whole weeks, ending today. */
export function gridRange(kind: HabitKind, today: string): {from: string, to: string} {
	const weeks = kind === 'weekly' ? GRID_WEEKLY : GRID_WEEKS
	return {from: addDays(mondayOf(today), -7 * (weeks - 1)), to: today}
}

/**
 * Day periods as week columns of seven cells, Monday first. Days the server
 * left out (after today) are null, so the last column ends on today.
 */
export function gridColumns(periods: PeriodResponse[], from: string, weeks = GRID_WEEKS): (PeriodResponse | null)[][] {
	const byDay = new Map(periods.map(p => [p.start, p]))
	return Array.from({length: weeks}, (_, w) =>
		Array.from({length: 7}, (_, d) => byDay.get(addDays(from, w * 7 + d)) ?? null))
}

/** The streak's unit for the labels: weekly habits count weeks, the others days. */
export function streakUnit(kind: HabitKind): 'days' | 'weeks' {
	return kind === 'weekly' ? 'weeks' : 'days'
}
