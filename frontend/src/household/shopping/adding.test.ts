import {describe, expect, it} from 'vitest'

import {planAdds, prepareNewItem} from './adding'
import {normalizeCatalogEntry, normalizeItem} from './queries'

const catalog = [normalizeCatalogEntry({id: 1, name: 'Plain flour', category_id: 4, quantity: 1, unit: 'kg'})]
const categories = [{id: 4, title: 'Pantry', position: 1}]
const add = (name: string, quantity = 0, unit = '') => ({list_id: 1, name, quantity, unit, category_id: 0})

describe('prepareNewItem', () => {
	it('takes the catalogue name, category and last amount when none is given', () => {
		expect(prepareNewItem({name: 'plain flour', quantity: 0, unit: ''}, 1, catalog, categories))
			.toEqual({list_id: 1, name: 'Plain flour', quantity: 1, unit: 'kg', shop_id: 0, category_id: 4})
	})

	it('keeps an amount that was given', () => {
		expect(prepareNewItem({name: 'plain flour', quantity: 250, unit: 'g'}, 1, catalog, categories).quantity).toBe(250)
	})
})

describe('planAdds', () => {
	const items = [
		normalizeItem({id: 1, uid: 'a', list_id: 1, name: 'Butter', quantity: 250, unit: 'g'}),
		normalizeItem({id: 2, uid: 'b', list_id: 1, name: 'Salt'}),
		normalizeItem({id: 3, uid: 'c', list_id: 1, name: 'Eggs', quantity: 6, done: true}),
	]

	it('tops up an item with the same name and unit', () => {
		const plan = planAdds([add('butter', 100, 'g')], items)
		expect(plan.topUps).toEqual([{item: items[0], quantity: 350}])
		expect(plan.creates).toEqual([])
	})

	it('skips an amount-less line already on the list, and adds the rest', () => {
		const plan = planAdds([add('salt'), add('Eggs', 3), add('Butter', 1, 'cup')], items)
		expect(plan.skipped).toEqual(['salt'])
		expect(plan.creates.map(c => c.name)).toEqual(['Eggs', 'Butter']) // eggs are ticked; cups aren't grams
	})

	it('combines repeats within the recipe', () => {
		const plan = planAdds([add('Milk', 100, 'mL'), add('milk', 200, 'mL'), add('butter', 50, 'g'), add('Butter', 50, 'g')], items)
		expect(plan.creates).toEqual([add('Milk', 300, 'mL')])
		expect(plan.topUps).toEqual([{item: items[0], quantity: 350}])
	})
})
