import {describe, expect, it} from 'vitest'
import {readFileSync} from 'node:fs'

// Read from disk: imported, the language files are compiled into message functions.
const read = (path: string) => JSON.parse(readFileSync(new URL(path, import.meta.url), 'utf8')) as Messages
const vikunjaEn = read('../i18n/lang/en.json')
const householdEn = read('./i18n/en.json')

// Vikunja strings that keep the name on purpose.
const KEEP = new Set([
	'misc.poweredBy', // credit to the project Mandarin is built on
	'user.settings.mcp.clients.codex.note', // names the VIKUNJA_MCP_TOKEN setting
])

type Messages = {[key: string]: string | Messages}

function strings(messages: Messages, path = ''): [string, string][] {
	return Object.entries(messages).flatMap(([key, value]) => {
		const at = path ? `${path}.${key}` : key
		return typeof value === 'string' ? [[at, value] as [string, string]] : strings(value, at)
	})
}

describe('rebranding', () => {
	const ours = new Map(strings(householdEn))

	// After an upstream merge, a new or changed English string that mentions
	// Vikunja fails here: add an override for it in household/i18n/en.json.
	it.each(strings(vikunjaEn).filter(([key, value]) => /vikunja/i.test(value) && !KEEP.has(key)))(
		'%s is overridden',
		key => {
			expect(ours.get(key), `add ${key} to household/i18n/en.json`).toBeDefined()
			expect(ours.get(key)).not.toMatch(/vikunja(?! export| instance)/i)
		},
	)
})
