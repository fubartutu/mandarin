// Connects the sync engine to the app: one engine per signed-in user, the
// real server and IndexedDB, and the moments a sync should happen (a change
// from another phone, signal returning, the app coming back to the screen).

import {computed, effectScope, onScopeDispose, watch} from 'vue'
import {useEventListener, useIntervalFn} from '@vueuse/core'

import {shoppingSync} from '@/client/generated'
import {useWebSocket} from '@/composables/useWebSocket'
import {i18n} from '@/i18n'
import {error} from '@/message'
import {useAuthStore} from '@/stores/auth'
import {createSyncEngine, type SyncEngine} from './engine'
import {normalizeSyncResult} from './model'
import {IndexedDbStorage} from './storage'

// Matches EventHouseholdShoppingChanged in pkg/websocket/household.go.
const CHANGED_EVENT = 'household.shopping.changed'
// A safety net in case a websocket message is missed.
const POLL_MS = 2 * 60_000

let engine: SyncEngine | null = null
let engineUserId = 0
let triggers: ReturnType<typeof effectScope> | null = null

function startEngine(userId: number): SyncEngine {
	const created = createSyncEngine(
		new IndexedDbStorage(userId),
		async ({cursor, ops}, signal) =>
			normalizeSyncResult((await shoppingSync({body: {cursor, ops}, signal})).data),
		{
			onRejected: rejected => error({
				message: i18n.global.t('household.shopping.sync.rejected', {
					names: [...new Set(rejected.map(r => r.op.item_name).filter(Boolean))].join(', '),
				}, rejected.length),
			}),
		},
	)

	// A detached effect scope owns the listeners, so they live as long as
	// the engine rather than as long as whichever component started it.
	triggers = effectScope(true)
	triggers.run(() => {
		const socket = useWebSocket()
		onScopeDispose(socket.subscribe(CHANGED_EVENT, () => created.requestSync(300)))
		watch(socket.authenticated, authenticated => {
			if (authenticated) created.requestSync() // reconnected: we may have missed changes
		})
		useEventListener(window, 'online', () => created.requestSync())
		useEventListener(document, 'visibilitychange', () => {
			if (document.visibilityState === 'visible') created.requestSync()
		})
		useIntervalFn(() => {
			if (document.visibilityState === 'visible') created.requestSync()
		}, POLL_MS)
		// Logging out (or in as someone else) ends this user's sync.
		watch(() => useAuthStore().info?.id, id => {
			if (id !== userId) stopEngine()
		})
	})
	created.requestSync()
	return created
}

function stopEngine() {
	engine?.stop()
	triggers?.stop()
	engine = null
	triggers = null
	engineUserId = 0
}

/**
 * The shopping sync engine for the signed-in user. Call it from a
 * component's setup; the first call starts it.
 */
export function useShoppingSync(): SyncEngine {
	const auth = useAuthStore()
	const userId = auth.info?.id ?? 0
	if (!engine || engineUserId !== userId) {
		stopEngine()
		engine = startEngine(userId)
		engineUserId = userId
	}
	return engine
}

/**
 * Syncs now, e.g. after a list was renamed online, so the change shows.
 * Resolves straight away before the engine has started.
 */
export function syncShopping(): Promise<void> {
	return engine?.sync() ?? Promise.resolve()
}

/** Stops syncing, e.g. on logout. */
export function stopShoppingSync() {
	stopEngine()
}

/** False while the server can't be reached: things that need it are disabled. */
export function useShoppingOnline() {
	const sync = useShoppingSync()
	return computed(() => sync.status.value !== 'offline')
}
