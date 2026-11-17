// Breaking a big task down (POST /api/v2/tasks/{task}/breakdown): pure
// helpers for the review panel, kept apart from the component so they're tested.

import type {Task} from '@/client/generated'

/** steps: each item becomes a subtask. chunks: a checklist in the description. */
export type BreakdownMode = 'steps' | 'chunks'

/** Server limit (models.MaxBreakdownItems). */
export const MAX_BREAKDOWN_ITEMS = 15

export interface BreakdownItem {
	/** Local only, for Vue's :key while items move. */
	key: number
	title: string
	/** The AI's rough estimate; 0 when typed or unknown. */
	minutes: number
}

let nextKey = 1

export function newItem(title = '', minutes = 0): BreakdownItem {
	return {key: nextKey++, title, minutes}
}

/**
 * A subtask is usually one stage of a bigger job, so it gets small chunks;
 * anything else gets broken into stages first.
 */
export function defaultMode(task: Pick<Task, 'related_tasks'>): BreakdownMode {
	return (task.related_tasks?.parenttask?.length ?? 0) > 0 ? 'chunks' : 'steps'
}

/**
 * Typed or pasted lines as items: one per line, list markers ("-", "*",
 * "1.", "[ ]") taken off, blank lines dropped.
 */
export function parseTyped(text: string): BreakdownItem[] {
	return text.split('\n')
		.map(line => line.trim().replace(/^(?:[-*•]|\d+[.)]|\[[ xX]?\])\s*/, '').trim())
		.filter(line => line !== '')
		.map(line => newItem(line))
}

/**
 * What gets saved for an item. A chunk keeps its estimate in the checklist
 * ("Rake one strip (~10 min)"): knowing it's short helps you start. Steps
 * become tasks, whose titles stay clean.
 */
export function itemText(item: BreakdownItem, mode: BreakdownMode): string {
	const title = item.title.trim()
	if (mode !== 'chunks' || item.minutes <= 0 || title === '' || /\(~?\d+ ?min\)$/.test(title)) {
		return title
	}
	return `${title} (~${item.minutes} min)`
}

/** A copy of items with the one at index moved by delta (-1 up, 1 down), if it can move. */
export function moveItem<T>(items: readonly T[], index: number, delta: number): T[] {
	const to = index + delta
	const out = [...items]
	if (to < 0 || to >= out.length || index < 0 || index >= out.length) return out
	const [item] = out.splice(index, 1)
	out.splice(to, 0, item!)
	return out
}

/** Total minutes of the items, for "about 1 h 20 min". */
export function totalMinutes(items: readonly BreakdownItem[]): number {
	return items.reduce((sum, item) => sum + Math.max(0, item.minutes), 0)
}
