import {describe, expect, it} from 'vitest'

import {savedViewFor, savedViews, withView} from './projectViews'

describe('savedViews', () => {
	it('reads the saved views', () => {
		expect(savedViews({household_project_views: {'5': 12, '7': 3}})).toEqual({'5': 12, '7': 3})
	})
	it('ignores anything odd in the settings JSON', () => {
		expect(savedViews(undefined)).toEqual({})
		expect(savedViews({household_project_views: 'kanban'})).toEqual({})
		expect(savedViews({household_project_views: {'5': 'kanban', '6': -1, '7': 2.5, '8': 4}})).toEqual({'8': 4})
	})
})

describe('savedViewFor', () => {
	it('is 0 for a project without one', () => {
		expect(savedViewFor({'5': 12}, 5)).toBe(12)
		expect(savedViewFor({'5': 12}, 6)).toBe(0)
	})
})

describe('withView', () => {
	it('adds or changes a project\'s view, keeping the others', () => {
		expect(withView({'5': 12}, 7, 3)).toEqual({'5': 12, '7': 3})
		expect(withView({'5': 12}, 5, 13)).toEqual({'5': 13})
	})
	it('is null when nothing would change, so nothing is written', () => {
		expect(withView({'5': 12}, 5, 12)).toBeNull()
		expect(withView({}, 0, 12)).toBeNull()
		expect(withView({}, 5, 0)).toBeNull()
	})
})
