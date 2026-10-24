import {describe, expect, it} from 'vitest'

import {cookingQuery, cookingScale} from './servings'

describe('cookingScale', () => {
	it('scales a recipe that serves 4 to the 6 servings asked for', () => {
		expect(cookingScale(4, {servings: '6'})).toEqual({servings: 6, factor: 1.5})
	})

	it('uses the recipe as written when no servings are asked for', () => {
		expect(cookingScale(4, {})).toEqual({servings: 4, factor: 1})
	})

	it('takes a multiple for a recipe that does not say how many it serves', () => {
		expect(cookingScale(0, {scale: '2'})).toEqual({servings: 0, factor: 2})
		expect(cookingScale(0, {})).toEqual({servings: 0, factor: 1})
	})

	it.each(['abc', '0', '-2', ''])('ignores servings=%j', value => {
		expect(cookingScale(4, {servings: value})).toEqual({servings: 4, factor: 1})
	})

	it('ignores servings when the recipe has none to scale from', () => {
		expect(cookingScale(0, {servings: '3'})).toEqual({servings: 0, factor: 1})
	})
})

describe('cookingQuery', () => {
	it('passes the servings shown on the recipe page', () => {
		expect(cookingQuery(4, 6)).toEqual({servings: '6'})
		expect(cookingScale(4, cookingQuery(4, 6))).toEqual({servings: 6, factor: 1.5})
	})

	it('passes a multiple for a recipe without servings', () => {
		expect(cookingQuery(0, 1.5)).toEqual({scale: '1.5'})
		expect(cookingQuery(0, 1)).toEqual({})
	})
})
