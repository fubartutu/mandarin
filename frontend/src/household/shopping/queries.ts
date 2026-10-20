// Shopping API calls that need a connection, following the conventions in
// .agents/docs/api.md. Items are read and changed through the offline sync
// engine instead (sync/); lists, shops and photos are managed online here,
// and each change asks the engine to sync so every screen picks it up.

import {queryOptions, useMutation} from '@tanstack/vue-query'

import {
	shoppingCatalogPhotoDelete,
	shoppingCatalogPhotoGet,
	shoppingCatalogPhotoUpload,
	shoppingListsCreate,
	shoppingListsDelete,
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
import {expectBlob} from '@/client/queries/blobResponse'
import {contextMutationOptions} from '@/client/queries/contextMutation'
import type {SyncStorage} from './sync/storage'
import {syncShopping} from './sync/useShoppingSync'

// ---- Normalised shapes ----
// The generated types mark every field optional. These are what the UI works
// with: the fields the backend always sends are required.

export type ShoppingShopResponse = Required<Pick<ShoppingShop, 'id' | 'list_id' | 'title' | 'position'>>
export type ShoppingListResponse = Required<Pick<ShoppingList, 'id' | 'title' | 'position'>> & {
	shops: ShoppingShopResponse[]
}
export type ShoppingCategoryResponse = Required<Pick<ShoppingCategory, 'id' | 'title' | 'position'>>
export type ShoppingItemResponse = Required<Pick<ShoppingItem,
	'id' | 'uid' | 'list_id' | 'name' | 'quantity' | 'unit' | 'note' | 'category_id' | 'done' | 'position' |
	'catalog_item_id' | 'has_photo'>> &
	Pick<ShoppingItem, 'done_at' | 'created_by'> & {not_stocked_shop_ids: number[]}
export type CatalogEntry = Required<Pick<ShoppingCatalogItem,
	'id' | 'name' | 'category_id' | 'quantity' | 'unit' | 'times_added' | 'has_photo' | 'photo_id'>> &
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
		photo_id: entry.photo_id ?? 0,
		last_added: entry.last_added,
		not_stocked_shop_ids: entry.not_stocked_shop_ids ?? [],
	}
}

export function normalizeCategory(category: ShoppingCategory): ShoppingCategoryResponse {
	return {id: category.id ?? 0, title: category.title ?? '', position: category.position ?? 0}
}

export function normalizeItem(item: ShoppingItem): ShoppingItemResponse {
	return {
		id: item.id ?? 0, // 0 until an item added offline reaches the server
		uid: item.uid ?? '',
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
	photo: (photoId: number) => [...shoppingKeys.all, 'photo', photoId] as const,
}

// ---- Photos ----

/**
 * A catalogue photo as a blob. Photos are kept in IndexedDB by photo_id
 * (which changes when the photo is replaced), so they show offline too.
 */
export function shoppingPhotoQuery(catalogId: number, photoId: number, storage: SyncStorage) {
	return queryOptions({
		queryKey: shoppingKeys.photo(photoId),
		queryFn: async ({signal}) => {
			const cached = await storage.getPhoto(photoId).catch(() => undefined)
			if (cached) return cached
			const blob = expectBlob(
				(await shoppingCatalogPhotoGet({path: {id: catalogId}, parseAs: 'blob', signal})).data,
				'Shopping photo',
			)
			await storage.putPhoto(photoId, blob).catch(() => undefined) // a full disk just means no offline copy
			return blob
		},
		staleTime: Infinity,
		// The row just shows no thumbnail if it fails.
		meta: {handlesError: true},
		retry: false,
	})
}

// ---- List mutations ----

export function createShoppingListMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (title: string) =>
			normalizeList((await shoppingListsCreate({body: {title}})).data),
		onSettled: () => syncShopping(),
	})
}

export function useCreateShoppingListMutation() {
	return useMutation(createShoppingListMutationOptions())
}

export function renameShoppingListMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (list: ShoppingListResponse) =>
			normalizeList((await shoppingListsUpdate({path: {list: list.id}, body: list})).data),
		onSettled: () => syncShopping(),
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
		onSettled: () => syncShopping(),
	})
}

export function useDeleteShoppingListMutation() {
	return useMutation(deleteShoppingListMutationOptions())
}

// ---- Shops ----

export function createShopMutationOptions() {
	return contextMutationOptions({
		mutationFn: async ({listId, title}: {listId: number, title: string}) =>
			normalizeShop((await shoppingShopsCreate({path: {list: listId}, body: {title}})).data),
		onSettled: () => syncShopping(),
	})
}

export function useCreateShopMutation() {
	return useMutation(createShopMutationOptions())
}

export function renameShopMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (shop: ShoppingShopResponse) =>
			normalizeShop((await shoppingShopsUpdate({path: {list: shop.list_id, shop: shop.id}, body: shop})).data),
		onSettled: () => syncShopping(),
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
		// The sync brings the list without the shop, and the marks gone.
		onSettled: () => syncShopping(),
	})
}

export function useDeleteShopMutation() {
	return useMutation(deleteShopMutationOptions())
}

// ---- Catalogue photos (need a connection) ----

export function uploadPhotoMutationOptions() {
	return contextMutationOptions({
		mutationFn: async ({catalogId, file}: {catalogId: number, file: File}) =>
			normalizeCatalogEntry((await shoppingCatalogPhotoUpload({path: {id: catalogId}, body: {photo: file}})).data),
		// The new photo_id arrives with the sync, and the row loads the new photo.
		onSettled: () => syncShopping(),
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
		onSettled: () => syncShopping(),
	})
}

export function useDeletePhotoMutation() {
	return useMutation(deletePhotoMutationOptions())
}
