// Mandarin: the quick reschedule menu's dates (RescheduleMenu.vue): absolute
// days like Todoist's (Today, Tomorrow, This weekend, Next week), not "+1 day"
// on the old date, so an overdue task really leaves the overdue list.

import {getDateWithTime} from '@/helpers/time/getDateWithTime'

export type QuickDay = 'today' | 'tomorrow' | 'weekend' | 'nextWeek'

const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate())
const addDays = (d: Date, n: number) => new Date(d.getFullYear(), d.getMonth(), d.getDate() + n)

/**
 * The quick days on offer, in order. "This weekend" is the coming Saturday
 * and is left out on a weekend, where Today and Tomorrow already cover it.
 * "Next week" is next Monday.
 */
export function quickDays(now: Date): {key: QuickDay, day: Date}[] {
	const today = startOfDay(now)
	const weekday = today.getDay() // Sunday is 0
	const days: {key: QuickDay, day: Date}[] = [
		{key: 'today', day: today},
		{key: 'tomorrow', day: addDays(today, 1)},
	]
	if (weekday >= 1 && weekday <= 4) { // Monday to Thursday; on Friday, Saturday is tomorrow
		days.push({key: 'weekend', day: addDays(today, 6 - weekday)})
	}
	days.push({key: 'nextWeek', day: addDays(today, ((8 - weekday) % 7) || 7)})
	return days
}

/** The day at the task's current time of day, or the user's default time for a task without one. */
export function atTaskTime(day: Date, due: Date | null): Date {
	if (!due) return getDateWithTime(day)
	return new Date(day.getFullYear(), day.getMonth(), day.getDate(), due.getHours(), due.getMinutes(), due.getSeconds())
}
