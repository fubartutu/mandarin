import {describe, expect, it} from 'vitest'

import {changelog, entryDate, unseenEntries} from './changelog'

describe('changelog', () => {
	it('is newest first, one entry per day', () => {
		const dates = changelog.map(entry => entry.date)
		expect(dates).toEqual([...new Set(dates)].sort().reverse())
	})

	it('has real dates and something in every entry', () => {
		for (const entry of changelog) {
			expect(entry.date).toMatch(/^\d{4}-\d{2}-\d{2}$/)
			expect(entryDate(entry.date).getDate()).toBe(Number(entry.date.slice(8)))
			expect(entry.changes.length).toBeGreaterThan(0)
		}
	})
})

describe('unseenEntries', () => {
	const entries = [
		{date: '2026-10-27', changes: []},
		{date: '2026-10-24', changes: []},
	]

	it('lists everything when never seen', () => {
		expect(unseenEntries(entries, '')).toHaveLength(2)
	})

	it('lists only newer entries', () => {
		expect(unseenEntries(entries, '2026-10-24').map(e => e.date)).toEqual(['2026-10-27'])
		expect(unseenEntries(entries, '2026-10-27')).toEqual([])
	})
})
