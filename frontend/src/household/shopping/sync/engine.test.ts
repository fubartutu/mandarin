import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

import {ApiError} from '@/client/problem'
import {normalizeItem, normalizeList} from '../queries'
import {createSyncEngine, isOfflineError, type Transport} from './engine'
import {createOp, updateOp, type SyncOp, type SyncResult} from './model'
import {MemoryStorage} from './storage'

function answer(extra: Partial<SyncResult> = {}): SyncResult {
	return {
		cursor: 5, full: false, lists: [], deletedListIds: [], categories: [], deletedCategoryIds: [],
		catalog: [], deletedCatalogIds: [], items: [], deletedItemUids: [], conflicts: [], rejected: [],
		...extra,
	}
}

// A pretend server: remembers what it was sent and answers by echoing the
// created items back, the way the real one includes your own changes.
function fakeServer() {
	const sent: {cursor: number, ops: SyncOp[]}[] = []
	let offline = false
	const transport = vi.fn<Transport>(async body => {
		if (offline) throw new TypeError('Failed to fetch')
		sent.push(body)
		const items = body.ops.filter(op => op.type === 'item.create').map((op, i) =>
			normalizeItem({id: 100 + i, uid: op.item_uid, list_id: op.list_id, name: op.set?.name}))
		return answer({cursor: body.cursor + 1, items, lists: [normalizeList({id: 1, title: 'Groceries'})]})
	})
	return {sent, transport, setOffline: (value: boolean) => { offline = value }}
}

const milk = {list_id: 1, name: 'Milk', quantity: 0, unit: '', category_id: 0}

describe('sync engine', () => {
	beforeEach(() => {
		vi.useFakeTimers()
	})
	afterEach(() => {
		vi.useRealTimers()
	})

	it('shows a change at once, then sends it', async () => {
		const server = fakeServer()
		const storage = new MemoryStorage()
		const engine = createSyncEngine(storage, server.transport)
		const op = createOp(milk)

		const changed = engine.change(op)
		await engine.loaded
		await Promise.resolve()
		expect(engine.data.value.items.get(op.item_uid)?.name).toBe('Milk')
		await changed
		expect(engine.pendingCount.value).toBe(1)
		expect(storage.outbox).toHaveLength(1)

		await vi.advanceTimersByTimeAsync(300)
		await engine.sync()
		expect(server.sent[0].ops.map(o => o.id)).toEqual([op.id])
		expect(engine.pendingCount.value).toBe(0)
		expect(storage.outbox).toHaveLength(0)
		expect(engine.data.value.items.get(op.item_uid)?.id).toBe(100)
		expect(engine.status.value).toBe('synced')
		engine.stop()
	})

	it('keeps changes while offline and sends them when the signal is back', async () => {
		const server = fakeServer()
		server.setOffline(true)
		const engine = createSyncEngine(new MemoryStorage(), server.transport)
		const op = createOp(milk)
		await engine.change(op)
		await engine.sync()
		expect(engine.status.value).toBe('offline')
		expect(engine.pendingCount.value).toBe(1)
		expect(engine.data.value.items.get(op.item_uid)).toBeDefined()

		server.setOffline(false)
		await vi.advanceTimersByTimeAsync(2_000) // the first retry
		expect(engine.status.value).toBe('synced')
		expect(engine.pendingCount.value).toBe(0)
		engine.stop()
	})

	it("doesn't lose a change made while a sync is on its way", async () => {
		const storage = new MemoryStorage()
		let release!: () => void
		const sent: SyncOp[][] = []
		// The server's copy, so its answers include the ops it applied.
		const onServer = new Map<string, ReturnType<typeof normalizeItem>>()
		const transport: Transport = async body => {
			sent.push(body.ops)
			if (sent.length === 1) await new Promise<void>(resolve => { release = resolve })
			for (const op of body.ops) {
				const current = onServer.get(op.item_uid) ?? normalizeItem({id: 1, uid: op.item_uid, list_id: 1})
				onServer.set(op.item_uid, {...current, ...op.set})
			}
			return answer({items: body.ops.map(op => onServer.get(op.item_uid)!)})
		}
		const engine = createSyncEngine(storage, transport)
		const add = createOp(milk)
		await engine.change(add)
		const first = engine.sync()
		await vi.advanceTimersByTimeAsync(0)

		// Ticked while the add is still being sent.
		const item = engine.data.value.items.get(add.item_uid)!
		await engine.change(updateOp(item, {done: true})!)
		expect(engine.data.value.items.get(add.item_uid)?.done).toBe(true)

		release()
		await first
		// The answer to the add didn't include the tick, but it's still shown, and sent next.
		expect(engine.data.value.items.get(add.item_uid)?.done).toBe(true)
		expect(sent).toHaveLength(2)
		expect(sent[1].map(op => op.type)).toEqual(['item.update'])
		engine.stop()
	})

	it('carries on from what was stored when the app opens again', async () => {
		const storage = new MemoryStorage()
		const op = createOp(milk)
		await storage.addOp(op)
		const engine = createSyncEngine(storage, fakeServer().transport)
		await engine.loaded
		expect(engine.pendingCount.value).toBe(1)
		expect(engine.data.value.items.get(op.item_uid)?.name).toBe('Milk')
		engine.stop()
	})

	it('reports rejected changes', async () => {
		const onRejected = vi.fn()
		const op = createOp(milk)
		const transport: Transport = async () => answer({rejected: [{opId: op.id, reason: 'list gone'}]})
		const engine = createSyncEngine(new MemoryStorage(), transport, {onRejected})
		await engine.change(op)
		await engine.sync()
		expect(onRejected).toHaveBeenCalledWith([{op, reason: 'list gone'}])
		expect(engine.pendingCount.value).toBe(0)
		engine.stop()
	})
})

describe('isOfflineError', () => {
	it('counts no connection and a down server as offline, not other errors', () => {
		expect(isOfflineError(new TypeError('Failed to fetch'))).toBe(true)
		expect(isOfflineError(new ApiError({status: 502, detail: 'Bad gateway'}))).toBe(true)
		expect(isOfflineError(new ApiError({status: 500, detail: 'boom'}))).toBe(false)
		expect(isOfflineError(new ApiError({status: 403, detail: 'no'}))).toBe(false)
	})
})
