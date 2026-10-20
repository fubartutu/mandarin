// The sync engine: keeps the displayed data, the outbox and the cursor, and
// talks to the server. Every change is shown and saved on the phone first,
// then sent when there's signal. See SPEC.md, "Offline sync".

import {computed, ref, shallowRef, readonly} from 'vue'

import {ApiError} from '@/client/problem'
import {
	applyOp,
	applyResult,
	emptyData,
	rebase,
	type ConflictEntry,
	type ShoppingData,
	type SyncOp,
	type SyncResult,
} from './model'
import type {SyncStorage} from './storage'

/** Sends ops and the cursor to the server, returns its answer. */
export type Transport = (body: {cursor: number, ops: SyncOp[]}, signal: AbortSignal) => Promise<SyncResult>

export type SyncStatus = 'loading' | 'syncing' | 'synced' | 'offline' | 'error'

export interface SyncEngineOptions {
	/** Called with ops the server refused (e.g. the list was deleted meanwhile). */
	onRejected?: (rejected: {op: SyncOp, reason: string}[]) => void
	/** How long one sync may take before it counts as "no signal". */
	timeoutMs?: number
}

// Ops per request; the server accepts up to 500.
const BATCH = 200
const RETRY_FIRST_MS = 2_000
const RETRY_MAX_MS = 60_000

/**
 * No signal, a dead Wi-Fi, or the server/proxy down: wait and retry quietly.
 * Anything else (a 4xx, a 500 from a bug) is shown as an error, still retried.
 */
export function isOfflineError(e: unknown): boolean {
	if (e instanceof ApiError) {
		return e.status === 0 || e.status === 502 || e.status === 503 || e.status === 504
	}
	return true // fetch throws a TypeError without a connection; timeouts throw too
}

export function createSyncEngine(storage: SyncStorage, transport: Transport, options: SyncEngineOptions = {}) {
	// shallowRef: the screen updates when we assign a new ShoppingData, and
	// Vue doesn't wrap every item in a reactive proxy (we never mutate them).
	const displayed = shallowRef<ShoppingData>(emptyData())
	const status = ref<SyncStatus>('loading')
	const pendingCount = ref(0)
	const lastSyncedAt = ref<number | null>(null)
	const lastError = ref('')

	let confirmed = emptyData()
	let outbox: SyncOp[] = []
	let cursor = 0
	// Outbox writes are chained so they land in IndexedDB in order, and a
	// sync waits for them: the ops it sends are always saved first.
	let writes: Promise<void> = Promise.resolve()

	let running: Promise<void> | null = null
	let again = false
	let retryTimer: ReturnType<typeof setTimeout> | undefined
	let retryDelay = RETRY_FIRST_MS
	let debounceTimer: ReturnType<typeof setTimeout> | undefined
	let stopped = false

	const loaded = storage.load().then(stored => {
		confirmed = stored.data
		outbox = stored.outbox
		cursor = stored.cursor
		pendingCount.value = outbox.length
		displayed.value = rebase(confirmed, outbox)
	})

	/** Records a change: on screen now, saved now, sent soon. */
	async function change(op: SyncOp) {
		await loaded
		outbox.push(op)
		pendingCount.value = outbox.length
		displayed.value = applyOp(displayed.value, op)
		writes = writes.then(() => storage.addOp(op))
		requestSync(300)
		await writes
	}

	/** Syncs after delayMs, folding several quick requests into one. */
	function requestSync(delayMs = 0) {
		clearTimeout(debounceTimer)
		debounceTimer = setTimeout(() => void sync(), delayMs)
	}

	function scheduleRetry() {
		clearTimeout(retryTimer)
		retryTimer = setTimeout(() => void sync(), retryDelay)
		retryDelay = Math.min(retryDelay * 2, RETRY_MAX_MS)
	}

	async function syncOnce(): Promise<boolean> {
		await loaded
		await writes
		const batch = outbox.slice(0, BATCH)
		status.value = 'syncing'

		let result: SyncResult
		try {
			result = await transport({cursor, ops: batch}, AbortSignal.timeout(options.timeoutMs ?? 20_000))
		} catch (e) {
			status.value = isOfflineError(e) ? 'offline' : 'error'
			lastError.value = e instanceof Error ? e.message : String(e)
			if (!stopped) scheduleRetry()
			return false
		}

		const sentIds = batch.map(op => op.id)
		await storage.saveResult(result, sentIds)
		const sent = new Set(sentIds)
		confirmed = applyResult(confirmed, result)
		cursor = result.cursor
		outbox = outbox.filter(op => !sent.has(op.id))
		pendingCount.value = outbox.length
		displayed.value = rebase(confirmed, outbox)

		status.value = 'synced'
		lastSyncedAt.value = Date.now()
		lastError.value = ''
		retryDelay = RETRY_FIRST_MS
		clearTimeout(retryTimer)

		if (result.rejected.length > 0 && options.onRejected) {
			const byId = new Map(batch.map(op => [op.id, op]))
			options.onRejected(result.rejected.flatMap(r => {
				const op = byId.get(r.opId)
				return op ? [{op, reason: r.reason}] : []
			}))
		}
		return true
	}

	/**
	 * Sends what's waiting and fetches what changed. Only one sync runs at a
	 * time; asking during one queues one more round after it.
	 */
	function sync(): Promise<void> {
		if (stopped) return Promise.resolve()
		if (running) {
			again = true
			return running
		}
		running = (async () => {
			try {
				let ok: boolean
				do {
					again = false
					ok = await syncOnce()
				} while (ok && !stopped && (again || outbox.length > 0))
			} finally {
				running = null
			}
		})()
		return running
	}

	function stop() {
		stopped = true
		clearTimeout(retryTimer)
		clearTimeout(debounceTimer)
	}

	const conflicts = computed<ConflictEntry[]>(() =>
		[...displayed.value.conflicts.values()].sort((a, b) => b.id - a.id))

	return {
		// A computed hands out the data as is; readonly() would wrap every Map in a proxy.
		data: computed(() => displayed.value),
		status: readonly(status),
		pendingCount: readonly(pendingCount),
		lastSyncedAt: readonly(lastSyncedAt),
		lastError: readonly(lastError),
		conflicts,
		loaded,
		storage,
		change,
		sync,
		requestSync,
		stop,
	}
}

export type SyncEngine = ReturnType<typeof createSyncEngine>
