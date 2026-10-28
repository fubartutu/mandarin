// Helpers for planning a recipe from outside the menu page (the recipe
// page's "Plan" button): where the menu shows a day, and how the
// confirmation names it.

import type {RouteLocationRaw} from 'vue-router'

import {dayKey, fromKey, pageStart} from './week'

/**
 * The menu page showing a YYYY-MM-DD day. Pages are 7 days starting from
 * today: plain /menu for the next 7 days (so it moves on with the days),
 * else /menu/<first day of that page>, a whole number of weeks from today.
 */
export function menuWeekRoute(day: string, now: Date): RouteLocationRaw {
	const today = dayKey(now)
	const start = pageStart(day, today)
	return start === today
		? {name: 'menu'}
		: {name: 'menu', params: {start}}
}

/** "Tue, 27 Oct" (in the locale's own order and words). */
export function plannedDayLabel(day: string, locale: string): string {
	return fromKey(day).toLocaleDateString(locale, {weekday: 'short', day: 'numeric', month: 'short'})
}

/**
 * How much of a recipe a planned entry needs: the entry's servings over the
 * recipe's (recipe for 4, entry for 6 gives 1.5). When either isn't known
 * (0), the recipe is used as written.
 */
export function entryFactor(entryServings: number, recipeServings: number): number {
	return entryServings > 0 && recipeServings > 0 ? entryServings / recipeServings : 1
}

/**
 * Cooking mode for a planned recipe entry, at the entry's servings when it
 * has them (else cooking mode uses the recipe as written).
 */
export function cookRoute(entry: {recipeId: number, servings: number}): RouteLocationRaw {
	return entry.servings > 0
		? {name: 'recipe.cook', params: {recipeId: entry.recipeId}, query: {servings: String(entry.servings)}}
		: {name: 'recipe.cook', params: {recipeId: entry.recipeId}}
}
