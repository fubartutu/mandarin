// Shopping data layer, following the conventions in .agents/docs/api.md:
// one key factory, query/mutation option factories, thin use*Mutation hooks.
// Components read through useShopping.ts and write through the hooks here.

import {queryOptions, useMutation, type QueryClient} from '@tanstack/vue-query'

import {
	shoppingCategoriesList,
	shoppingItemsCreate,
	shoppingItemsDelete,
	shoppingItemsList,
	shoppingItemsUpdate,
	shoppingListsClearDone,
	shoppingListsCreate,
	shoppingListsDelete,
	shoppingListsList,
	shoppingListsUpdate,
} from '@/client/generated'
import type {ShoppingCategory, ShoppingItem, ShoppingList} from '@/client/generated'
import {fetchAllPages} from '@/client/queries/fetchAllPages'
import {contextMutationOptions} from '@/client/queries/contextMutation'

// ---- Normalised shapes ----
// The generated types mark every field optional. These are what the UI works
// with: the fields the backend always sends are required.

export type ShoppingListResponse = Required<Pick<ShoppingList, 'id' | 'title' | 'position'>>
export type ShoppingCategoryResponse = Required<Pick<ShoppingCategory, 'id' | 'title' | 'position'>>
export type ShoppingItemResponse = Required<Pick<ShoppingItem,
	'id' | 'list_id' | 'name' | 'quantity' | 'unit' | 'note' | 'category_id' | 'done' | 'position'>> &
	Pick<ShoppingItem, 'done_at' | 'created_by'>

export function normalizeList(list: ShoppingList): ShoppingListResponse {
	return {id: list.id ?? 0, title: list.title ?? '', position: list.position ?? 0}
}

export function normalizeCategory(category: ShoppingCategory): ShoppingCategoryResponse {
	return {id: category.id ?? 0, title: category.title ?? '', position: category.position ?? 0}
}

export function normalizeItem(item: ShoppingItem): ShoppingItemResponse {
	return {
		id: item.id ?? 0,
		list_id: item.list_id ?? 0,
		name: item.name ?? '',
		quantity: item.quantity ?? 0,
		unit: item.unit ?? '',
		note: item.note ?? '',
		category_id: item.category_id ?? 0,
		done: item.done ?? false,
		position: item.position ?? 0,
		done_at: item.done_at,
		created_by: item.created_by,
	}
}

// ---- Keys ----

export const shoppingKeys = {
	all: ['shopping'] as const,
	lists: () => [...shoppingKeys.all, 'lists'] as const,
	categories: () => [...shoppingKeys.all, 'categories'] as const,
	items: (listId: number) => [...shoppingKeys.all, 'items', listId] as const,
}

// ---- Queries ----

export function shoppingListsQuery() {
	return queryOptions({
		queryKey: shoppingKeys.lists(),
		queryFn: async () => (await fetchAllPages(async page =>
			(await shoppingListsList({query: {page}})).data)).map(normalizeList),
	})
}

export function shoppingCategoriesQuery() {
	return queryOptions({
		queryKey: shoppingKeys.categories(),
		queryFn: async () => (await fetchAllPages(async page =>
			(await shoppingCategoriesList({query: {page}})).data)).map(normalizeCategory),
		staleTime: 5 * 60 * 1000,
	})
}

export function shoppingItemsQuery(listId: number) {
	return queryOptions({
		queryKey: shoppingKeys.items(listId),
		queryFn: async () => (await fetchAllPages(async page =>
			(await shoppingItemsList({path: {list: listId}, query: {page}})).data)).map(normalizeItem),
		// Someone else may be shopping: refresh when the phone comes back to the app.
		refetchOnWindowFocus: 'always',
	})
}

// ---- List mutations ----

export function createShoppingListMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (title: string) =>
			normalizeList((await shoppingListsCreate({body: {title}})).data),
		onSettled: (_input, client) => client.invalidateQueries({queryKey: shoppingKeys.lists()}),
	})
}

export function useCreateShoppingListMutation() {
	return useMutation(createShoppingListMutationOptions())
}

export function renameShoppingListMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (list: ShoppingListResponse) =>
			normalizeList((await shoppingListsUpdate({path: {list: list.id}, body: list})).data),
		onSettled: (_input, client) => client.invalidateQueries({queryKey: shoppingKeys.lists()}),
	})
}

export function useRenameShoppingListMutation() {
	return useMutation(renameShoppingListMutationOptions())
}

export function deleteShoppingListMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (listId: number) => {
			await shoppingListsDelete({path: {list: listId}})
		},
		onSuccess: (_data, listId, client) => client.removeQueries({queryKey: shoppingKeys.items(listId)}),
		onSettled: (_input, client) => client.invalidateQueries({queryKey: shoppingKeys.lists()}),
	})
}

export function useDeleteShoppingListMutation() {
	return useMutation(deleteShoppingListMutationOptions())
}

// ---- Item mutations (optimistic: the screen changes before the server answers) ----

function patchItems(client: QueryClient, listId: number, update: (items: ShoppingItemResponse[]) => ShoppingItemResponse[]) {
	// Bail when nothing is cached, so we never invent a list nobody loaded.
	client.setQueryData<ShoppingItemResponse[]>(shoppingKeys.items(listId), current => current ? update(current) : current)
}

export type NewShoppingItem = Pick<ShoppingItemResponse, 'list_id' | 'name' | 'quantity' | 'unit' | 'category_id'> &
	Partial<Pick<ShoppingItemResponse, 'note'>>

// Temporary ids for optimistic rows are negative so they never clash with real ones.
let nextTempId = -1

export function createShoppingItemMutationOptions() {
	return contextMutationOptions({
		mutationFn: async ({list_id, ...item}: NewShoppingItem) =>
			normalizeItem((await shoppingItemsCreate({path: {list: list_id}, body: item})).data),
		optimistic: {
			queryKeys: item => [shoppingKeys.items(item.list_id)],
			update: (item, client) => {
				const tempId = nextTempId--
				patchItems(client, item.list_id, items => [
					...items,
					normalizeItem({...item, id: tempId, done: false, position: 0}),
				])
				return tempId
			},
		},
		onSuccess: (created, item, client, tempId) =>
			patchItems(client, item.list_id, items => items.map(i => i.id === tempId ? created : i)),
		onSettled: (item, client) => client.invalidateQueries({queryKey: shoppingKeys.items(item.list_id)}),
	})
}

export function useCreateShoppingItemMutation() {
	return useMutation(createShoppingItemMutationOptions())
}

export function updateShoppingItemMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (item: ShoppingItemResponse) =>
			normalizeItem((await shoppingItemsUpdate({path: {list: item.list_id, item: item.id}, body: item})).data),
		optimistic: {
			queryKeys: item => [shoppingKeys.items(item.list_id)],
			update: (item, client) => patchItems(client, item.list_id, items =>
				items.map(i => i.id === item.id ? item : i)),
		},
		onSuccess: (updated, item, client) =>
			patchItems(client, item.list_id, items => items.map(i => i.id === updated.id ? updated : i)),
		onSettled: (item, client) => client.invalidateQueries({queryKey: shoppingKeys.items(item.list_id)}),
	})
}

export function useUpdateShoppingItemMutation() {
	return useMutation(updateShoppingItemMutationOptions())
}

export function deleteShoppingItemMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (item: Pick<ShoppingItemResponse, 'id' | 'list_id'>) => {
			await shoppingItemsDelete({path: {list: item.list_id, item: item.id}})
		},
		optimistic: {
			queryKeys: item => [shoppingKeys.items(item.list_id)],
			update: (item, client) => patchItems(client, item.list_id, items => items.filter(i => i.id !== item.id)),
		},
		onSettled: (item, client) => client.invalidateQueries({queryKey: shoppingKeys.items(item.list_id)}),
	})
}

export function useDeleteShoppingItemMutation() {
	return useMutation(deleteShoppingItemMutationOptions())
}

export function clearDoneItemsMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (listId: number) =>
			(await shoppingListsClearDone({path: {list: listId}})).data.deleted ?? 0,
		optimistic: {
			queryKeys: listId => [shoppingKeys.items(listId)],
			update: (listId, client) => patchItems(client, listId, items => items.filter(i => !i.done)),
		},
		onSettled: (listId, client) => client.invalidateQueries({queryKey: shoppingKeys.items(listId)}),
	})
}

export function useClearDoneItemsMutation() {
	return useMutation(clearDoneItemsMutationOptions())
}
