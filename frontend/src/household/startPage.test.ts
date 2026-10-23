import {describe, expect, it} from 'vitest'
import {START_LOCATION, type RouteLocationNormalized} from 'vue-router'

import {isAppOpening, startPageFrom} from './startPage'

const route = (name: string) => ({...START_LOCATION, name}) as RouteLocationNormalized

describe('startPageFrom', () => {
	it('keeps a known page', () => {
		expect(startPageFrom('shopping')).toBe('shopping')
		expect(startPageFrom('recipes')).toBe('recipes')
	})

	it('falls back to Home for unset or unknown values', () => {
		expect(startPageFrom(undefined)).toBe('home')
		expect(startPageFrom('kanban')).toBe('home')
	})
})

describe('isAppOpening', () => {
	it('is true when the app starts or someone signs in', () => {
		expect(isAppOpening(START_LOCATION)).toBe(true)
		expect(isAppOpening(route('user.login'))).toBe(true)
	})

	it('is false for clicks inside the app', () => {
		expect(isAppOpening(route('shopping'))).toBe(false)
		expect(isAppOpening(route('tasks.range'))).toBe(false)
	})
})
