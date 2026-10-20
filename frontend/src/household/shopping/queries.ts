// Shopping data layer, following the conventions in .agents/docs/api.md:
// one key factory, query/mutation option factories, thin use*Mutation hooks.
// Components read through useShopping.ts and write through the hooks here.

import {queryOptions, useMutation, type QueryClient} from '@tanstack/vue-query'

import {
	shoppingCatalogList,
	shoppingCatalogPhotoDelete,
	shoppingCatalogPhotoGet,
	shoppingCatalogPhotoUpload,
	shoppingCatalogUpdate,
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
	shoppingShopsCreate,
	shoppingShopsDelete,
	shoppingShopsUpdate,
} from '@/client/generated'
import type {
	ShoppingCatalogItem,
	ShoppingCategory,
	ShoppingItem,
	ShoppingList,
	ShoppingShop,
} from '@/client/generated'
import {ApiError} from '@/client/problem'
import {expectBlob} from '@/client/queries/blobResponse'
import {fetchAllPages} from '@/client/queries/fetchAllPages'
import {contextMutationOptions} from '@/client/queries/contextMutation'
import {queryClient} from '@/client/queryClient'

// ---- Normalised shapes ----
// The generated types mark every field optional. These are what the UI works
// with: the fields the backend always sends are required.

export type ShoppingShopResponse = Required<Pick<ShoppingShop, 'id' | 'list_id' | 'title' | 'position'>>
export type ShoppingListResponse = Required<Pick<ShoppingList, 'id' | 'title' | 'position'>> & {
	shops: ShoppingShopResponse[]
}
export type ShoppingCategoryResponse = Required<Pick<ShoppingCategory, 'id' | 'title' | 'position'>>
export type ShoppingItemResponse = Required<Pick<ShoppingItem,
	'id' | 'list_id' | 'name' | 'quantity' | 'unit' | 'note' | 'category_id' | 'done' | 'position' |
	'catalog_item_id' | 'has_photo'>> &
	Pick<ShoppingItem, 'done_at' | 'created_by'> & {not_stocked_shop_ids: number[]}
export type CatalogEntry = Required<Pick<ShoppingCatalogItem,
	'id' | 'name' | 'category_id' | 'quantity' | 'unit' | 'times_added' | 'has_photo'>> &
	Pick<ShoppingCatalogItem, 'last_added'> & {not_stocked_shop_ids: number[]}

export function normalizeShop(shop: ShoppingShop): ShoppingShopResponse {
	return {id: shop.id ?? 0, list_id: shop.list_id ?? 0, title: shop.title ?? '', position: shop.position ?? 0}
}

export function normalizeList(list: ShoppingList): ShoppingListResponse {
	return {
		id: list.id ?? 0,
		title: list.title ?? '',
		position: list.position ?? 0,
		shops: (list.shops ?? []).map(normalizeShop),
	}
}

export function normalizeCatalogEntry(entry: ShoppingCatalogItem): CatalogEntry {
	return {
		id: entry.id ?? 0,
		name: entry.name ?? '',
		category_id: entry.category_id ?? 0,
		quantity: entry.quantity ?? 0,
		unit: entry.unit ?? '',
		times_added: entry.times_added ?? 0,
		has_photo: entry.has_photo ?? false,
		last_added: entry.last_added,
		not_stocked_shop_ids: entry.not_stocked_shop_ids ?? [],
	}
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
		catalog_item_id: item.catalog_item_id ?? 0,
		has_photo: item.has_photo ?? false,
		not_stocked_shop_ids: item.not_stocked_shop_ids ?? [],
		done_at: item.done_at,
		created_by: item.created_by,
	}
}

// ---- Keys ----

export const shoppingKeys = {
	all: ['shopping'] as const,
	lists: () => [...shoppingKeys.all, 'lists'] as const,
	categories: () => [...shoppingKeys.all, 'categories'] as const,
	allItems: () => [...shoppingKeys.all, 'items'] as const,
	items: (listId: number) => [...shoppingKeys.allItems(), listId] as const,
	catalog: () => [...shoppingKeys.all, 'catalog'] as const,
	photo: (catalogId: number) => [...shoppingKeys.all, 'photo', catalogId] as const,
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

export function shoppingCatalogQuery() {
	return queryOptions({
		queryKey: shoppingKeys.catalog(),
		queryFn: async () => (await fetchAllPages(async page =>
			(await shoppingCatalogList({query: {page}})).data)).map(normalizeCatalogEntry),
		staleTime: 5 * 60 * 1000,
	})
}

export function shoppingPhotoQuery(catalogId: number) {
	return queryOptions({
		queryKey: shoppingKeys.photo(catalogId),
		queryFn: async ({signal}) => expectBlob(
			(await shoppingCatalogPhotoGet({path: {id: catalogId}, parseAs: 'blob', signal})).data,
			'Shopping photo',
		),
		staleTime: Infinity,
		// The row just shows no thumbnail if it fails.
		meta: {handlesError: true},
		retry: false,
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
		onSettled: (item, client) => Promise.all([
			client.invalidateQueries({queryKey: shoppingKeys.items(item.list_id)}),
			// A new name or a new use changes the catalogue (autocomplete).
			client.invalidateQueries({queryKey: shoppingKeys.catalog()}),
		]),
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

// Updates and deletes are safe to repeat ("done: true" twice is still done),
// so a server error such as a briefly locked SQLite database is retried.
// Creates aren't retried: that could add the item twice.
function retryServerErrors(failureCount: number, error: unknown) {
	return failureCount < 2 && error instanceof ApiError && error.status >= 500
}

export function useUpdateShoppingItemMutation() {
	return useMutation({...updateShoppingItemMutationOptions(), retry: retryServerErrors})
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
	return useMutation({...deleteShoppingItemMutationOptions(), retry: retryServerErrors})
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

// ---- Shops ----

export function createShopMutationOptions() {
	return contextMutationOptions({
		mutationFn: async ({listId, title}: {listId: number, title: string}) =>
			normalizeShop((await shoppingShopsCreate({path: {list: listId}, body: {title}})).data),
		onSettled: (_input, client) => client.invalidateQueries({queryKey: shoppingKeys.lists()}),
	})
}

export function useCreateShopMutation() {
	return useMutation(createShopMutationOptions())
}

export function renameShopMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (shop: ShoppingShopResponse) =>
			normalizeShop((await shoppingShopsUpdate({path: {list: shop.list_id, shop: shop.id}, body: shop})).data),
		onSettled: (_input, client) => client.invalidateQueries({queryKey: shoppingKeys.lists()}),
	})
}

export function useRenameShopMutation() {
	return useMutation(renameShopMutationOptions())
}

export function deleteShopMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (shop: Pick<ShoppingShopResponse, 'id' | 'list_id'>) => {
			await shoppingShopsDelete({path: {list: shop.list_id, shop: shop.id}})
		},
		onSettled: (_input, client) => Promise.all([
			client.invalidateQueries({queryKey: shoppingKeys.lists()}),
			// The shop's not-stocked marks went with it.
			client.invalidateQueries({queryKey: shoppingKeys.allItems()}),
			client.invalidateQueries({queryKey: shoppingKeys.catalog()}),
		]),
	})
}

export function useDeleteShopMutation() {
	return useMutation(deleteShopMutationOptions())
}

// ---- Catalogue: not-stocked shops and photos ----

export function setNotStockedMutationOptions() {
	return contextMutationOptions({
		mutationFn: async ({catalogId, shopIds}: {catalogId: number, shopIds: number[]}) =>
			normalizeCatalogEntry((await shoppingCatalogUpdate({
				path: {id: catalogId},
				// PUT replaces the entry, so send what we have plus the new marks.
				body: await currentCatalogBody(catalogId, shopIds),
			})).data),
		optimistic: {
			queryKeys: () => [shoppingKeys.allItems()],
			update: ({catalogId, shopIds}, client) => {
				client.setQueriesData<ShoppingItemResponse[]>({queryKey: shoppingKeys.allItems()}, current =>
					current?.map(i => i.catalog_item_id === catalogId ? {...i, not_stocked_shop_ids: shopIds} : i))
			},
		},
		onSettled: (_input, client) => Promise.all([
			client.invalidateQueries({queryKey: shoppingKeys.allItems()}),
			client.invalidateQueries({queryKey: shoppingKeys.catalog()}),
		]),
	})
}

// The catalogue entry as the server last sent it, with new not-stocked marks.
async function currentCatalogBody(catalogId: number, shopIds: number[]) {
	const entries = await queryClient.ensureQueryData(shoppingCatalogQuery())
	const entry = entries.find(e => e.id === catalogId)
	if (!entry) {
		throw new Error(`Catalogue entry ${catalogId} not found`)
	}
	return {
		name: entry.name,
		category_id: entry.category_id,
		quantity: entry.quantity,
		unit: entry.unit,
		not_stocked_shop_ids: shopIds,
	}
}

export function useSetNotStockedMutation() {
	return useMutation(setNotStockedMutationOptions())
}

function invalidatePhoto(client: QueryClient, catalogId: number) {
	return Promise.all([
		client.invalidateQueries({queryKey: shoppingKeys.photo(catalogId)}),
		client.invalidateQueries({queryKey: shoppingKeys.allItems()}),
		client.invalidateQueries({queryKey: shoppingKeys.catalog()}),
	])
}

export function uploadPhotoMutationOptions() {
	return contextMutationOptions({
		mutationFn: async ({catalogId, file}: {catalogId: number, file: File}) =>
			normalizeCatalogEntry((await shoppingCatalogPhotoUpload({path: {id: catalogId}, body: {photo: file}})).data),
		onSettled: ({catalogId}, client) => invalidatePhoto(client, catalogId),
	})
}

export function useUploadPhotoMutation() {
	return useMutation(uploadPhotoMutationOptions())
}

export function deletePhotoMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (catalogId: number) => {
			await shoppingCatalogPhotoDelete({path: {id: catalogId}})
		},
		onSettled: (catalogId, client) => invalidatePhoto(client, catalogId),
	})
}

export function useDeletePhotoMutation() {
	return useMutation(deletePhotoMutationOptions())
}
