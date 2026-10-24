// Helpers for planning a recipe from outside the menu page (the recipe
// page's "Plan" button): where the menu shows a day, and how the
// confirmation names it.

import type {RouteLocationRaw} from 'vue-router'

import {fromKey, mondayOf} from './week'

/**
 * The menu page showing the week a YYYY-MM-DD day is in: plain /menu for
 * this week (so it stays "this week" tomorrow), else /menu/<that Monday>.
 */
export function menuWeekRoute(day: string, now: Date): RouteLocationRaw {
	const monday = mondayOf(fromKey(day))
	return monday === mondayOf(now)
		? {name: 'menu'}
		: {name: 'menu', params: {week: monday}}
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
