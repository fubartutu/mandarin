// Recipe API calls, following the conventions in .agents/docs/api.md.
// Recipes are online-only: they're fetched with TanStack Query, whose cache
// keeps a recipe on screen if the signal drops while cooking.

import {queryOptions, useMutation} from '@tanstack/vue-query'

import {
	recipesCreate,
	recipesDelete,
	recipesImageDelete,
	recipesImageFetch,
	recipesImageGet,
	recipesImageMain,
	recipesImageUpload,
	recipesImport,
	recipesList,
	recipesRead,
	recipesUpdate,
} from '@/client/generated'
import type {Recipe, RecipeIngredient} from '@/client/generated'
import {expectBlob} from '@/client/queries/blobResponse'
import {contextMutationOptions} from '@/client/queries/contextMutation'
import {fetchAllPages} from '@/client/queries/fetchAllPages'

// ---- Normalised shapes ----

export type IngredientLine = Required<RecipeIngredient>

export interface RecipeResponse {
	id: number
	title: string
	source_url: string
	description: string
	servings: number
	yields: string
	prep_minutes: number
	cook_minutes: number
	total_minutes: number
	ingredients: IngredientLine[]
	steps: string[]
	tags: string[]
	notes: string
	image_ids: number[]
	image_urls: string[] // import previews only
}

/** What the editor sends: the server works out amounts from each line's text. */
export type RecipeDraft = Omit<RecipeResponse, 'id' | 'image_ids' | 'image_urls' | 'ingredients'> & {
	ingredients: Pick<IngredientLine, 'group' | 'text'>[]
}

export function normalizeIngredient(i: RecipeIngredient): IngredientLine {
	return {
		group: i.group ?? '',
		text: i.text ?? '',
		quantity: i.quantity ?? 0,
		unit: i.unit ?? '',
		name: i.name ?? '',
		note: i.note ?? '',
	}
}

export function normalizeRecipe(r: Recipe): RecipeResponse {
	return {
		id: r.id ?? 0,
		title: r.title ?? '',
		source_url: r.source_url ?? '',
		description: r.description ?? '',
		servings: r.servings ?? 0,
		yields: r.yields ?? '',
		prep_minutes: r.prep_minutes ?? 0,
		cook_minutes: r.cook_minutes ?? 0,
		total_minutes: r.total_minutes ?? 0,
		ingredients: (r.ingredients ?? []).map(normalizeIngredient),
		steps: r.steps ?? [],
		tags: r.tags ?? [],
		notes: r.notes ?? '',
		image_ids: r.image_ids ?? [],
		image_urls: r.image_urls ?? [],
	}
}

// ---- Keys and queries ----

export const recipeKeys = {
	all: ['recipes'] as const,
	list: () => [...recipeKeys.all, 'list'] as const,
	detail: (id: number) => [...recipeKeys.all, 'detail', id] as const,
	image: (imageId: number) => [...recipeKeys.all, 'image', imageId] as const,
}

/** Every recipe; the list is filtered on the device, so search is instant. */
export function recipesQuery() {
	return queryOptions({
		queryKey: recipeKeys.list(),
		queryFn: () => fetchAllPages(async page => (await recipesList({query: {page}})).data),
		select: recipes => recipes.map(normalizeRecipe),
		staleTime: 60 * 1000,
	})
}

export function recipeQuery(id: number) {
	return queryOptions({
		queryKey: recipeKeys.detail(id),
		queryFn: async ({signal}) => normalizeRecipe((await recipesRead({path: {recipe: id}, signal})).data),
		staleTime: 60 * 1000,
	})
}

/** A picture as a blob (it needs the login token, so no plain <img src>). Pictures never change. */
export function recipeImageQuery(recipeId: number, imageId: number) {
	return queryOptions({
		queryKey: recipeKeys.image(imageId),
		queryFn: async ({signal}) => expectBlob(
			(await recipesImageGet({path: {recipe: recipeId, image: imageId}, parseAs: 'blob', signal})).data,
			'Recipe image',
		),
		staleTime: Infinity,
		gcTime: 30 * 60 * 1000,
		meta: {handlesError: true}, // a missing picture just isn't shown
		retry: false,
	})
}

// ---- Mutations ----
// Each one refreshes the recipe queries afterwards (onSettled), so lists and
// the open recipe show the change.

const refresh = (client: import('@tanstack/vue-query').QueryClient) =>
	client.invalidateQueries({queryKey: recipeKeys.all, predicate: q => q.queryKey[1] !== 'image'})

export function useImportRecipeMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async (url: string) => normalizeRecipe((await recipesImport({body: {url}})).data),
	}))
}

export function useSaveRecipeMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async ({id, draft}: {id: number, draft: RecipeDraft}) => normalizeRecipe(id
			? (await recipesUpdate({path: {recipe: id}, body: draft})).data
			: (await recipesCreate({body: draft})).data),
		onSettled: (_input, client) => refresh(client),
	}))
}

export function useDeleteRecipeMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async (id: number) => {
			await recipesDelete({path: {recipe: id}})
		},
		onSettled: (_input, client) => refresh(client),
	}))
}

export function useAddImageMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async ({recipeId, file, url}: {recipeId: number, file?: File, url?: string}) => normalizeRecipe(file
			? (await recipesImageUpload({path: {recipe: recipeId}, body: {image: file}})).data
			: (await recipesImageFetch({path: {recipe: recipeId}, body: {url: url ?? ''}})).data),
		onSettled: (_input, client) => refresh(client),
	}))
}

export function useMakeImageMainMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async ({recipeId, imageId}: {recipeId: number, imageId: number}) =>
			normalizeRecipe((await recipesImageMain({path: {recipe: recipeId, image: imageId}})).data),
		onSettled: (_input, client) => refresh(client),
	}))
}

export function useDeleteImageMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async ({recipeId, imageId}: {recipeId: number, imageId: number}) => {
			await recipesImageDelete({path: {recipe: recipeId, image: imageId}})
		},
		onSettled: (_input, client) => refresh(client),
	}))
}
