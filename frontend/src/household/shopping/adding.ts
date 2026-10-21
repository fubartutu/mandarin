// Adding items to a list: from the add box, or a recipe's ingredients. The
// new items are filled in from the catalogue the way the server will, so
// they appear in the right group straight away instead of jumping there.

import {findCatalogEntry} from './autocomplete'
import type {CatalogEntry, ShoppingCategoryResponse, ShoppingItemResponse} from './queries'
import type {NewItem} from './sync/model'

export interface ItemInput {
	name: string
	quantity: number
	unit: string
	category_id?: number // 0 or missing: the catalogue's category
}

/** A new item for listId, with blanks filled from the catalogue. */
export function prepareNewItem(
	input: ItemInput,
	listId: number,
	catalog: CatalogEntry[],
	categories: ShoppingCategoryResponse[],
): NewItem {
	const known = findCatalogEntry(catalog, input.name)
	const categoryId = categories.some(c => c.id === input.category_id)
		? input.category_id!
		: known?.category_id ?? 0
	// No amount typed: use last time's.
	const keepLast = known && input.quantity === 0 && input.unit === ''
	return {
		name: known?.name ?? input.name.trim(),
		quantity: keepLast ? known.quantity : input.quantity,
		unit: keepLast ? known.unit : input.unit,
		list_id: listId,
		category_id: categoryId,
	}
}

const same = (a: string, b: string) => a.trim().toLowerCase() === b.trim().toLowerCase()

/**
 * An unticked item already on the list that's the same as a new one: same
 * name (ignoring case), amount and no note. The server merges such adds made
 * on two phones offline; on one phone we don't make them in the first place.
 */
export function findDuplicate(
	items: ShoppingItemResponse[],
	item: Pick<ShoppingItemResponse, 'list_id' | 'name' | 'quantity' | 'unit'>,
): ShoppingItemResponse | undefined {
	return items.find(i => i.list_id === item.list_id && !i.done && same(i.name, item.name) &&
		i.quantity === item.quantity && same(i.unit, item.unit) && i.note.trim() === '')
}

export interface AddPlan {
	creates: NewItem[]
	/** Unticked items to top up: 500 g flour on the list + 250 g from a recipe. */
	topUps: {item: ShoppingItemResponse, quantity: number}[]
	/** Names already on the list that weren't changed, e.g. "salt" with no amount. */
	skipped: string[]
}

/**
 * What adding several items (a recipe's ingredients) to a list should do.
 * An item with the same name and unit as an unticked one gets its amount
 * added to it; one with no amount whose name is already there is skipped;
 * everything else becomes a new row. Items in the batch combine the same way.
 */
export function planAdds(newItems: NewItem[], items: ShoppingItemResponse[]): AddPlan {
	const plan: AddPlan = {creates: [], topUps: [], skipped: []}
	for (const add of newItems) {
		const sameName = (i: {list_id: number, name: string}) => i.list_id === add.list_id && same(i.name, add.name)
		const open = items.filter(i => !i.done && sameName(i))

		if (add.quantity === 0) {
			// "Salt to taste": having some on the list is enough.
			if (open.length > 0 || plan.creates.some(sameName)) plan.skipped.push(add.name)
			else plan.creates.push({...add})
			continue
		}
		const pending = plan.creates.find(c => sameName(c) && same(c.unit, add.unit))
		const topUp = plan.topUps.find(t => sameName(t.item) && same(t.item.unit, add.unit))
		const existing = open.find(i => same(i.unit, add.unit) && i.quantity > 0)
		if (pending) pending.quantity += add.quantity
		else if (topUp) topUp.quantity += add.quantity
		else if (existing) plan.topUps.push({item: existing, quantity: existing.quantity + add.quantity})
		else plan.creates.push({...add})
	}
	return plan
}
