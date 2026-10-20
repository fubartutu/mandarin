// Autocomplete over the household catalogue, done in the browser: the whole
// catalogue is cached, so suggestions are instant (and will work offline).

import type {CatalogEntry} from './queries'

// Lower tier = better match.
function matchTier(name: string, query: string): number | null {
	if (name.startsWith(query)) return 0
	if (name.split(/[\s-]+/).some(word => word.startsWith(query))) return 1
	if (name.includes(query)) return 2
	return null
}

/**
 * The best catalogue matches for what's typed, best first:
 * name starts with it, then a word starts with it, then contains it;
 * within each, most often added, then most recently added, then A–Z.
 * Names in `exclude` (already on the list) are left out.
 */
export function suggestCatalogEntries(
	entries: CatalogEntry[],
	typed: string,
	exclude: Iterable<string> = [],
	limit = 6,
): CatalogEntry[] {
	const query = typed.trim().toLowerCase()
	if (query === '') {
		return []
	}
	const skip = new Set([...exclude].map(name => name.trim().toLowerCase()))

	return entries
		.flatMap(entry => {
			const name = entry.name.toLowerCase()
			const tier = skip.has(name) ? null : matchTier(name, query)
			// flatMap with [] drops non-matches in the same pass.
			return tier === null ? [] : [{entry, tier}]
		})
		.sort((a, b) =>
			a.tier - b.tier ||
			b.entry.times_added - a.entry.times_added ||
			(b.entry.last_added ?? '').localeCompare(a.entry.last_added ?? '') ||
			a.entry.name.localeCompare(b.entry.name))
		.slice(0, limit)
		.map(({entry}) => entry)
}

/** The catalogue entry with exactly this name (ignoring case), if any. */
export function findCatalogEntry(entries: CatalogEntry[], name: string): CatalogEntry | undefined {
	const key = name.trim().toLowerCase()
	return entries.find(entry => entry.name.toLowerCase() === key)
}
