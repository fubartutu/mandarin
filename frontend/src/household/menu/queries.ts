// Menu API calls, following the conventions in .agents/docs/api.md.
// The menu is online-only like recipes: TanStack Query fetches each week,
// and the service worker's API cache keeps an opened week readable offline.

import {queryOptions, useMutation} from '@tanstack/vue-query'
import type {QueryClient} from '@tanstack/vue-query'

import {menuCreate, menuDelete, menuList, menuUpdate} from '@/client/generated'
import type {EntryChangesWritable, MenuEntry} from '@/client/generated'
import {contextMutationOptions} from '@/client/queries/contextMutation'
import {fetchAllPages} from '@/client/queries/fetchAllPages'

export type Meal = 'breakfast' | 'lunch' | 'dinner'

/** An entry is a recipe (recipeId set, note empty) or a note (recipeId 0). */
export interface MenuEntryResponse {
	id: number
	day: string // YYYY-MM-DD
	meal: Meal
	position: number
	note: string
	recipeId: number
	recipeTitle: string
	servings: number
}

export function normalizeEntry(e: MenuEntry): MenuEntryResponse {
	return {
		id: e.id ?? 0,
		day: e.day ?? '',
		meal: e.meal ?? 'dinner',
		position: e.position ?? 0,
		note: e.note ?? '',
		recipeId: e.recipe_id ?? 0,
		recipeTitle: e.recipe_title ?? '',
		servings: e.servings ?? 0,
	}
}

/** What to add: a note's text, or a recipe (its servings default to the recipe's). */
export type NewEntry = {day: string, meal: Meal} & ({note: string} | {recipeId: number})

export const menuKeys = {
	all: ['menu'] as const,
	range: (from: string, to: string) => [...menuKeys.all, 'range', from, to] as const,
}

/** The entries from one day to another, both included. */
export function menuRangeQuery(from: string, to: string) {
	return queryOptions({
		queryKey: menuKeys.range(from, to),
		queryFn: () => fetchAllPages(async page => (await menuList({query: {from, to, page}})).data),
		select: entries => entries.map(normalizeEntry),
		staleTime: 30 * 1000,
	})
}

// Each mutation refreshes the menu afterwards, so every open week shows it.
const refresh = (client: QueryClient) => client.invalidateQueries({queryKey: menuKeys.all})

export function useAddEntryMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async (entry: NewEntry) => {
			// `'recipeId' in entry` narrows the union type to the recipe case.
			const body = 'recipeId' in entry
				? {day: entry.day, meal: entry.meal, recipe_id: entry.recipeId}
				: {day: entry.day, meal: entry.meal, note: entry.note}
			return normalizeEntry((await menuCreate({body})).data)
		},
		onSettled: (_input, client) => refresh(client),
	}))
}

/** Changes servings (recipe entries), the note (note entries) or the day (goes last there); fields left out stay. */
export function useUpdateEntryMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async ({id, changes}: {id: number, changes: EntryChangesWritable}) =>
			normalizeEntry((await menuUpdate({path: {entry: id}, body: changes})).data),
		onSettled: (_input, client) => refresh(client),
	}))
}

export function useDeleteEntryMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async (id: number) => {
			await menuDelete({path: {entry: id}})
		},
		onSettled: (_input, client) => refresh(client),
	}))
}
