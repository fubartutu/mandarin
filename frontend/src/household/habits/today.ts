// How the habits are laid out on the Habits page and the Home card. Pure
// functions, so they're tested without a server (today.test.ts).

import type {HabitGroupResponse, HabitResponse} from './queries'

/**
 * Where a habit stands today:
 * - due: still to do today (this week for weekly habits)
 * - done: today's goal is reached
 * - skipped / paused: today doesn't count
 * - rest: not one of the habit's days
 * - bad: a bad habit, which is only counted
 */
export type HabitState = 'due' | 'done' | 'skipped' | 'paused' | 'rest' | 'bad'

export function habitState(h: HabitResponse): HabitState {
	if (h.kind === 'bad') return 'bad'
	if (h.paused) return 'paused'
	if (h.progress.done_today) return 'done'
	if (h.progress.skipped) return 'skipped'
	if (h.progress.due_today) return 'due'
	return 'rest'
}

/** Habits first by position, then by when they were made. */
function byPosition(a: {position: number, id: number}, b: {position: number, id: number}) {
	return a.position - b.position || a.id - b.id
}

export interface HabitSection {
	/** The routine, or undefined for habits in none. */
	group?: HabitGroupResponse
	habits: HabitResponse[]
}

/**
 * The routines in order, each with its habits, then the habits in no
 * routine. Bad habits are left out (they get a section of their own), and
 * so are routines with no habits. A habit whose routine can't be seen
 * (someone else's personal one) counts as in none.
 */
export function habitSections(habits: HabitResponse[], groups: HabitGroupResponse[]): HabitSection[] {
	const good = habits.filter(h => h.kind !== 'bad').sort(byPosition)
	const known = new Set(groups.map(g => g.id))
	const sections: HabitSection[] = [...groups].sort(byPosition)
		.map(group => ({group, habits: good.filter(h => h.groupId === group.id)}))
	sections.push({habits: good.filter(h => !known.has(h.groupId))})
	return sections.filter(s => s.habits.length > 0)
}

export function badHabits(habits: HabitResponse[]): HabitResponse[] {
	return habits.filter(h => h.kind === 'bad').sort(byPosition)
}

/**
 * The Home card's list: what's still due today, then what's done, in the
 * page's order. Rest days, skipped, paused and bad habits stay off Home.
 */
export function homeHabits(habits: HabitResponse[], groups: HabitGroupResponse[]): HabitResponse[] {
	const ordered = habitSections(habits, groups).flatMap(s => s.habits)
	return [
		...ordered.filter(h => habitState(h) === 'due'),
		...ordered.filter(h => habitState(h) === 'done'),
	]
}

/** The weekday letters 0/1 string as Monday-first indexes (0 = Monday). */
export function weekdayIndexes(weekdays: string): number[] {
	return [...weekdays].flatMap((c, i) => c === '1' ? [i] : [])
}

/** Whether a habit is done on every day of the week. */
export function everyDay(weekdays: string): boolean {
	return weekdays === '1111111'
}
