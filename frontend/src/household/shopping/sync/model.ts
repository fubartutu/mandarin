// The phone's copy of the household's shopping data, and the pure functions
// that change it. No Vue, no IndexedDB, no network: engine.ts wires those in.
//
// Two copies exist at runtime:
//   confirmed  what the server last said (also stored in IndexedDB)
//   displayed  confirmed + the changes still waiting in the outbox
// After every sync, displayed is rebuilt from confirmed by replaying the
// outbox (rebase), so a change made while a sync was in flight isn't lost.

import type {
	ShoppingCatalogItem,
	ShoppingCategory,
	ShoppingConflict,
	ShoppingItem,
	ShoppingItemFields,
	ShoppingList,
	ShoppingSyncResult,
	User,
} from '@/client/generated'

import {
	normalizeCatalogEntry,
	normalizeCategory,
	normalizeItem,
	normalizeList,
	type CatalogEntry,
	type ShoppingCategoryResponse,
	type ShoppingItemResponse,
	type ShoppingListResponse,
} from '../queries'

export type ItemFields = ShoppingItemFields

export type OpType = 'item.create' | 'item.update' | 'item.delete' | 'item.not_stocked'

/** One change made on this device, waiting to be sent. */
export interface SyncOp {
	id: string
	type: OpType
	at: number // Unix ms on this device
	item_uid: string
	item_name?: string
	list_id?: number
	set?: ItemFields
	before?: ItemFields
	shop_ids?: number[]
}

export interface ConflictEntry {
	id: number
	item_uid: string
	item_name: string
	field: string
	winner_value: string
	winner_by?: User
	winner_at: string
	loser_value: string
	loser_by?: User
	loser_at: string
}

export interface ShoppingData {
	lists: Map<number, ShoppingListResponse>
	categories: Map<number, ShoppingCategoryResponse>
	catalog: Map<number, CatalogEntry>
	items: Map<string, ShoppingItemResponse> // by uid
	conflicts: Map<number, ConflictEntry>
}

export function emptyData(): ShoppingData {
	return {lists: new Map(), categories: new Map(), catalog: new Map(), items: new Map(), conflicts: new Map()}
}

/** A server answer, with the generated types' optional fields filled in. */
export interface SyncResult {
	cursor: number
	full: boolean
	lists: ShoppingListResponse[]
	deletedListIds: number[]
	categories: ShoppingCategoryResponse[]
	deletedCategoryIds: number[]
	catalog: CatalogEntry[]
	deletedCatalogIds: number[]
	items: ShoppingItemResponse[]
	deletedItemUids: string[]
	conflicts: ConflictEntry[]
	rejected: {opId: string, reason: string}[]
}

export function normalizeConflict(c: ShoppingConflict): ConflictEntry {
	return {
		id: c.id ?? 0,
		item_uid: c.item_uid ?? '',
		item_name: c.item_name ?? '',
		field: c.field ?? '',
		winner_value: c.winner_value ?? '',
		winner_by: c.winner_by,
		winner_at: c.winner_at ?? '',
		loser_value: c.loser_value ?? '',
		loser_by: c.loser_by,
		loser_at: c.loser_at ?? '',
	}
}

export function normalizeSyncResult(r: ShoppingSyncResult): SyncResult {
	return {
		cursor: r.cursor ?? 0,
		full: r.full ?? false,
		lists: (r.lists ?? []).map((l: ShoppingList) => normalizeList(l)),
		deletedListIds: r.deleted_list_ids ?? [],
		categories: (r.categories ?? []).map((c: ShoppingCategory) => normalizeCategory(c)),
		deletedCategoryIds: r.deleted_category_ids ?? [],
		catalog: (r.catalog ?? []).map((e: ShoppingCatalogItem) => normalizeCatalogEntry(e)),
		deletedCatalogIds: r.deleted_catalog_ids ?? [],
		items: (r.items ?? []).map((i: ShoppingItem) => normalizeItem(i)),
		deletedItemUids: r.deleted_item_uids ?? [],
		conflicts: (r.conflicts ?? []).map(normalizeConflict),
		rejected: (r.rejected ?? []).map(x => ({opId: x.op_id ?? '', reason: x.reason ?? ''})),
	}
}

// Merges rows into a copy of a map: upserts by key, then deletions.
function merge<K, V>(current: Map<K, V>, replace: boolean, rows: V[], key: (row: V) => K, deleted: K[]): Map<K, V> {
	const next = replace ? new Map<K, V>() : new Map(current)
	for (const row of rows) next.set(key(row), row)
	for (const k of deleted) next.delete(k)
	return next
}

/** Applies a server answer to the confirmed data. */
export function applyResult(data: ShoppingData, r: SyncResult): ShoppingData {
	return linkCatalog({
		lists: merge(data.lists, r.full, r.lists, l => l.id, r.deletedListIds),
		categories: merge(data.categories, r.full, r.categories, c => c.id, r.deletedCategoryIds),
		catalog: merge(data.catalog, r.full, r.catalog, e => e.id, r.deletedCatalogIds),
		items: merge(data.items, r.full, r.items, i => i.uid, r.deletedItemUids),
		conflicts: merge(data.conflicts, r.full, r.conflicts, c => c.id, []),
	})
}

/**
 * Copies photo and not-stocked info from catalogue entries onto items. The
 * server only resends an item when the item changes, so a photo or mark
 * added to its catalogue entry has to be carried over here.
 */
export function linkCatalog(data: ShoppingData): ShoppingData {
	const items = new Map<string, ShoppingItemResponse>()
	for (const [uid, item] of data.items) {
		const entry = data.catalog.get(item.catalog_item_id)
		items.set(uid, entry
			? {...item, has_photo: entry.has_photo, not_stocked_shop_ids: entry.not_stocked_shop_ids}
			: item)
	}
	return {...data, items}
}

function findEntryByName(data: ShoppingData, name: string): CatalogEntry | undefined {
	const key = name.trim().toLowerCase()
	for (const entry of data.catalog.values()) {
		if (entry.name.toLowerCase() === key) return entry
	}
	return undefined
}

function withFields(item: ShoppingItemResponse, set: ItemFields, at: number, data: ShoppingData): ShoppingItemResponse {
	const next = {...item}
	if (set.name !== undefined) {
		next.name = set.name.trim()
		// The server relinks a renamed item to the catalogue by name.
		const entry = findEntryByName(data, next.name)
		next.catalog_item_id = entry?.id ?? 0
		next.has_photo = entry?.has_photo ?? false
		next.not_stocked_shop_ids = entry?.not_stocked_shop_ids ?? []
	}
	if (set.quantity !== undefined) next.quantity = set.quantity
	if (set.unit !== undefined) next.unit = set.unit
	if (set.note !== undefined) next.note = set.note
	if (set.category_id !== undefined) next.category_id = set.category_id
	if (set.position !== undefined) next.position = set.position
	if (set.done !== undefined && set.done !== item.done) {
		next.done = set.done
		next.done_at = set.done ? new Date(at).toISOString() : undefined
	}
	return next
}

/** Applies one local change, the way the server will. */
export function applyOp(data: ShoppingData, op: SyncOp): ShoppingData {
	const items = new Map(data.items)
	const current = items.get(op.item_uid)

	switch (op.type) {
		case 'item.create': {
			if (current || !op.set || !op.list_id) return data
			const blank = normalizeItem({uid: op.item_uid, list_id: op.list_id, name: '', done: false})
			items.set(op.item_uid, withFields(blank, {name: '', ...op.set}, op.at, data))
			return {...data, items}
		}
		case 'item.update':
			if (!current || !op.set) return data
			items.set(op.item_uid, withFields(current, op.set, op.at, data))
			return {...data, items}
		case 'item.delete':
			if (!current) return data
			items.delete(op.item_uid)
			return {...data, items}
		case 'item.not_stocked': {
			if (!current) return data
			const shopIds = op.shop_ids ?? []
			const catalog = new Map(data.catalog)
			const entry = catalog.get(current.catalog_item_id)
			if (entry) catalog.set(entry.id, {...entry, not_stocked_shop_ids: shopIds})
			items.set(op.item_uid, {...current, not_stocked_shop_ids: shopIds})
			// Other items with the same catalogue entry follow.
			return entry ? linkCatalog({...data, items, catalog}) : {...data, items}
		}
	}
	return data
}

/** confirmed + the waiting changes: what the screen shows. */
export function rebase(confirmed: ShoppingData, outbox: SyncOp[]): ShoppingData {
	return outbox.reduce(applyOp, confirmed)
}

// ---- Building ops from what the user did ----

function newId(): string {
	// crypto.randomUUID needs HTTPS (or localhost); getRandomValues works on
	// plain http too, e.g. when the app is opened by its LAN address.
	const bytes = crypto.getRandomValues(new Uint8Array(16))
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // RFC 4122 variant
	const hex = [...bytes].map(b => b.toString(16).padStart(2, '0')).join('')
	return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}
export {newId as newUid}

export type NewItem = Pick<ShoppingItemResponse, 'list_id' | 'name' | 'quantity' | 'unit' | 'category_id'>

export function createOp(item: NewItem, at = Date.now()): SyncOp {
	return {
		id: newId(),
		type: 'item.create',
		at,
		item_uid: newId(),
		item_name: item.name,
		list_id: item.list_id,
		set: {name: item.name, quantity: item.quantity, unit: item.unit, category_id: item.category_id},
	}
}

type Editable = Pick<ShoppingItemResponse, 'name' | 'quantity' | 'unit' | 'note' | 'category_id' | 'done' | 'position'>

/**
 * An update op with just the fields that differ, plus their old values
 * (so the server can tell a conflict). null when nothing changed.
 * Quantity and unit are always sent together.
 */
export function updateOp(item: ShoppingItemResponse, changes: Partial<Editable>, at = Date.now()): SyncOp | null {
	const set: ItemFields = {}
	const before: ItemFields = {}
	const simple = ['name', 'note', 'category_id', 'done', 'position'] as const
	for (const field of simple) {
		const value = changes[field]
		if (value !== undefined && value !== item[field]) {
			// Indexed writes need a cast: TypeScript can't tie field to its value type here.
			(set as Record<string, unknown>)[field] = value;
			(before as Record<string, unknown>)[field] = item[field]
		}
	}
	const quantity = changes.quantity ?? item.quantity
	const unit = changes.unit ?? item.unit
	if (quantity !== item.quantity || unit !== item.unit) {
		Object.assign(set, {quantity, unit})
		Object.assign(before, {quantity: item.quantity, unit: item.unit})
	}
	if (Object.keys(set).length === 0) return null
	return {id: newId(), type: 'item.update', at, item_uid: item.uid, item_name: item.name, set, before}
}

export function deleteOp(item: ShoppingItemResponse, at = Date.now()): SyncOp {
	return {id: newId(), type: 'item.delete', at, item_uid: item.uid, item_name: item.name}
}

export function notStockedOp(item: ShoppingItemResponse, shopIds: number[], at = Date.now()): SyncOp {
	return {id: newId(), type: 'item.not_stocked', at, item_uid: item.uid, item_name: item.name, shop_ids: shopIds}
}
