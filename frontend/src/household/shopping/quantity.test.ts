import {describe, expect, it} from 'vitest'

import {formatQuantity, parseItemInput} from './quantity'

describe('parseItemInput', () => {
	// it.each runs the same test for every row of the table.
	it.each([
		['milk', 'milk', 0, ''],
		['  milk  ', 'milk', 0, ''],
		['2 L milk', 'milk', 2, 'L'],
		['2l milk', 'milk', 2, 'L'],
		['500g mince', 'mince', 500, 'g'],
		['500 grams mince', 'mince', 500, 'g'],
		['1.5 kg potatoes', 'potatoes', 1.5, 'kg'],
		['1,5 kg potatoes', 'potatoes', 1.5, 'kg'],
		['1/2 dozen eggs', 'eggs', 0.5, 'dozen'],
		['3 apples', 'apples', 3, ''],
		['3 packs Tim Tams', 'Tim Tams', 3, 'packs'],
		['2 pk chips', 'chips', 2, 'packs'],
		['7up', '7up', 0, ''],
		['4x4 tyres', '4x4 tyres', 0, ''],
		['2 L', '2 L', 0, ''],
		['12', '12', 0, ''],
		['school shoes size 3', 'school shoes size 3', 0, ''],
	])('%j → %j, %d %j', (input, name, quantity, unit) => {
		expect(parseItemInput(input)).toEqual({name, quantity, unit})
	})
})

describe('formatQuantity', () => {
	it.each([
		[0, '', ''],
		[2, '', '2'],
		[500, 'g', '500 g'],
		[1.5, 'kg', '1.5 kg'],
		[1 / 3, 'L', '0.33 L'],
		[0, 'packs', 'packs'],
		[undefined, undefined, ''],
	])('%j %j → %j', (quantity, unit, expected) => {
		expect(formatQuantity(quantity, unit)).toBe(expected)
	})
})
