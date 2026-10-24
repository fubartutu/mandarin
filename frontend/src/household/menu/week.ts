// Week arithmetic for the menu. Days are calendar days written YYYY-MM-DD
// (what the API stores), always worked out in the device's local time: the
// menu for "Tuesday" shouldn't depend on the UTC offset.
//
// Dates are moved with setDate() rather than by adding 24 hours, because a
// day with a daylight-saving change is 23 or 25 hours long.

export type DayStatus = 'past' | 'today' | 'future'

/** The local calendar day of a date, as YYYY-MM-DD. */
export function dayKey(date: Date): string {
	const pad = (n: number) => String(n).padStart(2, '0')
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

/** Local midnight of a YYYY-MM-DD day. */
export function fromKey(day: string): Date {
	const [year, month, date] = day.split('-').map(Number)
	return new Date(year, month - 1, date)
}

function addDays(day: string, days: number): string {
	const date = fromKey(day)
	date.setDate(date.getDate() + days)
	return dayKey(date)
}

/** The Monday of the week a date falls in (weeks run Monday to Sunday). */
export function mondayOf(date: Date): string {
	// getDay() is 0 for Sunday, 1 for Monday … 6 for Saturday.
	const sinceMonday = (date.getDay() + 6) % 7
	return addDays(dayKey(date), -sinceMonday)
}

/** The seven days of the week starting on that Monday. */
export function weekDays(monday: string): string[] {
	return Array.from({length: 7}, (_, i) => addDays(monday, i))
}

/** The Monday `weeks` weeks before (negative) or after. */
export function shiftWeek(monday: string, weeks: number): string {
	return addDays(monday, weeks * 7)
}

/** YYYY-MM-DD strings sort like the dates they are, so they compare directly. */
export function dayStatus(day: string, today: string): DayStatus {
	if (day < today) return 'past'
	if (day > today) return 'future'
	return 'today'
}

interface DayEntry {
	day: string
	meal: string
	position: number
}

/**
 * Each of `days` with its dinner entries, in position order. Only dinner is
 * shown for now; the other meals stay in the data.
 */
export function dinnersByDay<T extends DayEntry>(entries: T[], days: string[]): Map<string, T[]> {
	const byDay = new Map<string, T[]>(days.map(day => [day, []]))
	for (const entry of entries) {
		if (entry.meal === 'dinner') byDay.get(entry.day)?.push(entry)
	}
	for (const list of byDay.values()) list.sort((a, b) => a.position - b.position)
	return byDay
}
