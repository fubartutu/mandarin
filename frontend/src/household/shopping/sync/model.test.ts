import {describe, expect, it} from 'vitest'

import {normalizeCatalogEntry, normalizeItem} from '../queries'
import {
	applyOp,
	applyResult,
	createOp,
	deleteOp,
	emptyData,
	notStockedOp,
	rebase,
	updateOp,
	type ShoppingData,
	type SyncResult,
} from './model'

function result(extra: Partial<SyncResult> = {}): SyncResult {
	return {
		cursor: 1, full: false, lists: [], deletedListIds: [], categories: [], deletedCategoryIds: [],
		catalog: [], deletedCatalogIds: [], items: [], deletedItemUids: [], conflicts: [], rejected: [],
		...extra,
	}
}

const milkEntry = normalizeCatalogEntry({id: 7, name: 'Milk', category_id: 5, has_photo: true, photo_id: 3, not_stocked_shop_ids: [2]})
const milk = normalizeItem({id: 1, uid: 'u-milk', list_id: 1, name: 'Milk', catalog_item_id: 7})

function withMilk(): ShoppingData {
	return applyResult(emptyData(), result({full: true, catalog: [milkEntry], items: [milk]}))
}

describe('applyResult', () => {
	it('copies photo and not-stocked info from the catalogue onto items', () => {
		const item = withMilk().items.get('u-milk')
		expect(item?.has_photo).toBe(true)
		expect(item?.not_stocked_shop_ids).toEqual([2])
	})

	it('merges changes and removes deletions', () => {
		const bread = normalizeItem({id: 2, uid: 'u-bread', list_id: 1, name: 'Bread'})
		const data = applyResult(withMilk(), result({items: [bread], deletedItemUids: ['u-milk']}))
		expect([...data.items.keys()]).toEqual(['u-bread'])
		expect(data.catalog.size).toBe(1)
	})

	it('a full copy replaces everything', () => {
		const data = applyResult(withMilk(), result({full: true}))
		expect(data.items.size).toBe(0)
		expect(data.catalog.size).toBe(0)
	})
})

describe('applyOp', () => {
	it('adds a new item, linked to its catalogue entry by name', () => {
		const op = createOp({list_id: 1, name: 'milk', quantity: 2, unit: 'L', category_id: 5})
		const item = applyOp(emptyData(), op).items.get(op.item_uid)
		expect(item?.id).toBe(0)
		expect(item?.name).toBe('milk')

		const linked = applyOp(withMilk(), op).items.get(op.item_uid)
		expect(linked?.catalog_item_id).toBe(7)
		expect(linked?.has_photo).toBe(true)
	})

	it('ticks with a done time and unticks without one', () => {
		const ticked = applyOp(withMilk(), updateOp(milk, {done: true}, Date.UTC(2026, 9, 1))!)
		expect(ticked.items.get('u-milk')?.done_at).toBe('2026-10-01T00:00:00.000Z')
		const unticked = applyOp(ticked, updateOp(ticked.items.get('u-milk')!, {done: false})!)
		expect(unticked.items.get('u-milk')?.done_at).toBeUndefined()
	})

	it('deletes', () => {
		expect(applyOp(withMilk(), deleteOp(milk)).items.size).toBe(0)
	})

	it('marks not-stocked on the catalogue entry, so its other items follow', () => {
		const other = normalizeItem({id: 9, uid: 'u-milk2', list_id: 2, name: 'Milk', catalog_item_id: 7})
		const data = applyResult(emptyData(), result({full: true, catalog: [milkEntry], items: [milk, other]}))
		const next = applyOp(data, notStockedOp(milk, [4, 5]))
		expect(next.items.get('u-milk2')?.not_stocked_shop_ids).toEqual([4, 5])
		expect(next.catalog.get(7)?.not_stocked_shop_ids).toEqual([4, 5])
	})

	it('adds an item for one shop, and moves it to any shop', () => {
		const op = createOp({list_id: 1, name: 'Brie', quantity: 0, unit: '', category_id: 0, shop_id: 3})
		expect(op.set?.shop_id).toBe(3)
		const data = applyOp(emptyData(), op)
		const brie = data.items.get(op.item_uid)!
		expect(brie.shop_id).toBe(3)

		const move = updateOp(brie, {shop_id: 0})!
		expect(move.set).toEqual({shop_id: 0})
		expect(move.before).toEqual({shop_id: 3})
		expect(applyOp(data, move).items.get(op.item_uid)?.shop_id).toBe(0)
	})

	it('leaves the shop out of an add for any shop', () => {
		const op = createOp({list_id: 1, name: 'Milk', quantity: 0, unit: '', category_id: 0})
		expect(op.set).not.toHaveProperty('shop_id')
	})

	it("doesn't change the data it was given", () => {
		const data = withMilk()
		applyOp(data, deleteOp(milk))
		expect(data.items.size).toBe(1)
	})
})

describe('updateOp', () => {
	it('sends only what changed, with the old values', () => {
		const op = updateOp(milk, {...milk, note: 'lactose free'})
		expect(op?.set).toEqual({note: 'lactose free'})
		expect(op?.before).toEqual({note: ''})
	})

	it('sends quantity and unit together', () => {
		const op = updateOp(milk, {quantity: 2})
		expect(op?.set).toEqual({quantity: 2, unit: ''})
		expect(op?.before).toEqual({quantity: 0, unit: ''})
	})

	it('is null when nothing changed', () => {
		expect(updateOp(milk, {...milk})).toBeNull()
	})
})

describe('rebase', () => {
	it('replays waiting changes on top of what the server said', () => {
		const tick = updateOp(milk, {done: true})!
		// The server's answer doesn't include the tick yet, but a new note from someone else.
		const confirmed = applyResult(withMilk(), result({items: [{...milk, note: 'from the other phone'}]}))
		const shown = rebase(confirmed, [tick]).items.get('u-milk')
		expect(shown?.done).toBe(true)
		expect(shown?.note).toBe('from the other phone')
	})
})
