import {describe, expect, it} from 'vitest'

import {normalizeIngredient} from './queries'
import {
	formatAmount,
	formatMinutes,
	ingredientLabel,
	ingredientsToText,
	numberSteps,
	scaledQuantity,
	textToIngredients,
} from './scaling'

const flour = normalizeIngredient({text: '2 cups (250g) plain flour, sifted', quantity: 250, unit: 'g', name: 'plain flour', note: 'sifted'})
const salt = normalizeIngredient({text: '½ tsp salt', quantity: 0.5, unit: 'tsp', name: 'salt'})

describe('formatAmount', () => {
	it.each([
		[1.5, 'tsp', '1½'],
		[0.333, 'cup', '⅓'],
		[2, '', '2'],
		[2.2, 'cups', '2.2'],
		[1234.5, 'g', '1235'],
		[0.75, 'kg', '0.75'],
		[0, '', ''],
	])('%s %s → %s', (quantity, unit, expected) => {
		expect(formatAmount(quantity, unit)).toBe(expected)
	})
})

describe('ingredientLabel', () => {
	it('shows the line as written at the recipe size', () => {
		expect(ingredientLabel(flour, 1)).toBe('2 cups (250g) plain flour, sifted')
	})

	it('rebuilds it from its parts when scaled', () => {
		expect(ingredientLabel(flour, 1.5)).toBe('375 g plain flour, sifted')
		expect(ingredientLabel(salt, 3)).toBe('1½ tsp salt')
	})
})

it('scaledQuantity rounds grams and keeps small amounts', () => {
	expect(scaledQuantity(flour, 1.5)).toBe(375)
	expect(scaledQuantity(salt, 0.5)).toBe(0.25)
})

it('formatMinutes', () => {
	expect(formatMinutes(80)).toBe('1 h 20 min')
	expect(formatMinutes(45)).toBe('45 min')
	expect(formatMinutes(0)).toBe('')
})

describe('ingredient text', () => {
	const lines = [
		{group: '', text: '400 g spaghetti'},
		{group: 'Sauce', text: '2 x 400g cans tomatoes'},
		{group: 'Sauce', text: '1 onion'},
	]
	const text = '400 g spaghetti\n\nSauce:\n2 x 400g cans tomatoes\n1 onion'

	it('round-trips groups as heading lines', () => {
		expect(ingredientsToText(lines)).toBe(text)
		expect(textToIngredients(text)).toEqual(lines)
	})

	it('treats a line with an amount as an ingredient even if it ends in a colon', () => {
		expect(textToIngredients('2 eggs:')).toEqual([{group: '', text: '2 eggs:'}])
	})

	it('allows digits inside a heading', () => {
		expect(textToIngredients('Sauce (Note 2):\n1 tbsp soy')).toEqual([{group: 'Sauce (Note 2)', text: '1 tbsp soy'}])
	})
})

describe('numberSteps', () => {
	it('numbers steps and leaves section labels unnumbered', () => {
		expect(numberSteps(['Prep:', 'Chop.', 'Cook:', 'Fry.', 'Serve.'])).toEqual([
			{text: 'Prep', number: 0},
			{text: 'Chop.', number: 1},
			{text: 'Cook', number: 0},
			{text: 'Fry.', number: 2},
			{text: 'Serve.', number: 3},
		])
	})
})
