// Connects the shared sync engine to habits: one engine per signed-in user,
// the real server and IndexedDB, and the moments a sync should happen
// (signal returning, the app coming back to the screen, an edit, a new day).
// Like useShoppingSync.ts, but habits have no websocket event: a poll and
// those moments keep the household's ticks in step.

import {computed, effectScope, ref, watch} from 'vue'
import {useEventListener, useIntervalFn, useNow} from '@vueuse/core'

import {habitsSync} from '@/client/generated'
import {i18n} from '@/i18n'
import {error} from '@/message'
import {useAuthStore} from '@/stores/auth'

import {dayKey} from '../../menu/week'
import {createSyncEngine} from '../../sync/engine'
import {DocumentStorage} from '../../sync/idb'
import {HABITS_CHANGED_EVENT} from '../queries'
import {habitsModel, normalizeSyncResult, rollToDay, type HabitOp, type HabitsData, type HabitsResult} from './model'

// Others' ticks on household routines show within this, or sooner.
const POLL_MS = 2 * 60_000

function startEngine(userId: number) {
	const habitTitles = new Map<number, string>()
	const engine = createSyncEngine(
		habitsModel,
		new DocumentStorage<HabitsData, HabitOp, HabitsResult>('habits', userId, habitsModel),
		async ({ops}, signal) => normalizeSyncResult((await habitsSync({body: {ops}, signal})).data),
		{
			onRejected: rejected => error({
				message: i18n.global.t('household.habits.sync.rejected', {
					names: [...new Set(rejected.map(r => habitTitles.get(r.op.habit_id) ?? ''))].filter(Boolean).join(', '),
				}, rejected.length),
			}),
		},
	)

	// The phone's calendar day, set by the clock below.
	const today = ref(dayKey(new Date()))

	// A detached effect scope owns the listeners, so they live as long as
	// the engine rather than as long as whichever component started it.
	const triggers = effectScope(true)
	triggers.run(() => {
		// Re-checked every minute, so the day moves on at midnight with the app open.
		const now = useNow({interval: 60_000})
		watch(now, value => {
			const day = dayKey(value)
			if (day !== today.value) {
				today.value = day
				engine.requestSync()
			}
		})
		// Rejections name the habit, so keep its title at hand.
		watch(engine.data, data => data.habits.forEach(h => habitTitles.set(h.id, h.title)), {immediate: true})
		useEventListener(window, 'online', () => engine.requestSync())
		useEventListener(window, HABITS_CHANGED_EVENT, () => engine.requestSync())
		useEventListener(document, 'visibilitychange', () => {
			if (document.visibilityState === 'visible') engine.requestSync()
		})
		useIntervalFn(() => {
			if (document.visibilityState === 'visible') engine.requestSync()
		}, POLL_MS)
		// Logging out (or in as someone else) ends this user's sync.
		watch(() => useAuthStore().info?.id, id => {
			if (id !== userId) stopEngine()
		})
	})
	engine.requestSync()
	return {engine, triggers, today}
}

let running: ReturnType<typeof startEngine> | null = null
let runningUserId = 0

function stopEngine() {
	running?.engine.stop()
	running?.triggers.stop()
	running = null
	runningUserId = 0
}

function habitsEngine() {
	const userId = useAuthStore().info?.id ?? 0
	if (!running || runningUserId !== userId) {
		stopEngine()
		running = startEngine(userId)
		runningUserId = userId
	}
	return running
}

/**
 * Today's habits for the signed-in user, kept on the phone and synced. Call it
 * from a component's setup; the first call starts the engine. `habits` is
 * moved on to the phone's current day, so a page left open overnight (or
 * opened offline the next morning) starts the day empty.
 */
export function useHabitsSync() {
	const {engine, today} = habitsEngine()
	return {
		...engine,
		today: computed(() => today.value),
		habits: computed(() => engine.data.value.habits.map(h => rollToDay(h, today.value))),
		groups: computed(() => engine.data.value.groups),
		hasData: computed(() => engine.data.value.loaded),
		online: computed(() => engine.status.value !== 'offline'),
	}
}

export type HabitsSync = ReturnType<typeof useHabitsSync>
