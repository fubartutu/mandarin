import {describe, expect, it} from 'vitest'

import {findCatalogEntry, suggestCatalogEntries} from './autocomplete'
import {normalizeCatalogEntry} from './queries'

const entry = (id: number, name: string, extra: Record<string, unknown> = {}) =>
	normalizeCatalogEntry({id, name, ...extra})

const catalog = [
	entry(1, 'Milk', {times_added: 3, last_added: '2026-10-19T10:00:00Z'}),
	entry(2, 'Oat milk', {times_added: 1, last_added: '2026-10-20T10:00:00Z'}),
	entry(3, 'Coconut milk'),
	entry(4, 'Mince', {times_added: 5, last_added: '2026-10-18T10:00:00Z'}),
	entry(5, 'Salami'),
	entry(6, 'Mineral water'),
	entry(7, 'Band-aids'),
]

const names = (entries: ReturnType<typeof suggestCatalogEntries>) => entries.map(e => e.name)

describe('suggestCatalogEntries', () => {
	it('puts names starting with the text first, frequent ones first', () => {
		expect(names(suggestCatalogEntries(catalog, 'mi'))).toEqual([
			'Mince', 'Milk', 'Mineral water', // start with "mi": 5 uses, 3 uses, never
			'Oat milk', 'Coconut milk', // a word starts with "mi": recent use first
			'Salami', // contains "mi"
		])
	})

	it('ignores case and surrounding spaces', () => {
		expect(names(suggestCatalogEntries(catalog, '  MILK '))).toEqual(['Milk', 'Oat milk', 'Coconut milk'])
	})

	it('matches words after a hyphen', () => {
		expect(names(suggestCatalogEntries(catalog, 'aids'))).toEqual(['Band-aids'])
	})

	it('leaves out items already on the list', () => {
		expect(names(suggestCatalogEntries(catalog, 'milk', ['milk ', 'OAT MILK']))).toEqual(['Coconut milk'])
	})

	it('returns nothing for empty text and respects the limit', () => {
		expect(suggestCatalogEntries(catalog, '   ')).toEqual([])
		expect(suggestCatalogEntries(catalog, 'm', [], 2)).toHaveLength(2)
	})
})

describe('findCatalogEntry', () => {
	it('finds by exact name, ignoring case', () => {
		expect(findCatalogEntry(catalog, ' oat MILK')?.id).toBe(2)
		expect(findCatalogEntry(catalog, 'oat')).toBeUndefined()
	})
})
