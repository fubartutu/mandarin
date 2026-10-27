// Mandarin: calendar repeats Vikunja's single interval can't express, kept as
// an iCalendar RRULE on the server (TaskRepeat.rule): "every Mon and Thu",
// "every weekday", "every 2nd Tuesday", "every last day of the month",
// "every 15th", "every 2 months", "yearly". The server decides the next date
// when the task is done; this file builds and reads rules, describes them,
// finds the first date, and reads them from quick add titles.

export type Weekday = 'MO' | 'TU' | 'WE' | 'TH' | 'FR' | 'SA' | 'SU'
export const WEEKDAYS: Weekday[] = ['MO', 'TU', 'WE', 'TH', 'FR', 'SA', 'SU']
const WORKDAYS: Weekday[] = ['MO', 'TU', 'WE', 'TH', 'FR']

/** Which occurrence in the month: 1st to 4th, or -1 for the last. */
export type Nth = 1 | 2 | 3 | 4 | -1

export type RepeatRule =
	| {kind: 'weekly', interval: number, days: Weekday[]}
	| {kind: 'monthDay', interval: number, day: number} // day -1 = the last day
	| {kind: 'monthWeekday', interval: number, nth: Nth, weekday: Weekday}
	| {kind: 'months', interval: number} // on the due date's day
	| {kind: 'years', interval: number} // on the due date's day and month

export function toRRule(rule: RepeatRule): string {
	const interval = rule.interval > 1 ? `;INTERVAL=${rule.interval}` : ''
	switch (rule.kind) {
		case 'weekly':
			return `FREQ=WEEKLY${interval};BYDAY=${sortDays(rule.days).join(',')}`
		case 'monthDay':
			return `FREQ=MONTHLY${interval};BYMONTHDAY=${rule.day}`
		case 'monthWeekday':
			return `FREQ=MONTHLY${interval};BYDAY=${rule.nth}${rule.weekday}`
		case 'months':
			return `FREQ=MONTHLY${interval}`
		case 'years':
			return `FREQ=YEARLY${interval}`
	}
}

/** Reads a rule the server stored; null for one this UI can't show. */
export function fromRRule(text: string): RepeatRule | null {
	if (!text) return null
	const parts = new Map(text.toUpperCase().split(';').map(part => part.split('=') as [string, string]))
	const interval = parts.has('INTERVAL') ? Number(parts.get('INTERVAL')) : 1
	if (!Number.isInteger(interval) || interval < 1) return null
	const byDay = parts.get('BYDAY')
	const byMonthDay = parts.get('BYMONTHDAY')
	if (parts.has('BYSETPOS')) return null
	switch (parts.get('FREQ')) {
		case 'WEEKLY': {
			const days = (byDay ?? '').split(',').filter((day): day is Weekday => WEEKDAYS.includes(day as Weekday))
			return days.length && !byMonthDay ? {kind: 'weekly', interval, days: sortDays(days)} : null
		}
		case 'MONTHLY': {
			if (byMonthDay !== undefined && byDay === undefined) {
				const day = Number(byMonthDay)
				return day === -1 || (day >= 1 && day <= 31) ? {kind: 'monthDay', interval, day} : null
			}
			if (byDay !== undefined && byMonthDay === undefined) {
				const match = /^([+-]?\d)(MO|TU|WE|TH|FR|SA|SU)$/.exec(byDay)
				const nth = match ? Number(match[1]) : 0
				return match && [1, 2, 3, 4, -1].includes(nth)
					? {kind: 'monthWeekday', interval, nth: nth as Nth, weekday: match[2] as Weekday}
					: null
			}
			return byDay === undefined && byMonthDay === undefined ? {kind: 'months', interval} : null
		}
		case 'YEARLY':
			return byDay === undefined && byMonthDay === undefined ? {kind: 'years', interval} : null
	}
	return null
}

function sortDays(days: Weekday[]): Weekday[] {
	return [...new Set(days)].sort((a, b) => WEEKDAYS.indexOf(a) - WEEKDAYS.indexOf(b))
}

export function isWorkdays(days: Weekday[]): boolean {
	return sortDays(days).join() === WORKDAYS.join()
}

const DAY = 24 * 3600

/**
 * The repeat_after a task with this rule shows to code that only knows
 * intervals (Vikunja's own UI, CalDAV, other apps); the server uses the same.
 */
export function standInSeconds(rule: RepeatRule): number {
	switch (rule.kind) {
		case 'weekly':
			return rule.interval * 7 * DAY
		case 'monthDay':
		case 'monthWeekday':
		case 'months':
			return rule.interval * 30 * DAY
		case 'years':
			return rule.interval * 365 * DAY
	}
}

const jsDay = (day: Weekday) => (WEEKDAYS.indexOf(day) + 1) % 7 // Date#getDay: Sunday is 0

/** The day of month of the nth weekday, or null when the month has no such day (a 5th). */
function nthWeekdayOf(year: number, month: number, nth: Nth, weekday: Weekday): number | null {
	if (nth === -1) {
		const last = new Date(year, month + 1, 0)
		return last.getDate() - ((last.getDay() - jsDay(weekday) + 7) % 7)
	}
	const first = new Date(year, month, 1).getDay()
	return 1 + ((jsDay(weekday) - first + 7) % 7) + (nth - 1) * 7
}

/**
 * The first date on or after `from` the rule falls on, at `from`'s time of
 * day. For month and year intervals that is `from` itself: they repeat on its day.
 */
export function firstOccurrence(rule: RepeatRule, from: Date): Date {
	const at = (year: number, month: number, day: number) =>
		new Date(year, month, day, from.getHours(), from.getMinutes(), from.getSeconds())
	switch (rule.kind) {
		case 'weekly':
			for (let i = 0; i < 7; i++) {
				const d = at(from.getFullYear(), from.getMonth(), from.getDate() + i)
				if (rule.days.some(day => jsDay(day) === d.getDay())) return d
			}
			return from
		case 'monthDay':
		case 'monthWeekday':
			// Look at this month and the next ones; a 31st may skip a few.
			for (let i = 0; i < 13; i++) {
				const year = from.getFullYear()
				const month = from.getMonth() + i
				const daysInMonth = new Date(year, month + 1, 0).getDate()
				let day: number | null
				if (rule.kind === 'monthDay') {
					day = rule.day === -1 ? daysInMonth : (rule.day <= daysInMonth ? rule.day : null)
				} else {
					day = nthWeekdayOf(year, month, rule.nth, rule.weekday)
				}
				if (day === null || day > daysInMonth) continue
				const d = at(year, month, day)
				if (d >= at(from.getFullYear(), from.getMonth(), from.getDate())) return d
			}
			return from
		case 'months':
		case 'years':
			return from
	}
}

// --- Quick add -----------------------------------------------------------

const DAY_WORDS: Record<string, Weekday> = {
	monday: 'MO', mon: 'MO',
	tuesday: 'TU', tues: 'TU', tue: 'TU',
	wednesday: 'WE', wed: 'WE',
	thursday: 'TH', thurs: 'TH', thur: 'TH', thu: 'TH',
	friday: 'FR', fri: 'FR',
	saturday: 'SA', sat: 'SA',
	sunday: 'SU', sun: 'SU',
}
const DAY_WORD = `(?:${Object.keys(DAY_WORDS).join('|')})s?`
const DAY_LIST = `${DAY_WORD}(?:(?:, ?| and | ?& ?|, and )${DAY_WORD})*`
const NUMBER_WORDS: Record<string, number> = {
	other: 2, two: 2, three: 3, four: 4, five: 5, six: 6, seven: 7, eight: 8, nine: 9, ten: 10, eleven: 11, twelve: 12,
}
const COUNT = `(?:[0-9]+|${Object.keys(NUMBER_WORDS).join('|')})`
const NTH_WORDS: Record<string, Nth> = {
	'1st': 1, first: 1, '2nd': 2, second: 2, '3rd': 3, third: 3, '4th': 4, fourth: 4, last: -1,
}
const NTH = `(?:${Object.keys(NTH_WORDS).join('|')})`
const EVERY = '(?:every|each)'
const OF_MONTH = '(?: of (?:the|each|every) month)?'

const toCount = (word: string) => NUMBER_WORDS[word.toLowerCase()] ?? parseInt(word, 10)
const dayOf = (word: string) => DAY_WORDS[word.toLowerCase().replace(/s$/, '')] ?? DAY_WORDS[word.toLowerCase()]

/** The Vikunja repeat word that stands in for a rule in the title (see quickAdd.ts). */
function vikunjaWord(rule: RepeatRule): string {
	return rule.kind === 'weekly' ? 'weekly' : rule.kind === 'years' ? 'yearly' : 'monthly'
}

type Pattern = [RegExp, (m: RegExpExecArray) => RepeatRule | null]

const patterns: Pattern[] = [
	[new RegExp(`${EVERY} (?:weekday|workday|work day)s?`, 'i'),
		() => ({kind: 'weekly', interval: 1, days: [...WORKDAYS]})],
	[new RegExp(`${EVERY} weekends?`, 'i'),
		() => ({kind: 'weekly', interval: 1, days: ['SA', 'SU']})],
	[new RegExp(`${EVERY} last day${OF_MONTH}`, 'i'),
		() => ({kind: 'monthDay', interval: 1, day: -1})],
	[new RegExp(`${EVERY} (${NTH}) (${DAY_WORD})${OF_MONTH}`, 'i'),
		m => ({kind: 'monthWeekday', interval: 1, nth: NTH_WORDS[m[1].toLowerCase()], weekday: dayOf(m[2])})],
	[new RegExp(`(?:${EVERY} (?:month )?on the|monthly on the|${EVERY}) ([0-9]{1,2})(?:st|nd|rd|th)${OF_MONTH}`, 'i'),
		m => {
			const day = parseInt(m[1], 10)
			return day >= 1 && day <= 31 ? {kind: 'monthDay', interval: 1, day} : null
		}],
	[new RegExp(`${EVERY} (?:(${COUNT}) weeks? on |(other) )?(${DAY_LIST})`, 'i'),
		m => {
			const days = m[3].split(/, ?| and | ?& ?|, and /i).map(dayOf).filter(Boolean)
			const interval = m[1] ? toCount(m[1]) : m[2] ? 2 : 1
			return days.length ? {kind: 'weekly', interval, days: sortDays(days)} : null
		}],
	[new RegExp(`${EVERY} (${COUNT}) months?`, 'i'),
		m => {
			const interval = toCount(m[1])
			return interval >= 2 ? {kind: 'months', interval} : null // "every month" is Vikunja's
		}],
	[/(?:biannually|semiannually|semi-annually)/i, () => ({kind: 'months', interval: 6})],
	[new RegExp(`${EVERY} (${COUNT}) years?`, 'i'), m => ({kind: 'years', interval: toCount(m[1])})],
	[/(?:biennially)/i, () => ({kind: 'years', interval: 2})],
	[new RegExp(`(?:yearly|annually|${EVERY} year)`, 'i'), () => ({kind: 'years', interval: 1})],
]

/**
 * Finds a calendar repeat in a quick add title and swaps it for the plain
 * Vikunja word ("weekly", "monthly", "yearly"), so Vikunja's magic still sets
 * a due date and reminders as for any repeating task; the rule then replaces
 * its interval. Null when there's none.
 */
export function extractRepeatRule(text: string): {text: string, rule: RepeatRule | null} {
	for (const [pattern, build] of patterns) {
		const bounded = new RegExp(`(^| )${pattern.source}(?= |$)`, pattern.flags)
		const match = bounded.exec(text)
		if (!match) continue
		const groups = Object.assign([...match.slice(1)], {index: match.index, input: text}) as unknown as RegExpExecArray
		const rule = build(groups)
		if (!rule || rule.interval < 1 || rule.interval > 100) continue
		const replaced = text.slice(0, match.index) + match[1] + vikunjaWord(rule) + text.slice(match.index + match[0].length)
		return {text: replaced, rule}
	}
	return {text, rule: null}
}
