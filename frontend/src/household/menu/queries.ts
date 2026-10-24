// Menu API calls, following the conventions in .agents/docs/api.md.
// The menu is online-only like recipes: TanStack Query fetches each week,
// and the service worker's API cache keeps an opened week readable offline.

import {queryOptions, useMutation} from '@tanstack/vue-query'
import type {QueryClient} from '@tanstack/vue-query'

import {menuCreate, menuDelete, menuList} from '@/client/generated'
import type {MenuEntry} from '@/client/generated'
import {contextMutationOptions} from '@/client/queries/contextMutation'
import {fetchAllPages} from '@/client/queries/fetchAllPages'

export type Meal = 'breakfast' | 'lunch' | 'dinner'

export interface MenuEntryResponse {
	id: number
	day: string // YYYY-MM-DD
	meal: Meal
	position: number
	note: string
}

export function normalizeEntry(e: MenuEntry): MenuEntryResponse {
	return {
		id: e.id ?? 0,
		day: e.day ?? '',
		meal: e.meal ?? 'dinner',
		position: e.position ?? 0,
		note: e.note ?? '',
	}
}

export const menuKeys = {
	all: ['menu'] as const,
	week: (from: string, to: string) => [...menuKeys.all, 'range', from, to] as const,
}

/** The entries from one day to another, both included. */
export function menuRangeQuery(from: string, to: string) {
	return queryOptions({
		queryKey: menuKeys.week(from, to),
		queryFn: () => fetchAllPages(async page => (await menuList({query: {from, to, page}})).data),
		select: entries => entries.map(normalizeEntry),
		staleTime: 30 * 1000,
	})
}

// Each mutation refreshes the menu afterwards, so every open week shows it.
const refresh = (client: QueryClient) => client.invalidateQueries({queryKey: menuKeys.all})

export function useAddNoteMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async ({day, meal, note}: {day: string, meal: Meal, note: string}) =>
			normalizeEntry((await menuCreate({body: {day, meal, note}})).data),
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
