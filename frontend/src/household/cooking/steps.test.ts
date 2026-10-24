import {describe, expect, it} from 'vitest'

import {cookingSteps, moveStep} from './steps'

describe('cookingSteps', () => {
	it('numbers the steps and carries section labels onto the steps under them', () => {
		expect(cookingSteps(['Prep:', 'Chop the onion', 'Stir fry:', 'Fry the onion', 'Serve'])).toEqual([
			{text: 'Chop the onion', number: 1, section: 'Prep'},
			{text: 'Fry the onion', number: 2, section: 'Stir fry'},
			{text: 'Serve', number: 3, section: 'Stir fry'},
		])
	})

	it('leaves the section empty before any label', () => {
		expect(cookingSteps(['Boil water'])).toEqual([{text: 'Boil water', number: 1, section: ''}])
	})

	it('is empty for a recipe without a method', () => {
		expect(cookingSteps([])).toEqual([])
	})
})

describe('moveStep', () => {
	it('moves forward and back', () => {
		expect(moveStep(0, 1, 3)).toBe(1)
		expect(moveStep(2, -1, 3)).toBe(1)
	})

	it('stays on the first and last step', () => {
		expect(moveStep(0, -1, 3)).toBe(0)
		expect(moveStep(2, 1, 3)).toBe(2)
	})

	it('pulls an out-of-range step back into the recipe', () => {
		expect(moveStep(7, 0, 3)).toBe(2)
		expect(moveStep(0, 1, 0)).toBe(0)
	})
})
