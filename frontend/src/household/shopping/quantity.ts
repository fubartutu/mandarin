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
 *   "3 apples"      → {name: "apples", quantity: 3, unit: ""}
 *   "7up"           → {name: "7up", quantity: 0, unit: ""}
 */
export function parseItemInput(text: string): ParsedItem {
	const trimmed = text.trim()
	const match = LEADING.exec(trimmed)
	if (!match) {
		return {name: trimmed, quantity: 0, unit: ''}
	}

	// Destructuring: skip the whole match, take the three groups.
	const [, rawNumber, word = '', rest] = match
	const unit = UNITS[word.toLowerCase()]

	if (unit !== undefined) {
		// "500g mince": the word was a unit. Without a name it's not an item.
		return rest
			? {name: rest, quantity: toNumber(rawNumber), unit}
			: {name: trimmed, quantity: 0, unit: ''}
	}

	// A number glued to a non-unit word ("7up", "4x4") is part of the name.
	const glued = word !== '' && !/\s/.test(trimmed.slice(match[1].length, match[1].length + 1))
	const name = [word, rest].filter(Boolean).join(' ')
	if (glued || name === '') {
		return {name: trimmed, quantity: 0, unit: ''}
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
