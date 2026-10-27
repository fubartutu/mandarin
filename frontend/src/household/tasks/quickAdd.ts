// Mandarin: quick add's half of the repeat end. useQuickAddTask (FORK lines)
// calls these: one before Vikunja's magic reads the title, one after the
// task and its labels are saved.

import {error} from '@/message'
import {i18n} from '@/i18n'
import {PREFIXES, type PrefixMode} from '@/modules/quickAddMagic'
import {extractRepeatEnd, type RepeatEnd} from './repeatEnd'
import {saveRepeatEnd} from './queries'

/** extractRepeatEnd, unless quick add magic is switched off in the user's settings. */
export function extractQuickAddRepeatEnd(title: string, mode: PrefixMode): {text: string, end: RepeatEnd | null} {
	if (PREFIXES[mode] === undefined) return {text: title, end: null}
	return extractRepeatEnd(title)
}

/** Saves the typed end on the new task. A failure never fails the task itself. */
export async function saveQuickAddRepeatEnd(task: {id?: number, repeat_after?: number, repeat_mode?: number}, end: RepeatEnd | null) {
	const repeats = (task.repeat_after ?? 0) > 0 || task.repeat_mode === 1 // 1 = monthly
	if (!end || !task.id || !repeats) return
	try {
		await saveRepeatEnd(task.id, end)
	} catch {
		error({message: i18n.global.t('household.tasks.repeatEnd.saveFailed')})
	}
}
