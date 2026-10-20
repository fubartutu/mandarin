// Where the phone keeps its copy: IndexedDB, the browser's database, which
// survives closing the app and works offline. MemoryStorage does the same job
// in tests (jsdom has no IndexedDB).

import {applyResult, emptyData, type ShoppingData, type SyncOp, type SyncResult} from './model'

export interface Stored {
	data: ShoppingData
	outbox: SyncOp[]
	cursor: number
}

export interface SyncStorage {
	load(): Promise<Stored>
	addOp(op: SyncOp): Promise<void>
	/** Saves a server answer and drops the ops it answered, in one go. */
	saveResult(result: SyncResult, sentOpIds: string[]): Promise<void>
	/** Cached photos, by the catalogue entry's photo_id. */
	getPhoto(photoId: number): Promise<Blob | undefined>
	putPhoto(photoId: number, blob: Blob): Promise<void>
	/** Deletes cached photos whose id isn't in keep. */
	prunePhotos(keep: Set<number>): Promise<void>
}

export class MemoryStorage implements SyncStorage {
	data = emptyData()
	outbox: SyncOp[] = []
	cursor = 0
	photos = new Map<number, Blob>()

	async load(): Promise<Stored> {
		return {data: this.data, outbox: [...this.outbox], cursor: this.cursor}
	}
	async addOp(op: SyncOp) {
		this.outbox.push(op)
	}
	async saveResult(result: SyncResult, sentOpIds: string[]) {
		this.data = applyResult(this.data, result)
		this.cursor = result.cursor
		const sent = new Set(sentOpIds)
		this.outbox = this.outbox.filter(op => !sent.has(op.id))
	}
	async getPhoto(photoId: number) {
		return this.photos.get(photoId)
	}
	async putPhoto(photoId: number, blob: Blob) {
		this.photos.set(photoId, blob)
	}
	async prunePhotos(keep: Set<number>) {
		for (const id of this.photos.keys()) {
			if (!keep.has(id)) this.photos.delete(id)
		}
	}
}

// ---- IndexedDB ----

const VERSION = 1
// Object store name → the row field that is its key.
const DATA_STORES = {lists: 'id', categories: 'id', catalog: 'id', items: 'uid', conflicts: 'id'} as const
type DataStore = keyof typeof DATA_STORES

// IndexedDB predates promises: requests report back through callbacks.
// These two helpers wrap them so we can use await.
function done<T>(request: IDBRequest<T>): Promise<T> {
	return new Promise((resolve, reject) => {
		request.onsuccess = () => resolve(request.result)
		request.onerror = () => reject(request.error)
	})
}

function finished(tx: IDBTransaction): Promise<void> {
	return new Promise((resolve, reject) => {
		tx.oncomplete = () => resolve()
		tx.onerror = () => reject(tx.error)
		tx.onabort = () => reject(tx.error ?? new Error('IndexedDB transaction aborted'))
	})
}

export class IndexedDbStorage implements SyncStorage {
	private db: Promise<IDBDatabase>

	/** One database per user, so a shared tablet keeps each person's copy apart. */
	constructor(userId: number) {
		const request = indexedDB.open(`mandarin-shopping-${userId}`, VERSION)
		// Runs when the database is new (or VERSION goes up): create the stores.
		request.onupgradeneeded = () => {
			const db = request.result
			for (const [name, keyPath] of Object.entries(DATA_STORES)) {
				db.createObjectStore(name, {keyPath})
			}
			// autoIncrement keys keep the outbox in the order changes were made.
			db.createObjectStore('outbox', {autoIncrement: true})
			db.createObjectStore('meta')
			db.createObjectStore('photos')
		}
		this.db = done(request)
	}

	async load(): Promise<Stored> {
		const db = await this.db
		const tx = db.transaction([...Object.keys(DATA_STORES), 'outbox', 'meta'], 'readonly')
		const all = (name: DataStore) => done(tx.objectStore(name).getAll())
		const [lists, categories, catalog, items, conflicts, outbox, cursor] = await Promise.all([
			all('lists'), all('categories'), all('catalog'), all('items'), all('conflicts'),
			done(tx.objectStore('outbox').getAll()),
			done(tx.objectStore('meta').get('cursor')),
		])
		// Rebuilding through applyResult also re-links items to the catalogue.
		const data = applyResult(emptyData(), {
			cursor: 0, full: true, lists, categories, catalog, items, conflicts,
			deletedListIds: [], deletedCategoryIds: [], deletedCatalogIds: [], deletedItemUids: [], rejected: [],
		})
		return {data, outbox, cursor: typeof cursor === 'number' ? cursor : 0}
	}

	async addOp(op: SyncOp) {
		const db = await this.db
		const tx = db.transaction('outbox', 'readwrite')
		tx.objectStore('outbox').add(op)
		await finished(tx)
	}

	async saveResult(result: SyncResult, sentOpIds: string[]) {
		const db = await this.db
		const tx = db.transaction([...Object.keys(DATA_STORES), 'outbox', 'meta'], 'readwrite')
		const store = (name: DataStore) => tx.objectStore(name)
		if (result.full) {
			for (const name of Object.keys(DATA_STORES) as DataStore[]) store(name).clear()
		}
		// Vue's reactive proxies can't be stored, so rows are plain copies (structuredClone needs plain data).
		const put = (name: DataStore, rows: object[]) => rows.forEach(row => store(name).put(JSON.parse(JSON.stringify(row))))
		const remove = (name: DataStore, keys: (number | string)[]) => keys.forEach(key => store(name).delete(key))
		put('lists', result.lists)
		remove('lists', result.deletedListIds)
		put('categories', result.categories)
		remove('categories', result.deletedCategoryIds)
		put('catalog', result.catalog)
		remove('catalog', result.deletedCatalogIds)
		put('items', result.items)
		remove('items', result.deletedItemUids)
		put('conflicts', result.conflicts)
		tx.objectStore('meta').put(result.cursor, 'cursor')

		// Drop the answered ops. A cursor walks the outbox, deleting matches.
		const sent = new Set(sentOpIds)
		const cursorRequest = tx.objectStore('outbox').openCursor()
		cursorRequest.onsuccess = () => {
			const cursor = cursorRequest.result
			if (!cursor) return
			if (sent.has((cursor.value as SyncOp).id)) cursor.delete()
			cursor.continue()
		}
		await finished(tx)
	}

	async getPhoto(photoId: number) {
		const db = await this.db
		return done<Blob | undefined>(db.transaction('photos').objectStore('photos').get(photoId))
	}

	async putPhoto(photoId: number, blob: Blob) {
		const db = await this.db
		const tx = db.transaction('photos', 'readwrite')
		tx.objectStore('photos').put(blob, photoId)
		await finished(tx)
	}

	async prunePhotos(keep: Set<number>) {
		const db = await this.db
		const tx = db.transaction('photos', 'readwrite')
		const keys = await done(tx.objectStore('photos').getAllKeys())
		for (const key of keys) {
			if (!keep.has(key as number)) tx.objectStore('photos').delete(key)
		}
		await finished(tx)
	}
}
