import {beforeEach, describe, expect, it, vi} from 'vitest'

import {keepOfflineApiCacheFor} from './offlineTasks'

describe('tasks kept for offline reading', () => {
	const deleted: string[] = []

	beforeEach(() => {
		deleted.length = 0
		localStorage.clear()
		vi.stubGlobal('caches', {delete: async (name: string) => {
			deleted.push(name)
			return true
		}})
	})

	it('are deleted when a different user signs in, and kept for the same one', async () => {
		await keepOfflineApiCacheFor(1)
		await keepOfflineApiCacheFor(1)
		expect(deleted).toEqual(['mandarin-api']) // the first time: nothing known about whose they were
		await keepOfflineApiCacheFor(2)
		expect(deleted).toEqual(['mandarin-api', 'mandarin-api'])
	})

	it('ignore a missing user', async () => {
		await keepOfflineApiCacheFor(0)
		expect(deleted).toEqual([])
	})
})
