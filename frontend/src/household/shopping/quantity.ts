// Parsing and formatting of "2 L milk"-style shopping entries.

export interface ParsedItem {
	name: string
	quantity: number // 0 = no quantity
	unit: string
}

// Words we treat as a unit when they follow a number. Anything else after a
// number is part of the name: "2 apples" is quantity 2 of "apples".
// The map value is how the unit is stored and shown.
const UNITS: Record<string, string> = {
	mg: 'mg', g: 'g', gram: 'g', grams: 'g', kg: 'kg', kilo: 'kg', kilos: 'kg',
	ml: 'mL', l: 'L', litre: 'L', litres: 'L', liter: 'L', liters: 'L',
	pack: 'packs', packs: 'packs', pk: 'packs', packet: 'packs', packets: 'packs',
	can: 'cans', cans: 'cans', tin: 'tins', tins: 'tins',
	bottle: 'bottles', bottles: 'bottles', box: 'boxes', boxes: 'boxes',
	bag: 'bags', bags: 'bags', jar: 'jars', jars: 'jars',
	bunch: 'bunches', bunches: 'bunches', dozen: 'dozen', doz: 'dozen',
	loaf: 'loaves', loaves: 'loaves', pair: 'pairs', pairs: 'pairs',
	roll: 'rolls', rolls: 'rolls', tray: 'trays', trays: 'trays',
}

// A leading number: 2, 1.5, 1,5 (comma decimals) or a fraction like 1/2.
// Regex parts: ^\s* skip spaces, (\d+(?:[.,]\d+)?|\d+\/\d+) the number,
// \s*([a-zA-Z]+)? an optional word stuck to it or after a space, rest = name.
const LEADING = /^\s*(\d+\/\d+|\d+(?:[.,]\d+)?)\s*([a-zA-Z]+)?\s*(.*)$/

// A trailing amount: "flour 1kg", "milk 2 L". (.+?) is the name (the ? makes
// it match as little as possible), \s+ a space before the number.
const TRAILING = /^(.+?)\s+(\d+\/\d+|\d+(?:[.,]\d+)?)\s*([a-zA-Z]+)$/

// Amounts said rather than typed, as dictation writes them: "two eggs",
// "twenty-five g mince". Up to ninety-nine is plenty for a shopping list.
const SMALL_NUMBERS: Record<string, number> = {
	one: 1, two: 2, three: 3, four: 4, five: 5, six: 6, seven: 7, eight: 8, nine: 9, ten: 10,
	eleven: 11, twelve: 12, thirteen: 13, fourteen: 14, fifteen: 15,
	sixteen: 16, seventeen: 17, eighteen: 18, nineteen: 19,
}
const TENS: Record<string, number> = {
	twenty: 20, thirty: 30, forty: 40, fifty: 50, sixty: 60, seventy: 70, eighty: 80, ninety: 90,
}

/**
 * The number spelt by words[start…], and how many words it used: "two" → 2
 * (1 word), "twenty five" or "twenty-five" → 25 (1 or 2 words). Null if
 * words[start] isn't a number word.
 */
function readNumberWords(words: string[], start: number): {value: number, used: number} | null {
	const first = words[start]?.toLowerCase() ?? ''
	const [tens, ones] = first.split('-')
	if (ones !== undefined) {
		// "twenty-five" in one word
		return TENS[tens] !== undefined && SMALL_NUMBERS[ones] !== undefined && SMALL_NUMBERS[ones] < 10
			? {value: TENS[tens] + SMALL_NUMBERS[ones], used: 1}
			: null
	}
	if (SMALL_NUMBERS[first] !== undefined) {
		return {value: SMALL_NUMBERS[first], used: 1}
	}
	if (TENS[first] !== undefined) {
		const next = SMALL_NUMBERS[words[start + 1]?.toLowerCase() ?? '']
		return next !== undefined && next < 10
			? {value: TENS[first] + next, used: 2}
			: {value: TENS[first], used: 1}
	}
	return null
}

const isUnit = (word: string | undefined) => word !== undefined && UNITS[word.toLowerCase()] !== undefined

/**
 * Turns a spoken amount into digits, so the parsers below can read it:
 * "two eggs" → "2 eggs", "a dozen eggs" → "1 dozen eggs", "half a kilo of
 * mince" → "0.5 kilo of mince", "milk two litres" → "milk 2 litres".
 * "a" and "half" count only before a unit: "a cake" stays as it is.
 */
export function spokenAmountToDigits(text: string): string {
	const words = text.trim().split(/\s+/)
	const lower = words.map(w => w.toLowerCase())

	// At the start.
	if ((lower[0] === 'a' || lower[0] === 'an') && isUnit(words[1])) {
		return ['1', ...words.slice(1)].join(' ')
	}
	if (lower[0] === 'half' && (lower[1] === 'a' || lower[1] === 'an') && isUnit(words[2])) {
		return ['0.5', ...words.slice(2)].join(' ')
	}
	const leading = readNumberWords(words, 0)
	if (leading !== null && words.length > leading.used) {
		return [String(leading.value), ...words.slice(leading.used)].join(' ')
	}

	// At the end, before a unit: "milk two litres", "flour a kilo".
	const last = words.length - 1
	if (last >= 2 && isUnit(words[last])) {
		if (lower[last - 1] === 'a' || lower[last - 1] === 'an') {
			return [...words.slice(0, last - 1), '1', words[last]].join(' ')
		}
		// Try a two-word number ("twenty five") first, then one word.
		for (const used of [2, 1]) {
			const start = last - used
			const trailing = start >= 1 ? readNumberWords(words, start) : null
			if (trailing !== null && trailing.used === used) {
				return [...words.slice(0, start), String(trailing.value), words[last]].join(' ')
			}
		}
	}
	return text
}

function toNumber(raw: string): number {
	if (raw.includes('/')) {
		const [top, bottom] = raw.split('/').map(Number)
		return bottom ? top / bottom : 0
	}
	return Number(raw.replace(',', '.'))
}

/**
 * Splits what someone typed into name, quantity and unit.
 *
 *   "milk"          → {name: "milk", quantity: 0, unit: ""}
 *   "2 L milk"      → {name: "milk", quantity: 2, unit: "L"}
 *   "500g mince"    → {name: "mince", quantity: 500, unit: "g"}
 *   "2 litres of milk" → {name: "milk", quantity: 2, unit: "L"}
 *   "3 apples"      → {name: "apples", quantity: 3, unit: ""}
 *   "flour 1kg"     → {name: "flour", quantity: 1, unit: "kg"}
 *   "7up"           → {name: "7up", quantity: 0, unit: ""}
 *   "two eggs"      → {name: "eggs", quantity: 2, unit: ""}
 *
 * An amount at the end needs a unit: in "school shoes size 3" the 3 is
 * part of the name.
 */
export function parseItemInput(text: string): ParsedItem {
	const trimmed = spokenAmountToDigits(text).trim()
	// ?? uses the right side only when the left is null or undefined.
	return parseLeading(trimmed) ?? parseTrailing(trimmed) ?? {name: trimmed, quantity: 0, unit: ''}
}

/** "flour 1kg"-style input, or null if it doesn't end with an amount and unit. */
function parseTrailing(trimmed: string): ParsedItem | null {
	const match = TRAILING.exec(trimmed)
	if (!match) {
		return null
	}
	const [, name, rawNumber, word] = match
	const unit = UNITS[word.toLowerCase()]
	return unit === undefined ? null : {name, quantity: toNumber(rawNumber), unit}
}

/** "2 L milk"-style input, or null if it doesn't start with an amount. */
function parseLeading(trimmed: string): ParsedItem | null {
	const match = LEADING.exec(trimmed)
	if (!match) {
		return null
	}

	// Destructuring: skip the whole match, take the three groups.
	const [, rawNumber, word = '', rest] = match
	const unit = UNITS[word.toLowerCase()]

	if (unit !== undefined) {
		// "500g mince": the word was a unit. Without a name it's not an item.
		// "2 litres of milk" (how it's said, so how dictation writes it) drops the "of".
		const name = rest.replace(/^of\s+/i, '')
		return name
			? {name, quantity: toNumber(rawNumber), unit}
			: null
	}

	// A number glued to a non-unit word ("7up", "4x4") is part of the name.
	const glued = word !== '' && !/\s/.test(trimmed.slice(match[1].length, match[1].length + 1))
	const name = [word, rest].filter(Boolean).join(' ')
	if (glued || name === '') {
		return null
	}
	return {name, quantity: toNumber(rawNumber), unit: ''}
}

/** "500 g", "2", "1.5 kg", "" for no quantity. */
export function formatQuantity(quantity: number | undefined, unit: string | undefined): string {
	if (!quantity) {
		return unit ?? ''
	}
	// Up to two decimals, without trailing zeros: 1.50 → 1.5
	const amount = Number(quantity.toFixed(2)).toString()
	return unit ? `${amount} ${unit}` : amount
}
