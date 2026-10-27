// Mandarin: Todoist phrases that Vikunja's quick add magic doesn't know,
// read before it (see quickAdd.ts):
// - "every! 3 days": repeat from when it's done, not from the due date
// - "weekly starting nov 1": the first occurrence is that date
// - "p1".."p4": Todoist's priorities, p1 the most urgent

import {getRepeats} from '@/modules/quickAddMagic/repeatParser'
import {parseDate} from '@/modules/quickAddMagic/dateParser'
import {PRIORITIES} from '@/constants/priorities'

// "every!" is only Todoist's way of saying it; the rest is Vikunja's "every …".
const FROM_COMPLETION = /(^| )(every|each)!(?= )/i

const SECONDS_A_DAY = 24 * 3600
const UNIT_SECONDS: Record<string, number> = {
	hours: 3600,
	days: SECONDS_A_DAY,
	weeks: 7 * SECONDS_A_DAY,
	// Repeating from completion counts in seconds on the server, so months
	// and years are approximate.
	months: 30 * SECONDS_A_DAY,
	years: 365 * SECONDS_A_DAY,
}

/**
 * Turns "every!" into "every" and gives the interval in seconds to repeat
 * after once it's done (0 when there is no "every!").
 */
export function extractFromCompletion(text: string): {text: string, fromCompletion: number} {
	if (!FROM_COMPLETION.test(text)) return {text, fromCompletion: 0}
	const replaced = text.replace(FROM_COMPLETION, (_match, space: string, word: string) => space + word)
	// Only counts when Vikunja then reads a repeat out of it.
	const repeats = getRepeats(replaced).repeats
	const seconds = repeats ? repeats.amount * (UNIT_SECONDS[repeats.type] ?? 0) : 0
	if (seconds <= 0) return {text, fromCompletion: 0}
	return {text: replaced, fromCompletion: seconds}
}

// Todoist's p4 is its "no priority"; here it is low, which is what people mean when they type it.
const TODOIST_PRIORITIES: Record<string, number> = {
	'1': PRIORITIES.URGENT,
	'2': PRIORITIES.HIGH,
	'3': PRIORITIES.MEDIUM,
	'4': PRIORITIES.LOW,
}
const PRIORITY = /(^| )p([1-4])(?= |$)/i

export function extractTodoistPriority(text: string): {text: string, priority: number | null} {
	const match = PRIORITY.exec(text)
	if (!match) return {text, priority: null}
	const rest = (text.slice(0, match.index) + ' ' + text.slice(match.index + match[0].length)).replace(/ {2,}/g, ' ').trim()
	return {text: rest, priority: TODOIST_PRIORITIES[match[2]]}
}

const STARTING = /(^| )(starting|beginning)( on)? /i

/**
 * Drops the word "starting" before a date in a repeating title, so Vikunja's
 * magic reads the date as the first due date: "water plants daily starting nov 1".
 */
export function extractStarting(text: string, now = new Date()): string {
	if (getRepeats(text).repeats === null) return text
	const match = STARTING.exec(text)
	if (!match) return text
	const after = text.slice(match.index + match[0].length)
	// A copy: parseDate's weekday branch moves the date it is given.
	if (parseDate(after, new Date(now)).date === null) return text
	return (text.slice(0, match.index) + ' ' + after).replace(/ {2,}/g, ' ').trim()
}
