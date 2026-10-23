// Recipes opened lately on this device, newest first, for the Home page.

import {useStorage} from '@vueuse/core'

const KEEP = 6

export function useRecentRecipes() {
	// useStorage keeps a ref in step with localStorage (cleared on logout).
	return useStorage<number[]>('mandarin.recipes.recent', [])
}

/** ids with id moved to the front, at most KEEP long. */
export function withRecent(ids: readonly number[], id: number): number[] {
	return [id, ...ids.filter(other => other !== id)].slice(0, KEEP)
}

export function rememberRecipeViewed(id: number) {
	const recent = useRecentRecipes()
	recent.value = withRecent(recent.value, id)
}
