// IndexedDB (the browser's database, which survives closing the app and works
// offline) for the shared sync engine: small promise helpers, and a storage
// for modules whose whole copy fits in one record (habits). Shopping keeps
// its rows in stores of their own (shopping/sync/storage.ts).

import type {Stored, SyncStorage} from './engine'

/** A request's result as a promise. */
export function done<T>(request: IDBRequest<T>): Promise<T> {
	return new Promise((resolve, reject) => {
		request.onsuccess = () => resolve(request.result)
		request.onerror = () => reject(request.error)
	})
}

/** Resolves when a transaction has been written. */
export function finished(tx: IDBTransaction): Promise<void> {
	return new Promise((resolve, reject) => {
		tx.oncomplete = () => resolve()
		tx.onerror = () => reject(tx.error)
		tx.onabort = () => reject(tx.error ?? new Error('IndexedDB transaction aborted'))
	})
}

const VERSION = 1

/**
 * The whole confirmed copy as one record, plus the outbox. For modules whose
 * server answer is always the whole picture (not changes since a cursor), so
 * a saved answer replaces the copy. One database per module and user
 * (`mandarin-<name>-<userId>`), so a shared tablet keeps each person's copy apart.
 */
export class DocumentStorage<Data, Op extends {id: string}, Result extends {cursor: number}> implements SyncStorage<Data, Op, Result> {
	private db: Promise<IDBDatabase>

	constructor(
		name: string,
		userId: number,
		private model: {empty(): Data, applyResult(data: Data, result: Result): Data},
	) {
		const request = indexedDB.open(`mandarin-${name}-${userId}`, VERSION)
		// Runs when the database is new (or VERSION goes up): create the stores.
		request.onupgradeneeded = () => {
			request.result.createObjectStore('state')
			// autoIncrement keys keep the outbox in the order changes were made.
			request.result.createObjectStore('outbox', {autoIncrement: true})
		}
		this.db = done(request)
	}

	async load(): Promise<Stored<Data, Op>> {
		const db = await this.db
		const tx = db.transaction(['state', 'outbox'], 'readonly')
		const [data, cursor, outbox] = await Promise.all([
			done(tx.objectStore('state').get('data')),
			done(tx.objectStore('state').get('cursor')),
			done(tx.objectStore('outbox').getAll()),
		])
		return {
			data: (data as Data | undefined) ?? this.model.empty(),
			outbox: outbox as Op[],
			cursor: typeof cursor === 'number' ? cursor : 0,
		}
	}

	async addOp(op: Op) {
		const db = await this.db
		const tx = db.transaction('outbox', 'readwrite')
		tx.objectStore('outbox').add(op)
		await finished(tx)
	}

	async saveResult(result: Result, sentOpIds: string[]) {
		const db = await this.db
		const tx = db.transaction(['state', 'outbox'], 'readwrite')
		const state = tx.objectStore('state')
		state.put(this.model.applyResult(this.model.empty(), result), 'data')
		state.put(result.cursor, 'cursor')

		// Drop the answered ops. A cursor walks the outbox, deleting matches.
		const sent = new Set(sentOpIds)
		const walk = tx.objectStore('outbox').openCursor()
		walk.onsuccess = () => {
			const cursor = walk.result
			if (!cursor) return
			if (sent.has((cursor.value as Op).id)) cursor.delete()
			cursor.continue()
		}
		await finished(tx)
	}
}

/** The same in memory, for tests (jsdom has no IndexedDB). */
export class MemoryDocumentStorage<Data, Op extends {id: string}, Result extends {cursor: number}> implements SyncStorage<Data, Op, Result> {
	data: Data
	outbox: Op[] = []
	cursor = 0

	constructor(private model: {empty(): Data, applyResult(data: Data, result: Result): Data}) {
		this.data = model.empty()
	}

	async load(): Promise<Stored<Data, Op>> {
		return {data: this.data, outbox: [...this.outbox], cursor: this.cursor}
	}
	async addOp(op: Op) {
		this.outbox.push(op)
	}
	async saveResult(result: Result, sentOpIds: string[]) {
		this.data = this.model.applyResult(this.data, result)
		this.cursor = result.cursor
		const sent = new Set(sentOpIds)
		this.outbox = this.outbox.filter(op => !sent.has(op.id))
	}
}
