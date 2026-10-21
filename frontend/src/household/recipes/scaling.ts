// Scaling a recipe's ingredients and showing amounts the way a cook reads
// them: "1½ tsp", "375 g".

import type {IngredientLine} from './queries'

// Weights and volumes measured on scales and jugs: decimals, rounded sensibly.
const METRIC = new Set(['g', 'kg', 'mg', 'ml', 'l'])

const FRACTIONS: [number, string][] = [[1 / 4, '¼'], [1 / 3, '⅓'], [1 / 2, '½'], [2 / 3, '⅔'], [3 / 4, '¾']]

/** "1½" for 1.5, "⅓" for 0.33, "2.2" when no fraction is close. */
export function formatAmount(quantity: number, unit = ''): string {
	if (quantity <= 0) return ''
	if (METRIC.has(unit.toLowerCase())) {
		// 1234.5 g → 1235 g; 0.75 kg stays 0.75 kg.
		const rounded = quantity >= 10 ? Math.round(quantity) : Number(quantity.toFixed(2))
		return String(rounded)
	}
	const whole = Math.floor(quantity)
	const rest = quantity - whole
	if (rest < 0.03) return String(whole)
	if (rest > 0.97) return String(whole + 1)
	const near = FRACTIONS.find(([value]) => Math.abs(rest - value) < 0.03)
	if (near) return whole > 0 ? `${whole}${near[1]}` : near[1]
	return String(Number(quantity.toFixed(2)))
}

/** The line to show: as written at the recipe's own size, rebuilt from its parts when scaled. */
export function ingredientLabel(line: IngredientLine, factor: number): string {
	if (factor === 1 || line.quantity === 0) return line.text
	const amount = [formatAmount(line.quantity * factor, line.unit), line.unit].filter(Boolean).join(' ')
	return [amount, line.name].filter(Boolean).join(' ') + (line.note ? `, ${line.note}` : '')
}

/** A scaled amount to shop for: whole grams, tidy fractions otherwise. */
export function scaledQuantity(line: IngredientLine, factor: number): number {
	const quantity = line.quantity * factor
	if (METRIC.has(line.unit.toLowerCase()) && quantity >= 10) return Math.round(quantity)
	return Number(quantity.toFixed(2))
}

/** "1 h 20 min", "45 min", "" for none. */
export function formatMinutes(minutes: number): string {
	if (minutes <= 0) return ''
	const hours = Math.floor(minutes / 60)
	const rest = minutes % 60
	return [hours ? `${hours} h` : '', rest ? `${rest} min` : ''].filter(Boolean).join(' ')
}

// ---- Editing ingredients as text ----
// The editor shows ingredients as one line each, with a heading line ending
// in ":" (e.g. "Sauce:") starting a group.

export function ingredientsToText(lines: Pick<IngredientLine, 'group' | 'text'>[]): string {
	const out: string[] = []
	let group = ''
	for (const line of lines) {
		if (line.group !== group) {
			if (out.length > 0) out.push('')
			if (line.group) out.push(`${line.group}:`)
			group = line.group
		}
		out.push(line.text)
	}
	return out.join('\n')
}

/**
 * A heading line: ends in ":" and doesn't start with an amount. "Sauce (Note
 * 2):" is a heading; "2 eggs:" is an ingredient with a stray colon. Also used
 * for section labels in the method ("Stir fry:").
 */
export function isHeading(line: string): boolean {
	return line.endsWith(':') && !/^[\d½⅓⅔¼¾⅛]/.test(line)
}

export interface MethodEntry {
	text: string
	number: number // 0 for a heading
}

/** Numbers the steps, skipping section labels. */
export function numberSteps(steps: string[]): MethodEntry[] {
	let n = 0
	return steps.map(text => isHeading(text) ? {text: text.slice(0, -1), number: 0} : {text, number: ++n})
}

export function textToIngredients(text: string): Pick<IngredientLine, 'group' | 'text'>[] {
	const lines: Pick<IngredientLine, 'group' | 'text'>[] = []
	let group = ''
	for (const raw of text.split('\n')) {
		const line = raw.trim()
		if (line === '') continue
		if (isHeading(line)) {
			group = line.slice(0, -1).trim()
			continue
		}
		lines.push({group, text: line})
	}
	return lines
}
