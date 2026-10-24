import type {RouteRecordRaw} from 'vue-router'

// Spread into Vikunja's route table in src/router/index.ts (marked FORK).
export const householdRoutes: RouteRecordRaw[] = [
	{
		// `:listId?` is optional: /shopping opens the last-used list.
		path: '/shopping/:listId?',
		name: 'shopping',
		component: () => import('./shopping/ShoppingView.vue'),
		// Hand the param to the component as a number prop instead of a string.
		props: route => ({listId: Number(route.params.listId) || 0}),
	},
	{
		path: '/recipes',
		name: 'recipes',
		component: () => import('./recipes/RecipesView.vue'),
	},
	{
		// /recipes/new?url=… imports that page straight away.
		path: '/recipes/new',
		name: 'recipe.new',
		component: () => import('./recipes/RecipeEditor.vue'),
		props: {recipeId: 0},
	},
	{
		// (\d+): only numbers, so "new" above isn't taken for an id.
		path: '/recipes/:recipeId(\\d+)',
		name: 'recipe',
		component: () => import('./recipes/RecipeView.vue'),
		props: route => ({recipeId: Number(route.params.recipeId)}),
	},
	{
		path: '/recipes/:recipeId(\\d+)/edit',
		name: 'recipe.edit',
		component: () => import('./recipes/RecipeEditor.vue'),
		props: route => ({recipeId: Number(route.params.recipeId)}),
	},
	{
		// ?servings=6 (or ?scale=1.5 for a recipe without servings): see cooking/servings.ts.
		path: '/recipes/:recipeId(\\d+)/cook',
		name: 'recipe.cook',
		component: () => import('./cooking/CookingView.vue'),
		props: route => ({recipeId: Number(route.params.recipeId)}),
	},
]
