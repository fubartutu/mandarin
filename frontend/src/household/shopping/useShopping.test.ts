import {describe, expect, it} from 'vitest'

import {normalizeItem} from './queries'
import {findDuplicate} from './adding'
import {groupItems} from './useShopping'

const categories = [
	{id: 1, title: 'Fruit & Veg', position: 1},
	{id: 2, title: 'Dairy', position: 2},
	{id: 3, title: 'Frozen', position: 3},
]

const item = (id: number, name: string, extra: Record<string, unknown> = {}) =>
	normalizeItem({id, name, list_id: 1, ...extra})

describe('groupItems', () => {
	it('groups open items by category order and drops empty groups', () => {
		const {groups} = groupItems([
			item(1, 'Milk', {category_id: 2}),
			item(2, 'Apples', {category_id: 1}),
			item(3, 'Cheese', {category_id: 2}),
		], categories, 'Other')

		expect(groups.map(g => g.title)).toEqual(['Fruit & Veg', 'Dairy'])
		expect(groups[1].items.map(i => i.name)).toEqual(['Milk', 'Cheese'])
	})

	it('puts uncategorised and unknown-category items last', () => {
		const {groups} = groupItems([
			item(1, 'Batteries'),
			item(2, 'Ice', {category_id: 99}),
			item(3, 'Peas', {category_id: 3}),
		], categories, 'Other')

		expect(groups.map(g => [g.categoryId, g.title])).toEqual([[3, 'Frozen'], [0, 'Other']])
		expect(groups[1].items.map(i => i.name)).toEqual(['Batteries', 'Ice'])
	})

	it('moves ticked items out, most recently ticked first', () => {
		const {groups, done} = groupItems([
			item(1, 'Milk', {category_id: 2, done: true, done_at: '2026-10-20T10:00:00Z'}),
			item(2, 'Bread', {done: true, done_at: '2026-10-20T11:00:00Z'}),
			item(3, 'Apples', {category_id: 1}),
		], categories, 'Other')

		expect(groups.map(g => g.title)).toEqual(['Fruit & Veg'])
		expect(done.map(i => i.name)).toEqual(['Bread', 'Milk'])
	})

	it('orders within a group by position, then id', () => {
		const {groups} = groupItems([
			item(3, 'C', {category_id: 1, position: 1}),
			item(2, 'B', {category_id: 1}),
			item(1, 'A', {category_id: 1}),
		], categories, 'Other')

		expect(groups[0].items.map(i => i.name)).toEqual(['A', 'B', 'C'])
	})
})

describe('findDuplicate', () => {
	const items = [
		item(1, 'Milk', {quantity: 2, unit: 'L'}),
		item(2, 'Bread', {done: true}),
		item(3, 'Eggs', {note: 'free range'}),
	]
	const add = (name: string, quantity = 0, unit = '') => ({list_id: 1, name, quantity, unit})

	it('finds an unticked item with the same name and amount', () => {
		expect(findDuplicate(items, add(' milk ', 2, 'l'))?.id).toBe(1)
	})

	it('ignores other amounts, ticked items, items with a note and other lists', () => {
		expect(findDuplicate(items, add('Milk', 1, 'L'))).toBeUndefined()
		expect(findDuplicate(items, add('Bread'))).toBeUndefined()
		expect(findDuplicate(items, add('Eggs'))).toBeUndefined()
		expect(findDuplicate(items, {...add('Milk', 2, 'L'), list_id: 2})).toBeUndefined()
	})
})
