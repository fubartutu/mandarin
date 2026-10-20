// Read side of the shopping module. A composable is a function that sets up
// reactive state; components call it inside <script setup>.

import {computed, toValue, type MaybeRefOrGetter} from 'vue'
import {useQuery} from '@tanstack/vue-query'

import {
	shoppingCategoriesQuery,
	shoppingItemsQuery,
	shoppingListsQuery,
	type ShoppingCategoryResponse,
	type ShoppingItemResponse,
} from './queries'

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
	const byPosition = (a: ShoppingItemResponse, b: ShoppingItemResponse) => a.position - b.position || a.id - b.id
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

export function useShoppingLists() {
	const query = useQuery(shoppingListsQuery())
	return {
		lists: computed(() => query.data.value ?? []),
		isPending: query.isPending,
	}
}

export function useShoppingCategories() {
	const query = useQuery(shoppingCategoriesQuery())
	return {
		categories: computed(() => query.data.value ?? []),
		isPending: query.isPending,
	}
}

export function useShoppingItems(listId: MaybeRefOrGetter<number>) {
	// Passing a computed options object makes the query follow listId changes.
	const query = useQuery(computed(() => ({
		...shoppingItemsQuery(toValue(listId)),
		enabled: toValue(listId) > 0,
	})))
	return {
		items: computed(() => query.data.value ?? []),
		isPending: query.isPending,
	}
}
