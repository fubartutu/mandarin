import {describe, expect, it} from 'vitest'

import {
	clearCooking,
	deviceStorage,
	fitToRecipe,
	forgetStaleCooking,
	loadCooking,
	saveCooking,
	type CookingState,
} from './resume'

// A Map behind the Storage interface: each test gets its own device.
function memoryStorage(): Storage {
	const items = new Map<string, string>()
	return {
		get length() { return items.size },
		key: (i: number) => [...items.keys()][i] ?? null,
		getItem: (key: string) => items.get(key) ?? null,
		setItem: (key: string, value: string) => { items.set(key, String(value)) },
		removeItem: (key: string) => { items.delete(key) },
		clear: () => items.clear(),
	}
}

const AT = new Date('2026-10-24T18:00:00Z').getTime()
const MINUTE = 60_000
const HOUR = 60 * MINUTE

const timer = {id: 't1', stepNumber: 2, label: '20 minutes', seconds: 1200, endsAt: AT + 20 * MINUTE}
const rung = {id: 't0', stepNumber: 1, label: '5 min', seconds: 300, endsAt: AT - MINUTE}

const cooking: CookingState = {stepIndex: 2, ticked: [0, 3], timers: [timer], finished: [rung]}

describe('cooking mode resume', () => {
	it('gives back the step, ticks and timers saved for that recipe', () => {
		const storage = memoryStorage()
		saveCooking(storage, 7, cooking, AT)

		expect(loadCooking(storage, 7, AT + 2 * HOUR)).toEqual(cooking)
	})

	it('keeps each recipe to itself', () => {
		const storage = memoryStorage()
		saveCooking(storage, 7, cooking, AT)

		expect(loadCooking(storage, 8, AT)).toBeNull()
	})

	it('starts fresh after 6 hours and forgets the old state', () => {
		const storage = memoryStorage()
		saveCooking(storage, 7, cooking, AT)

		expect(loadCooking(storage, 7, AT + 6 * HOUR - 1)).toEqual(cooking)
		expect(loadCooking(storage, 7, AT + 6 * HOUR)).toBeNull()
		expect(storage.length).toBe(0)
	})

	it.each([
		['not JSON', '{oops'],
		['another version', JSON.stringify({...cooking, version: 2, savedAt: AT})],
		['no time saved', JSON.stringify({...cooking, version: 1})],
		['a broken step', JSON.stringify({...cooking, version: 1, savedAt: AT, stepIndex: 'two'})],
		['broken ticks', JSON.stringify({...cooking, version: 1, savedAt: AT, ticked: [1, 'x']})],
		['a broken timer', JSON.stringify({...cooking, version: 1, savedAt: AT, timers: [{id: 't1'}]})],
		['no finished list', JSON.stringify({...cooking, version: 1, savedAt: AT, finished: null})],
		['plain null', 'null'],
	])('drops a saved state that is %s', (_, raw) => {
		const storage = memoryStorage()
		storage.setItem('mandarin-cooking-7', raw)

		expect(loadCooking(storage, 7, AT)).toBeNull()
		expect(storage.length).toBe(0)
	})

	it('fits a saved state to the recipe as it is now', () => {
		const saved = {...cooking, stepIndex: 5, ticked: [0, 3, 4, 9]}

		expect(fitToRecipe(saved, {
			steps: 3,
			ingredients: 4,
		})).toEqual({
			...cooking,
			stepIndex: 2,
			ticked: [0, 3],
		})
		expect(fitToRecipe({
			...cooking,
			stepIndex: -1,
		}, {
			steps: 0,
			ingredients: 0,
		})).toEqual({
			...cooking,
			stepIndex: 0,
			ticked: [],
		})
	})

	it('forgets a recipe once it has been cooked', () => {
		const storage = memoryStorage()
		saveCooking(storage, 7, cooking, AT)
		saveCooking(storage, 8, cooking, AT)

		clearCooking(storage, 7)

		expect(loadCooking(storage, 7, AT)).toBeNull()
		expect(loadCooking(storage, 8, AT)).toEqual(cooking)
	})

	it('tidies away other recipes that have gone stale, and nothing else', () => {
		const storage = memoryStorage()
		saveCooking(storage, 1, cooking, AT - 7 * HOUR)
		saveCooking(storage, 2, cooking, AT - 5 * HOUR)
		saveCooking(storage, 3, cooking, AT - 9 * HOUR)
		storage.setItem('mandarin-cooking-4', 'garbage')
		storage.setItem('mandarin.recipes.recent', '[1,2]')

		forgetStaleCooking(storage, AT)

		expect([...Array(storage.length).keys()].map(i => storage.key(i)).sort())
			.toEqual(['mandarin-cooking-2', 'mandarin.recipes.recent'])
	})
})

describe('cooking mode resume without storage', () => {
	// Private browsing, a full quota or blocked site data: every call throws.
	const broken: Storage = {
		get length(): number { throw new Error('SecurityError') },
		key() { throw new Error('SecurityError') },
		getItem() { throw new Error('SecurityError') },
		setItem() { throw new Error('QuotaExceededError') },
		removeItem() { throw new Error('SecurityError') },
		clear() { throw new Error('SecurityError') },
	}

	it('carries on as if nothing was ever saved', () => {
		expect(() => saveCooking(broken, 7, cooking, AT)).not.toThrow()
		expect(loadCooking(broken, 7, AT)).toBeNull()
		expect(() => clearCooking(broken, 7)).not.toThrow()
		expect(() => forgetStaleCooking(broken, AT)).not.toThrow()
	})

	it('also copes with no storage at all', () => {
		expect(() => saveCooking(null, 7, cooking, AT)).not.toThrow()
		expect(loadCooking(null, 7, AT)).toBeNull()
		expect(() => clearCooking(null, 7)).not.toThrow()
		expect(() => forgetStaleCooking(null, AT)).not.toThrow()
	})

	it('finds no storage when the browser refuses access to it', () => {
		const original = Object.getOwnPropertyDescriptor(window, 'localStorage')!
		Object.defineProperty(window, 'localStorage', {configurable: true, get() { throw new Error('SecurityError') }})
		try {
			expect(deviceStorage()).toBeNull()
		} finally {
			Object.defineProperty(window, 'localStorage', original)
		}
		expect(deviceStorage()).toBe(window.localStorage)
	})
})
