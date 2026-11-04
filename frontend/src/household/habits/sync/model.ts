// Habits in the shared offline engine (household/sync/engine.ts): the data a
// phone keeps (today's habits and the routines), the changes it can queue
// (ticks and skips), and how a queued change shows before the server has it.
// Pure functions, tested in model.test.ts. The server works out the real
// progress (streaks) when the changes arrive: POST /api/v2/habits/sync.

import type {SyncOp as ApiSyncOp, SyncResult as ApiSyncResult} from '@/client/generated'

import {mondayOf} from '../grid'
import {normalizeGroup, normalizeHabit, type HabitGroupResponse, type HabitResponse} from '../queries'
import {newUid} from '../../shopping/sync/model'
import type {SyncAnswer} from '../../sync/engine'
import {fromKey} from '../../menu/week'

export type HabitOpType = NonNullable<ApiSyncOp['type']>

/** A queued change, as the server takes it. */
export interface HabitOp {
	id: string
	type: HabitOpType
	habit_id: number
	uid?: string // tick, untick
	day?: string // tick, skip, unskip: YYYY-MM-DD on the phone's calendar
}

export interface HabitsData {
	habits: HabitResponse[]
	groups: HabitGroupResponse[]
	/** False until the first answer: "no habits yet" isn't shown while loading. */
	loaded: boolean
}

export interface HabitsResult extends SyncAnswer {
	habits: HabitResponse[]
	groups: HabitGroupResponse[]
}

export function emptyData(): HabitsData {
	return {habits: [], groups: [], loaded: false}
}

export function normalizeSyncResult(r: ApiSyncResult): HabitsResult {
	return {
		cursor: 0, // every answer is the whole picture
		habits: (r.habits ?? []).map(normalizeHabit),
		groups: (r.groups ?? []).map(normalizeGroup),
		rejected: (r.rejected ?? []).map(x => ({opId: x.op_id ?? '', reason: x.reason ?? ''})),
	}
}

/** The server's answer replaces the copy. */
export function applyResult(_data: HabitsData, r: HabitsResult): HabitsData {
	return {habits: r.habits, groups: r.groups, loaded: true}
}

/** Monday is 0, as in a habit's weekdays string. */
function weekdayIndex(day: string): number {
	return (fromKey(day).getDay() + 6) % 7
}

/**
 * The habit as it stands on a later day than its progress was worked out
 * for: the phone was offline over midnight. Today starts empty (a weekly
 * habit keeps its week's ticks until Monday); streaks wait for the server.
 */
export function rollToDay(h: HabitResponse, day: string): HabitResponse {
	const p = h.progress
	if (!p.today || day <= p.today) return h
	const sameWeek = h.kind === 'weekly' && mondayOf(day) === mondayOf(p.today)
	const count = sameWeek ? p.today_count : 0
	const done = h.kind !== 'bad' && count >= p.goal
	const scheduled = h.kind === 'weekly' || h.weekdays[weekdayIndex(day)] === '1'
	return {
		...h,
		progress: {
			...p,
			today: day,
			today_count: count,
			today_entries: sameWeek ? p.today_entries : [],
			done_today: done,
			skipped: sameWeek && p.skipped,
			due_today: h.kind !== 'bad' && !h.paused && scheduled && !done && !(sameWeek && p.skipped),
			this_week: mondayOf(day) === mondayOf(p.today) ? p.this_week : 0,
		},
	}
}

/** Today's ticks changed: the count, and whether that reaches the goal. */
function withEntries(h: HabitResponse, entries: string[]): HabitResponse {
	const p = h.progress
	const count = entries.length
	// Bad habits have no goal: counting one never makes them "done".
	const done = h.kind !== 'bad' && count >= p.goal
	// Due stays as it was, unless the change reached (or left) the goal.
	const wasDue = p.due_today || p.done_today
	return {
		...h,
		progress: {
			...p,
			today_entries: entries,
			today_count: count,
			this_week: p.this_week + (count - p.today_count),
			done_today: done,
			due_today: wasDue && !done,
		},
	}
}

function applyToHabit(habit: HabitResponse, op: HabitOp): HabitResponse {
	// A change made on a later day than the copy: move the copy on first.
	const h = op.day ? rollToDay(habit, op.day) : habit
	const p = h.progress
	switch (op.type) {
		case 'tick':
			// A late tick on an earlier day shows in the history grid, not today.
			return op.day === p.today && op.uid && !p.today_entries.includes(op.uid)
				? withEntries(h, [...p.today_entries, op.uid])
				: h
		case 'untick':
			return withEntries(h, p.today_entries.filter(u => u !== op.uid))
		case 'skip':
			return op.day === p.today && !p.done_today
				? {...h, progress: {...p, skipped: true, due_today: false}}
				: h
		case 'unskip':
			return op.day === p.today && p.skipped
				? {...h, progress: {...p, skipped: false, due_today: !h.paused && !p.done_today}}
				: h
	}
}

export function applyOp(data: HabitsData, op: HabitOp): HabitsData {
	return {...data, habits: data.habits.map(h => h.id === op.habit_id ? applyToHabit(h, op) : h)}
}

export const habitsModel = {empty: emptyData, applyOp, applyResult}

// ---- Building ops from what the user did ----

export function tickOp(habit: HabitResponse, day: string): HabitOp {
	return {id: newUid(), type: 'tick', habit_id: habit.id, uid: newUid(), day}
}

export function untickOp(habit: HabitResponse, uid: string): HabitOp {
	return {id: newUid(), type: 'untick', habit_id: habit.id, uid}
}

export function skipOp(habit: HabitResponse, day: string, skip: boolean): HabitOp {
	return {id: newUid(), type: skip ? 'skip' : 'unskip', habit_id: habit.id, day}
}
