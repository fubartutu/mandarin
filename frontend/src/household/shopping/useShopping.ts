// Read side of the shopping module. A composable is a function that sets up
// reactive state; components call it inside <script setup>. Everything here
// reads the sync engine's copy on the phone, so it works offline.

import {computed, toValue, type MaybeRefOrGetter} from 'vue'

import type {ShoppingCategoryResponse, ShoppingItemResponse} from './queries'
import {useShoppingSync} from './sync/useShoppingSync'

export interface ItemGroup {
	categoryId: number // 0 = uncategorised
	title: string
	items: ShoppingItemResponse[]
}

/**
 * Splits a list's items into category groups (in category order, empty
 * groups left out, uncategorised last) and the ticked items.
 */
export function groupItems(
	items: ShoppingItemResponse[],
	categories: ShoppingCategoryResponse[],
	uncategorisedTitle: string,
): {groups: ItemGroup[], done: ShoppingItemResponse[]} {
	// Items added offline have no id yet (0); they sort after the rest, in the order they were added.
	const id = (i: ShoppingItemResponse) => i.id || Number.MAX_SAFE_INTEGER
	const byPosition = (a: ShoppingItemResponse, b: ShoppingItemResponse) => a.position - b.position || id(a) - id(b)
	const open = items.filter(i => !i.done)
	const known = new Set(categories.map(c => c.id))

	const groups: ItemGroup[] = categories
		.map(c => ({categoryId: c.id, title: c.title, items: open.filter(i => i.category_id === c.id).sort(byPosition)}))
		.filter(g => g.items.length > 0)

	// Items whose category is 0 or was deleted meanwhile.
	const loose = open.filter(i => !known.has(i.category_id)).sort(byPosition)
	if (loose.length > 0) {
		groups.push({categoryId: 0, title: uncategorisedTitle, items: loose})
	}

	// Most recently ticked first, like a receipt.
	const done = items.filter(i => i.done).sort((a, b) => (b.done_at ?? '').localeCompare(a.done_at ?? ''))
	return {groups, done}
}

const byPositionThenId = (a: {position: number, id: number}, b: {position: number, id: number}) =>
	a.position - b.position || a.id - b.id

/** Lists (with shops), categories and the catalogue, from the phone's copy. */
export function useShoppingData() {
	const sync = useShoppingSync()
	const data = sync.data
	return {
		lists: computed(() => [...data.value.lists.values()].sort(byPositionThenId)),
		categories: computed(() => [...data.value.categories.values()].sort(byPositionThenId)),
		catalog: computed(() => [...data.value.catalog.values()]),
		// A new phone has nothing to show until its first sync (unless that can't happen yet).
		isPending: computed(() => data.value.lists.size === 0 && sync.lastSyncedAt.value === null &&
			!['offline', 'error'].includes(sync.status.value)),
	}
}

/** Items a shop's tab shows: everything except what the shop doesn't stock. 0 = all shops. */
export function itemsForShop(items: ShoppingItemResponse[], shopId: number): ShoppingItemResponse[] {
	return shopId === 0 ? items : items.filter(i => !i.not_stocked_shop_ids.includes(shopId))
}

export function useShoppingItems(listId: MaybeRefOrGetter<number>) {
	const sync = useShoppingSync()
	return {
		items: computed(() => [...sync.data.value.items.values()].filter(i => i.list_id === toValue(listId))),
	}
}
