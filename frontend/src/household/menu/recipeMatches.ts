// Recipe search in the menu's "+" field. The whole recipe list is already on
// the device (recipesQuery), so matching is a plain filter.

export const MAX_RECIPE_MATCHES = 6

interface Titled {
	title: string
}

/**
 * The recipes whose title contains `text` (case-insensitive), in list order,
 * at most six. Empty text matches nothing.
 *
 * `<T extends Titled>` keeps the caller's own type: pass full recipes and
 * full recipes come back.
 */
export function matchRecipes<T extends Titled>(recipes: readonly T[], text: string): T[] {
	const needle = text.trim().toLocaleLowerCase()
	if (needle === '') return []
	const found: T[] = []
	for (const recipe of recipes) {
		if (recipe.title.toLocaleLowerCase().includes(needle)) {
			found.push(recipe)
			if (found.length === MAX_RECIPE_MATCHES) break
		}
	}
	return found
}
