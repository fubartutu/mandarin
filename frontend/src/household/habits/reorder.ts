// Moving a habit or a routine up or down. moveItem is pure, tested in
// reorder.test.ts.

import {ref, watch} from 'vue'

interface Positioned {
	id: number
	position: number
}

/**
 * The new positions after moving `items[index]` one place up (-1) or down
 * (+1) in `items` (already in the order shown). The whole list is numbered
 * 0, 1, 2… again, so equal or gappy positions sort out too, and only the
 * items whose position changes are returned: each one is a write, and
 * SQLite takes one writer at a time. Empty when the move goes nowhere.
 */
export function moveItem<T extends Positioned>(items: T[], index: number, by: -1 | 1): {item: T, position: number}[] {
	const to = index + by
	if (index < 0 || index >= items.length || to < 0 || to >= items.length) return []
	const order = [...items]
	const [moved] = order.splice(index, 1)
	order.splice(to, 0, moved)
	return order.flatMap((item, position) => item.position === position ? [] : [{item, position}])
}

/**
 * Positions moved here but not back from the server yet, so the list shows
 * the new order at once and a second tap works from it. An entry goes once
 * `source` (the server's copy) has that position, or when its write fails.
 */
export function usePendingPositions(source: () => Positioned[]) {
	const pending = ref(new Map<number, number>())

	watch(source, items => {
		for (const item of items) {
			if (pending.value.get(item.id) === item.position) pending.value.delete(item.id)
		}
	})

	/** A copy of the items with pending positions applied. */
	function apply<T extends Positioned>(items: T[]): T[] {
		return items.map(item => pending.value.has(item.id) ? {...item, position: pending.value.get(item.id)!} : item)
	}

	return {
		apply,
		set: (id: number, position: number) => pending.value.set(id, position),
		drop: (id: number) => pending.value.delete(id),
	}
}
