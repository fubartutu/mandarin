// Cooking mode's place in a recipe, kept on this device so leaving (or the
// phone locking) and coming back within 6 hours carries on where it was.
// One localStorage key per recipe; nothing goes to the server.
//
// Storage can be missing or throw on any call (private browsing, a full
// quota, blocked site data). Every access here is wrapped, so the worst case
// is cooking mode starting fresh, never cooking mode breaking.

import type {CookingTimer} from './useCookingTimers'

export interface CookingState {
	stepIndex: number
	ticked: number[] // ingredient line indexes
	timers: CookingTimer[] // running: their endsAt keeps counting while away
	finished: CookingTimer[] // banner not yet dismissed
}

const VERSION = 1
export const RESUME_WITHIN_MS = 6 * 60 * 60 * 1000

interface SavedCooking extends CookingState {
	version: typeof VERSION
	savedAt: number // epoch ms
}

const PREFIX = 'mandarin-cooking-'
const key = (recipeId: number) => `${PREFIX}${recipeId}`

/** This device's localStorage, or null where the browser won't hand it over. */
export function deviceStorage(): Storage | null {
	try {
		return window.localStorage ?? null
	} catch {
		return null
	}
}

/** Runs fn against storage, or gives fallback when there's none or it throws. */
function using<T>(storage: Storage | null, fallback: T, fn: (storage: Storage) => T): T {
	if (!storage) return fallback
	try {
		return fn(storage)
	} catch {
		return fallback
	}
}

export function saveCooking(storage: Storage | null, recipeId: number, state: CookingState, now: number) {
	const saved: SavedCooking = {version: VERSION, savedAt: now, ...state}
	using(storage, undefined, s => s.setItem(key(recipeId), JSON.stringify(saved)))
}

export function clearCooking(storage: Storage | null, recipeId: number) {
	using(storage, undefined, s => s.removeItem(key(recipeId)))
}

const isNumber = (value: unknown): value is number => typeof value === 'number' && Number.isFinite(value)

function isTimer(value: unknown): value is CookingTimer {
	if (typeof value !== 'object' || value === null) return false
	const timer = value as Record<string, unknown>
	return typeof timer.id === 'string' && typeof timer.label === 'string'
		&& isNumber(timer.stepNumber) && isNumber(timer.seconds) && isNumber(timer.endsAt)
}

const isTimerList = (value: unknown): value is CookingTimer[] => Array.isArray(value) && value.every(isTimer)

/** The saved state, or null when it isn't one this version wrote. */
function parse(raw: string): SavedCooking | null {
	let value: unknown
	try {
		value = JSON.parse(raw)
	} catch {
		return null
	}
	if (typeof value !== 'object' || value === null) return null
	const saved = value as Record<string, unknown>
	const valid = saved.version === VERSION && isNumber(saved.savedAt)
		&& Number.isInteger(saved.stepIndex)
		&& Array.isArray(saved.ticked) && saved.ticked.every(Number.isInteger)
		&& isTimerList(saved.timers) && isTimerList(saved.finished)
	return valid ? saved as unknown as SavedCooking : null
}

const isFresh = (saved: SavedCooking, now: number) => now - saved.savedAt < RESUME_WITHIN_MS

/**
 * The state saved for this recipe if it's under 6 hours old. A stale or
 * unreadable one is deleted, so the next save starts clean.
 */
export function loadCooking(storage: Storage | null, recipeId: number, now: number): CookingState | null {
	return using(storage, null, s => {
		const raw = s.getItem(key(recipeId))
		if (raw === null) return null
		const saved = parse(raw)
		if (!saved || !isFresh(saved, now)) {
			s.removeItem(key(recipeId))
			return null
		}
		const {stepIndex, ticked, timers, finished} = saved
		return {stepIndex, ticked, timers, finished}
	})
}

/** Deletes every recipe's saved state that is stale or unreadable. */
export function forgetStaleCooking(storage: Storage | null, now: number) {
	using(storage, undefined, s => {
		// Collect first: removing while walking key(i) shifts the indexes.
		const keys: string[] = []
		for (let i = 0; i < s.length; i++) {
			const k = s.key(i)
			if (k?.startsWith(PREFIX)) keys.push(k)
		}
		for (const k of keys) {
			const raw = s.getItem(k)
			const saved = raw === null ? null : parse(raw)
			if (!saved || !isFresh(saved, now)) s.removeItem(k)
		}
	})
}

/** A saved state made to fit the recipe now: its method or ingredients may have been edited since. */
export function fitToRecipe(state: CookingState, recipe: {steps: number, ingredients: number}): CookingState {
	return {
		...state,
		stepIndex: Math.max(0, Math.min(state.stepIndex, recipe.steps - 1)),
		ticked: state.ticked.filter(index => index >= 0 && index < recipe.ingredients),
	}
}
