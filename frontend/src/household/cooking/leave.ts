import type {Router} from 'vue-router'

/**
 * Closes cooking mode: a history step back to the page it was opened from
 * (recipe, Menu or Home), so the browser's back button doesn't then return
 * to cooking mode. Opened straight from a link, there is nothing to go back
 * to in the app, so the recipe page takes its place.
 */
export function leaveCooking(
	router: Pick<Router, 'back' | 'replace'>,
	recipeId: number,
	// vue-router keeps the previous in-app path in history.state.back.
	historyState: {back?: unknown} | null = window.history.state,
) {
	if (historyState?.back) {
		router.back()
		return
	}
	router.replace({
		name: 'recipe',
		params: {recipeId},
	})
}
