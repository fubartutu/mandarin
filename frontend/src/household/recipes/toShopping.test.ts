import {describe, expect, it} from 'vitest'

import {normalizeCatalogEntry} from '../shopping/queries'
import type {IngredientLine} from './queries'
import {ingredientToItem, splitLine} from './toShopping'

const titles = ['Fruit & Veg', 'Meat & Seafood', 'Dairy & Eggs', 'Pantry', 'Frozen', 'Bakery', 'Deli', 'Other']
const categories = titles.map((title, i) => ({id: i + 1, title, position: i + 1}))
const cat = (title: string) => categories.find(c => c.title === title)!.id

const catalog = [
	['Canned chickpeas', 'Pantry'], ['Canned tomatoes', 'Pantry'], ['Tomatoes', 'Fruit & Veg'],
	['Brown onions', 'Fruit & Veg'], ['Red onions', 'Fruit & Veg'], ['Olive oil', 'Pantry'],
	['Whole chicken', 'Meat & Seafood'], ['Chicken stock', 'Pantry'], ['Sliced chicken', 'Deli'],
	['Eggs', 'Dairy & Eggs'], ['Parmesan', 'Dairy & Eggs'], ['Fresh pasta', 'Other'],
].map(([name, title], i) => normalizeCatalogEntry({id: i + 1, name, category_id: cat(title)}))

/** A line as the server sends it; with only text, as one it couldn't split. */
const line = (text: string, parts: Partial<IngredientLine> = {}): IngredientLine =>
	({group: '', text, quantity: 0, unit: '', name: text, note: '', ...parts})

const add = (l: IngredientLine, factor = 1) => ingredientToItem(l, factor, catalog, categories)

describe('ingredientToItem', () => {
	it('turns a can of chickpeas into Chickpeas, 400 g can, in Pantry', () => {
		const expected = {name: 'Chickpeas', quantity: 400, unit: 'g can', category_id: cat('Pantry')}
		// Split by the server's simple parser…
		expect(add(line('400 g can chickpeas, drained',
			{quantity: 400, unit: 'g', name: 'can chickpeas', note: 'drained'}))).toEqual(expected)
		// …or saved unsplit.
		expect(add(line('400 g can chickpeas, drained'))).toEqual(expected)
	})

	it('scales the amount', () => {
		expect(add(line('400 g can chickpeas, drained'), 2)).toMatchObject({quantity: 800, unit: 'g can'})
	})

	it('keeps a can count and its size', () => {
		expect(add(line('2 x 400g cans chickpeas'))).toEqual(
			{name: 'Chickpeas', quantity: 2, unit: '× 400g cans', category_id: cat('Pantry')})
	})

	it('uses the catalogue name for names the household already has', () => {
		expect(add(line('2 brown onions, finely chopped', {quantity: 2, name: 'brown onions', note: 'finely chopped'})))
			.toEqual({name: 'Brown onions', quantity: 2, unit: '', category_id: cat('Fruit & Veg')})
		expect(add(line('3 large eggs', {quantity: 3, name: 'large eggs'})))
			.toMatchObject({name: 'Eggs', category_id: cat('Dairy & Eggs')})
		expect(add(line('2 tbsp extra virgin olive oil', {quantity: 2, unit: 'tbsp', name: 'extra virgin olive oil'})))
			.toMatchObject({name: 'Olive oil', category_id: cat('Pantry')})
		expect(add(line('1 cup grated parmesan', {quantity: 1, unit: 'cup', name: 'grated parmesan'})))
			.toMatchObject({name: 'Parmesan', unit: 'cup'})
	})

	it('keeps a catalogue name that starts with a describing word', () => {
		expect(add(line('500 g fresh pasta', {quantity: 500, unit: 'g', name: 'fresh pasta'})))
			.toMatchObject({name: 'Fresh pasta', category_id: cat('Other')})
	})

	it('picks canned tomatoes over the fresh ones for a can', () => {
		expect(add(line('400 g can diced tomatoes', {quantity: 400, unit: 'g', name: 'can diced tomatoes'})))
			.toMatchObject({name: 'Canned tomatoes', unit: 'g can', category_id: cat('Pantry')})
	})

	it('guesses from similar catalogue entries', () => {
		expect(add(line('1 onion', {quantity: 1, name: 'onion'})))
			.toMatchObject({name: 'Onion', category_id: cat('Fruit & Veg')})
		expect(add(line('1 red capsicum', {quantity: 1, name: 'red capsicum'})).category_id).toBe(cat('Fruit & Veg'))
	})

	it('falls back to keywords when the catalogue disagrees or has nothing', () => {
		// Whole chicken (Meat), Chicken stock (Pantry), Sliced chicken (Deli).
		expect(add(line('500 g chicken thigh fillets', {quantity: 500, unit: 'g', name: 'chicken thigh fillets'})))
			.toMatchObject({name: 'Chicken thigh fillets', category_id: cat('Meat & Seafood')})
		expect(add(line('1 tsp ground cumin', {quantity: 1, unit: 'tsp', name: 'ground cumin'})).category_id)
			.toBe(cat('Pantry'))
		expect(add(line('1 cup frozen peas', {quantity: 1, unit: 'cup', name: 'frozen peas'})).category_id)
			.toBe(cat('Frozen'))
	})

	it('drops notes, brackets and serving words', () => {
		expect(add(line('salt and pepper, to taste', {name: 'salt and pepper'})).name).toBe('Salt and pepper')
		expect(add(line('coriander leaves to serve', {name: 'coriander leaves to serve'})))
			.toMatchObject({name: 'Coriander leaves', quantity: 0, unit: ''})
		expect(add(line('200g feta (crumbled)', {quantity: 200, unit: 'g', name: 'feta (crumbled)'})).name).toBe('Feta')
	})

	it('leaves things it knows nothing about uncategorised', () => {
		expect(add(line('1 widget', {quantity: 1, name: 'widget'}))).toEqual(
			{name: 'Widget', quantity: 1, unit: '', category_id: 0})
	})
})

describe('splitLine', () => {
	it('reads amounts and units like the server does', () => {
		expect(splitLine('2 1/2 cups plain flour')).toEqual({quantity: 2.5, unit: 'cups', name: 'plain flour'})
		expect(splitLine('½ tsp salt')).toEqual({quantity: .5, unit: 'tsp', name: 'salt'})
		expect(splitLine('200g of butter')).toEqual({quantity: 200, unit: 'g', name: 'butter'})
		expect(splitLine('salt')).toEqual({quantity: 0, unit: '', name: 'salt'})
	})
})
