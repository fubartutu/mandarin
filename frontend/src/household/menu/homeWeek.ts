// What the Home page's menu card shows: tonight's dinner, then the days
// after it up to Sunday that have a dinner planned. Days already past
// aren't shown, so they don't count towards "planned" either.

import {dinnersByDay, fromKey, mondayOf, weekDays} from './week'

interface DayEntry {
	day: string
	meal: string
	position: number
}

export interface HomeWeek<T> {
	/** Tonight's dinner entries, in position order. */
	tonight: T[]
	/** Tomorrow … Sunday, only the days with dinner entries. */
	rest: {day: string, entries: T[]}[]
	/** Nothing planned from tonight to Sunday. */
	empty: boolean
}

/** `entries` from this week's menu, `today` as YYYY-MM-DD. */
export function homeWeek<T extends DayEntry>(entries: T[], today: string): HomeWeek<T> {
	const days = weekDays(mondayOf(fromKey(today))).filter(day => day >= today)
	const dinners = dinnersByDay(entries, days)
	const tonight = dinners.get(today) ?? []
	const rest = days
		.slice(1)
		.map(day => ({day, entries: dinners.get(day) ?? []}))
		.filter(d => d.entries.length > 0)
	return {tonight, rest, empty: tonight.length === 0 && rest.length === 0}
}
