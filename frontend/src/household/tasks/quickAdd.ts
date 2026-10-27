// Mandarin: quick add's household magic, around Vikunja's. useQuickAddTask
// (FORK lines) reads it off the title before Vikunja's magic does, applies it
// to the built task, and saves the repeat end once the task exists.

import type {Task} from '@/client/generated'
import {error} from '@/message'
import {i18n} from '@/i18n'
import {PREFIXES, type PrefixMode} from '@/modules/quickAddMagic'
import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'
import {extractRepeatEnd, type RepeatEnd} from './repeatEnd'
import {extractFromCompletion, extractStarting, extractTodoistPriority} from './todoistMagic'
import {saveRepeatEnd} from './queries'

export interface HouseholdMagic {
	repeatEnd: RepeatEnd | null
	/** "every!": seconds to repeat after, counted from when it's done; 0 = not set. */
	fromCompletion: number
	/** From "p1".."p4", already in Vikunja's scale. */
	priority: number | null
}

const NONE: HouseholdMagic = {repeatEnd: null, fromCompletion: 0, priority: null}

/** Takes the household phrases off a title; nothing when quick add magic is off. */
export function readHouseholdMagic(title: string, mode: PrefixMode, now = new Date()): {text: string, magic: HouseholdMagic} {
	if (PREFIXES[mode] === undefined) return {text: title, magic: NONE}
	const completion = extractFromCompletion(title)
	const priority = extractTodoistPriority(completion.text)
	const end = extractRepeatEnd(priority.text, now)
	return {
		text: extractStarting(end.text, now),
		magic: {repeatEnd: end.end, fromCompletion: completion.fromCompletion, priority: priority.priority},
	}
}

/** Adds what Vikunja's magic can't set to the task it built. */
export function applyHouseholdMagic(task: Task, magic: HouseholdMagic | null): Task {
	if (!magic) return task
	const result = {...task}
	if (magic.fromCompletion > 0) {
		result.repeat_mode = TASK_REPEAT_MODES.REPEAT_MODE_FROM_CURRENT_DATE
		result.repeat_after = magic.fromCompletion
	}
	// Vikunja's own "!3" wins over "p1".
	if (magic.priority !== null && !result.priority) {
		result.priority = magic.priority
	}
	return result
}

/** Saves the typed end on the new task. A failure never fails the task itself. */
export async function saveHouseholdMagic(task: {id?: number, repeat_after?: number, repeat_mode?: number}, magic: HouseholdMagic | null) {
	const end = magic?.repeatEnd
	const repeats = (task.repeat_after ?? 0) > 0 || task.repeat_mode === TASK_REPEAT_MODES.REPEAT_MODE_MONTH
	if (!end || !task.id || !repeats) return
	try {
		await saveRepeatEnd(task.id, end)
	} catch {
		error({message: i18n.global.t('household.tasks.repeatEnd.saveFailed')})
	}
}
