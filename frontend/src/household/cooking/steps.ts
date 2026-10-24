// The method as cooking mode pages through it: one step at a time, each
// carrying the section label ("Sauce") it sits under.

import {numberSteps} from '../recipes/scaling'

export interface CookingStep {
	text: string
	number: number // 1-based, as on the recipe page
	section: string // '' before any section label
}

export function cookingSteps(steps: string[]): CookingStep[] {
	let section = ''
	const out: CookingStep[] = []
	for (const entry of numberSteps(steps)) {
		if (entry.number === 0) section = entry.text
		else out.push({text: entry.text, number: entry.number, section})
	}
	return out
}

/** The step index after moving by `delta`, kept within the `count` steps. */
export function moveStep(current: number, delta: number, count: number): number {
	if (count <= 0) return 0
	return Math.min(Math.max(current + delta, 0), count - 1)
}
