import {describe, expect, it} from 'vitest'

import {matchRecipes} from './recipeMatches'

const recipes = [
	{id: 1, title: 'Lasagne'},
	{id: 2, title: 'Chicken curry'},
	{id: 3, title: 'Green salad'},
	{id: 4, title: 'Thai green curry'},
]

describe('matchRecipes', () => {
	it('finds titles containing the text, ignoring case', () => {
		expect(matchRecipes(recipes, 'CURRY').map(r => r.id)).toEqual([2, 4])
		expect(matchRecipes(recipes, 'green').map(r => r.id)).toEqual([3, 4])
	})

	it('ignores spaces around the text', () => {
		expect(matchRecipes(recipes, '  lasa ').map(r => r.id)).toEqual([1])
	})

	it('finds nothing for empty text or no match', () => {
		expect(matchRecipes(recipes, '')).toEqual([])
		expect(matchRecipes(recipes, '   ')).toEqual([])
		expect(matchRecipes(recipes, 'pizza')).toEqual([])
	})

	it('shows at most six', () => {
		const many = Array.from({length: 10}, (_, i) => ({id: i + 1, title: `Soup ${i + 1}`}))
		expect(matchRecipes(many, 'soup').map(r => r.id)).toEqual([1, 2, 3, 4, 5, 6])
	})
})
