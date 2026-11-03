// The shopping sync engine: the shared offline engine (household/sync/engine.ts)
// with shopping's data and changes (model.ts), plus the conflict log. See
// SPEC.md, "Offline sync".

import {computed} from 'vue'

import {
	createSyncEngine as createEngine,
	isOfflineError,
	type SyncEngineOptions as Options,
	type SyncStatus,
	type Transport as TransportOf,
} from '../../sync/engine'
import {
	applyOp,
	applyResult,
	emptyData,
	type ConflictEntry,
	type ShoppingData,
	type SyncOp,
	type SyncResult,
} from './model'
import type {SyncStorage} from './storage'

export {isOfflineError, type SyncStatus}

export type Transport = TransportOf<SyncOp, SyncResult>
export type SyncEngineOptions = Options<SyncOp>

export const shoppingModel = {empty: emptyData, applyOp, applyResult}

export function createSyncEngine(storage: SyncStorage, transport: Transport, options: SyncEngineOptions = {}) {
	const engine = createEngine<ShoppingData, SyncOp, SyncResult, SyncStorage>(shoppingModel, storage, transport, options)

	const conflicts = computed<ConflictEntry[]>(() =>
		[...engine.data.value.conflicts.values()].sort((a, b) => b.id - a.id))

	return {...engine, conflicts}
}

export type SyncEngine = ReturnType<typeof createSyncEngine>
