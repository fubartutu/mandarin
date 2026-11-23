import {describe, expect, it} from 'vitest'

import {moveItem} from './reorder'

const items = (...positions: number[]) => positions.map((position, i) => ({id: i + 1, position}))
const moves = (r: ReturnType<typeof moveItem>) => r.map(m => [m.item.id, m.position])

describe('moveItem', () => {
	it('swaps neighbours, writing only those two', () => {
		expect(moves(moveItem(items(0, 1, 2), 2, -1))).toEqual([[3, 1], [2, 2]])
		expect(moves(moveItem(items(0, 1, 2), 0, 1))).toEqual([[2, 0], [1, 1]])
	})

	it('does nothing past either end', () => {
		expect(moveItem(items(0, 1), 0, -1)).toEqual([])
		expect(moveItem(items(0, 1), 1, 1)).toEqual([])
	})

	it('renumbers equal or gappy positions', () => {
		// All 0 (habits made before positions mattered): every one after the first moves.
		expect(moves(moveItem(items(0, 0, 0), 2, -1))).toEqual([[3, 1], [2, 2]])
		expect(moves(moveItem(items(3, 7, 9), 0, 1))).toEqual([[2, 0], [1, 1], [3, 2]])
	})
})
