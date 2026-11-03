import type {HabitResponse} from './queries'
import {skipOp, tickOp, untickOp} from './sync/model'
import {useHabitsSync} from './sync/useHabitsSync'

/**
 * Ticking from the Habits page, the Home card and the grid. Every change goes
 * through the offline queue: it shows at once and is sent when there's signal.
 * A one-a-day habit toggles; a habit with a goal above one (or a bad habit)
 * counts up with each tap, and `undo` takes the last tap off.
 */
export function useHabitTicks() {
	const sync = useHabitsSync()

	function add(habit: HabitResponse, day = sync.today.value) {
		void sync.change(tickOp(habit, day))
	}

	function remove(habit: HabitResponse, uid: string) {
		void sync.change(untickOp(habit, uid))
	}

	function undo(habit: HabitResponse) {
		const uid = habit.progress.today_entries.at(-1)
		if (uid) remove(habit, uid)
	}

	function tap(habit: HabitResponse) {
		if (habit.kind !== 'bad' && habit.progress.goal === 1 && habit.progress.done_today) undo(habit)
		else add(habit)
	}

	function skip(habit: HabitResponse, on: boolean) {
		void sync.change(skipOp(habit, sync.today.value, on))
	}

	return {add, remove, undo, tap, skip}
}
