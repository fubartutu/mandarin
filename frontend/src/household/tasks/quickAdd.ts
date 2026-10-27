// Mandarin: quick add's household magic, around Vikunja's. useQuickAddTask
// (FORK lines) reads it off the title before Vikunja's magic does, applies it
// to the built task, and saves the repeat rule and end once the task exists.

import type {Task} from '@/client/generated'
import {error} from '@/message'
import {i18n} from '@/i18n'
import {PREFIXES, type PrefixMode} from '@/modules/quickAddMagic'
import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'
import {getDateWithTime} from '@/helpers/time/getDateWithTime'
import {extractRepeatEnd, type RepeatEnd} from './repeatEnd'
import {extractRepeatRule, firstOccurrence, standInSeconds, toRRule, type RepeatRule} from './repeatRule'
import {extractFromCompletion, extractStarting, extractTodoistPriority} from './todoistMagic'
import {saveRepeat} from './queries'

export interface HouseholdMagic {
	/** "every mon, thu", "every 2nd tuesday", …; decides the dates instead of Vikunja's interval. */
	rule: RepeatRule | null
	repeatEnd: RepeatEnd | null
	/** "every!": seconds to repeat after, counted from when it's done; 0 = not set. */
	fromCompletion: number
	/** From "p1".."p4", already in Vikunja's scale. */
	priority: number | null
}

const NONE: HouseholdMagic = {rule: null, repeatEnd: null, fromCompletion: 0, priority: null}

/** Takes the household phrases off a title; nothing when quick add magic is off. */
export function readHouseholdMagic(title: string, mode: PrefixMode, now = new Date()): {text: string, magic: HouseholdMagic} {
	if (PREFIXES[mode] === undefined) return {text: title, magic: NONE}
	const completion = extractFromCompletion(title)
	const priority = extractTodoistPriority(completion.text)
	// Leaves Vikunja's own repeat word in its place, so the end and "starting" still see a repeat.
	const rule = extractRepeatRule(priority.text)
	const end = extractRepeatEnd(rule.text, now)
	return {
		text: extractStarting(end.text, now),
		magic: {rule: rule.rule, repeatEnd: end.end, fromCompletion: completion.fromCompletion, priority: priority.priority},
	}
}

/** Adds what Vikunja's magic can't set to the task it built. */
export function applyHouseholdMagic(task: Task, magic: HouseholdMagic | null): Task {
	if (!magic) return task
	const result = {...task}
	if (magic.rule) {
		// Vikunja's word set a due date and reminders; move the date to the
		// rule's first day and stand in for the interval (the rule decides).
		const from = result.due_date ? new Date(result.due_date) : getDateWithTime(new Date())
		result.due_date = firstOccurrence(magic.rule, from).toISOString()
		result.repeat_after = standInSeconds(magic.rule)
		result.repeat_mode = TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT
	} else if (magic.fromCompletion > 0) {
		result.repeat_mode = TASK_REPEAT_MODES.REPEAT_MODE_FROM_CURRENT_DATE
		result.repeat_after = magic.fromCompletion
	}
	// Vikunja's own "!3" wins over "p1".
	if (magic.priority !== null && !result.priority) {
		result.priority = magic.priority
	}
	return result
}

/** Saves the typed rule and end on the new task. A failure never fails the task itself. */
export async function saveHouseholdMagic(task: {id?: number, repeat_after?: number, repeat_mode?: number}, magic: HouseholdMagic | null) {
	if (!magic || (!magic.rule && !magic.repeatEnd)) return
	const repeats = (task.repeat_after ?? 0) > 0 || task.repeat_mode === TASK_REPEAT_MODES.REPEAT_MODE_MONTH
	if (!task.id || !repeats) return
	try {
		await saveRepeat(task.id, {
			rule: magic.rule ? toRRule(magic.rule) : '',
			until: magic.repeatEnd?.until ?? null,
			repeatsLeft: magic.repeatEnd?.repeatsLeft ?? 0,
		})
	} catch {
		error({message: i18n.global.t('household.tasks.repeatEnd.saveFailed')})
	}
}
