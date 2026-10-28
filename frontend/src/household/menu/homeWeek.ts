// What the Home page's menu card shows: tonight's dinner, then the next
// six days that have a dinner planned (the same 7 days as the Menu page).

import {dinnersByDay, weekDays} from './week'

interface DayEntry {
	day: string
	meal: string
	position: number
}

export interface HomeWeek<T> {
	/** Tonight's dinner entries, in position order. */
	tonight: T[]
	/** Tomorrow and the five days after it, only those with dinner entries. */
	rest: {day: string, entries: T[]}[]
	/** Nothing planned in the next 7 days. */
	empty: boolean
}

/** `entries` from the next 7 days' menu, `today` as YYYY-MM-DD. */
export function homeWeek<T extends DayEntry>(entries: T[], today: string): HomeWeek<T> {
	const days = weekDays(today)
	const dinners = dinnersByDay(entries, days)
	const tonight = dinners.get(today) ?? []
	const rest = days
		.slice(1)
		.map(day => ({day, entries: dinners.get(day) ?? []}))
		.filter(d => d.entries.length > 0)
	return {tonight, rest, empty: tonight.length === 0 && rest.length === 0}
}
