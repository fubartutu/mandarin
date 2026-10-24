// How much cooking mode makes, carried in the address:
//   /recipes/12/cook?servings=6  for a recipe that says how many it serves
//   /recipes/12/cook?scale=1.5   (a multiple) for one that doesn't
// The recipe page passes what it's showing; a planned meal can pass its own.

import type {LocationQuery} from 'vue-router'

export interface CookingScale {
	servings: number // 0 when the recipe doesn't say
	factor: number // multiplier for the ingredient amounts
}

function positive(value: LocationQuery[string]): number {
	// A query value can be a string, an array (?a=1&a=2) or null.
	const n = Number(Array.isArray(value) ? value[0] : value)
	return Number.isFinite(n) && n > 0 ? n : 0
}

export function cookingScale(recipeServings: number, query: LocationQuery): CookingScale {
	if (recipeServings > 0) {
		const servings = positive(query.servings) || recipeServings
		return {servings, factor: servings / recipeServings}
	}
	return {servings: 0, factor: positive(query.scale) || 1}
}

/** The query for opening cooking mode at the amount the recipe page shows. */
export function cookingQuery(recipeServings: number, amount: number): Record<string, string> {
	if (recipeServings > 0) return {servings: String(amount)}
	return amount === 1 ? {} : {scale: String(amount)}
}
