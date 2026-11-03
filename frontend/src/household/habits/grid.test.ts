import {describe, expect, it} from 'vitest'

import {gridColumns, gridRange, mondayOf} from './grid'

describe('mondayOf', () => {
	it('finds the Monday of the week, Sunday included', () => {
		expect(mondayOf('2026-11-04')).toBe('2026-11-02') // Wednesday
		expect(mondayOf('2026-11-02')).toBe('2026-11-02')
		expect(mondayOf('2026-11-08')).toBe('2026-11-02') // Sunday
	})
})

describe('gridRange', () => {
	it('starts on a Monday, whole weeks back, and ends today', () => {
		expect(gridRange('daily', '2026-11-04')).toEqual({from: '2026-08-17', to: '2026-11-04'})
		expect(gridRange('weekly', '2026-11-04').from).toBe('2026-05-11')
	})
})

describe('gridColumns', () => {
	it('places each day in its week and leaves the days after today empty', () => {
		const periods = [
			{start: '2026-10-26', status: 'done' as const, count: 1},
			{start: '2026-11-04', status: 'pending' as const, count: 0},
		]
		const columns = gridColumns(periods, '2026-10-26', 2)
		expect(columns).toHaveLength(2)
		expect(columns[0][0]?.status).toBe('done')
		expect(columns[0][1]).toBeNull()
		expect(columns[1][2]?.status).toBe('pending') // Wednesday
		expect(columns[1][6]).toBeNull()
	})
})
