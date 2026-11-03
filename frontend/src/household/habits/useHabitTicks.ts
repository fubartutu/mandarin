import {newTickUid, type HabitResponse, useTickMutation, useUntickMutation} from './queries'

/**
 * Ticking from the Habits page and the Home card.
 * A one-a-day habit toggles; a habit with a goal above one (or a bad habit)
 * counts up with each tap, and `undo` takes the last tap off.
 */
export function useHabitTicks() {
	const tick = useTickMutation()
	const untick = useUntickMutation()

	function add(habit: HabitResponse, day = habit.progress.today) {
		tick.mutate({habit, day, uid: newTickUid()})
	}

	function undo(habit: HabitResponse) {
		const uid = habit.progress.today_entries.at(-1)
		if (uid) untick.mutate({habit, uid})
	}

	function tap(habit: HabitResponse) {
		if (habit.kind !== 'bad' && habit.progress.goal === 1 && habit.progress.done_today) undo(habit)
		else add(habit)
	}

	return {add, undo, tap}
}
