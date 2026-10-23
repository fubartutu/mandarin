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
		['2 litres of milk', 'milk', 2, 'L'],
		['500 grams of Mince', 'Mince', 500, 'g'],
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
		['flour 1kg', 'flour', 1, 'kg'],
		['milk 2 L', 'milk', 2, 'L'],
		['Coke 1.25 litres', 'Coke', 1.25, 'L'],
		['eggs 1/2 dozen', 'eggs', 0.5, 'dozen'],
		['Tim Tams 3 packs', 'Tim Tams', 3, 'packs'],
		['7up 2L', '7up', 2, 'L'],
		['apples 3', 'apples 3', 0, ''],
		['size 10 shoes', 'size 10 shoes', 0, ''],
		['flour1kg', 'flour1kg', 0, ''],
		// Said rather than typed, as dictation writes it.
		['two eggs', 'eggs', 2, ''],
		['Two eggs', 'eggs', 2, ''],
		['three packs of Tim Tams', 'Tim Tams', 3, 'packs'],
		['twenty five g mince', 'mince', 25, 'g'],
		['twenty-five g mince', 'mince', 25, 'g'],
		['twenty apples', 'apples', 20, ''],
		['a dozen eggs', 'eggs', 1, 'dozen'],
		['an L of milk', 'milk', 1, 'L'],
		['half a kilo of mince', 'mince', 0.5, 'kg'],
		['milk two litres', 'milk', 2, 'L'],
		['flour a kilo', 'flour', 1, 'kg'],
		['Coke twenty five cans', 'Coke', 25, 'cans'],
		['a cake', 'a cake', 0, ''],
		['two', 'two', 0, ''],
		['seven-up', 'seven-up', 0, ''],
		['shoes size two', 'shoes size two', 0, ''],
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
