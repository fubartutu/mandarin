// Mandarin: when a repeating task stops repeating, typed into quick add the
// Todoist way: "water plants daily until 5 Nov", "every week until friday",
// "take tablets daily 10 times", "daily for 3 weeks". Vikunja's quick add magic
// doesn't know these, so useQuickAddTask (FORK) takes them out of the title
// first and saves them after the task is created.

import {getRepeats} from '@/modules/quickAddMagic/repeatParser'
import {parseDate} from '@/modules/quickAddMagic/dateParser'

/** What ends the series; the server's TaskRepeatEnd. */
export interface RepeatEnd {
	/** The last moment an occurrence may be due (the end of the chosen day). */
	until: Date | null
	/** Occurrences left, the current one included. 0 = no limit. */
	repeatsLeft: number
}

export const MAX_REPEATS_LEFT = 1000

const NUMBER_WORDS: Record<string, number> = {
	one: 1, two: 2, three: 3, four: 4, five: 5, six: 6, seven: 7, eight: 8, nine: 9, ten: 10,
	eleven: 11, twelve: 12, twenty: 20, thirty: 30,
}
const NUMBER = `([0-9]+|${Object.keys(NUMBER_WORDS).join('|')})`

function toNumber(word: string): number {
	return NUMBER_WORDS[word.toLowerCase()] ?? parseInt(word, 10)
}

/** The end of the day d falls on, in local time: "until 5 Nov" includes that evening. */
export function endOfDay(d: Date): Date {
	return new Date(d.getFullYear(), d.getMonth(), d.getDate(), 23, 59, 59)
}

// "for 10 times", "10 times", "x10"
const COUNT = new RegExp(`(^| )(for )?(${NUMBER} times|x([0-9]+))(?= |$)`, 'i')
// "for 3 weeks", "for two months"
const SPAN = new RegExp(`(^| )for ${NUMBER} (days?|weeks?|months?|years?)(?= |$)`, 'i')
// "until …", "till …", "ending …": the date follows, then maybe more of the title.
const UNTIL = /(^| )(until|till|ending) /i
// A time typed after the end date belongs to the task: "until fri at 5pm".
const TIME = /( (at|@) [0-9][0-9]?(:[0-9][0-9])?( ?(a|p)m)?)(?= |$)/i

/**
 * Takes the repeat end out of a quick add title. Only titles that also repeat
 * ("daily", "every 2 weeks", …) are touched, so "read until chapter 5" stays as typed.
 */
export function extractRepeatEnd(text: string, now = new Date()): {text: string, end: RepeatEnd | null} {
	if (getRepeats(text).repeats === null) {
		return {text, end: null}
	}

	const end: RepeatEnd = {until: null, repeatsLeft: 0}

	const count = COUNT.exec(text)
	if (count) {
		const n = toNumber(count[4] ?? count[5])
		if (n >= 1 && n <= MAX_REPEATS_LEFT) {
			end.repeatsLeft = n
			text = cut(text, count.index, count[0].length)
		}
	}

	const span = SPAN.exec(text)
	if (span) {
		const n = toNumber(span[2])
		const until = new Date(now)
		const unit = span[3].toLowerCase().replace(/s$/, '')
		if (unit === 'day') until.setDate(until.getDate() + n - 1)
		if (unit === 'week') until.setDate(until.getDate() + 7 * n - 1)
		if (unit === 'month') until.setMonth(until.getMonth() + n, until.getDate() - 1)
		if (unit === 'year') until.setFullYear(until.getFullYear() + n, until.getMonth(), until.getDate() - 1)
		end.until = endOfDay(until)
		text = cut(text, span.index, span[0].length)
	}

	const until = UNTIL.exec(text)
	if (until && end.until === null) {
		const before = text.slice(0, until.index)
		let after = text.slice(until.index + until[0].length)
		// Keep a time for the task's own due date, not the end date.
		let time = ''
		const t = TIME.exec(after)
		if (t) {
			time = t[1]
			after = cut(after, t.index, t[0].length)
		}
		// A copy: parseDate's weekday branch moves the date it is given.
		const parsed = parseDate(after, new Date(now))
		if (parsed.date !== null) {
			end.until = endOfDay(parsed.date)
			text = [before, time.trim(), parsed.newText.trim()].filter(Boolean).join(' ')
		}
	}

	if (end.until === null && end.repeatsLeft === 0) {
		return {text, end: null}
	}
	return {text: text.replace(/ {2,}/g, ' ').trim(), end}
}

function cut(text: string, index: number, length: number): string {
	return (text.slice(0, index) + ' ' + text.slice(index + length)).replace(/ {2,}/g, ' ').trim()
}
