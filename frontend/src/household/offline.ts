// Starting the app with no signal. Vikunja expects the server at start-up:
// it fetches /info, renews an expired login and loads the user, and shows a
// full-screen "offline" page without a connection. The shopping list has to
// open in a dead zone, so a few marked hooks (FORK) in Vikunja's start-up
// code call these helpers instead.
//
// Nothing here weakens the login: offline, the app only shows the copy
// already on the phone, and every request to the server still needs a valid
// token (which is renewed as soon as there's signal).

import {START_LOCATION, type Router} from 'vue-router'

import {getToken, RefreshTokenError, type RefreshFailure} from '@/helpers/auth'

// How long a start-up request may take on a weak signal before the app gives
// up on it and opens from the phone's copy. The request carries on in the
// background, so a slow server still gets through.
const START_TIMEOUT_MS = 4_000

let startedOffline = false

/** A start-up request that took too long; treated like no signal. */
export class StartTimeoutError extends Error {
	constructor() {
		super('Server not reachable')
		this.name = 'StartTimeoutError'
	}
}

/** A start-up check couldn't reach the server. */
export function markStartedOffline() {
	startedOffline = true
}

/** Resolves or rejects like promise, or rejects with timeoutError() after a few seconds. */
export function withinStartTimeout<T>(promise: Promise<T>, timeoutError: () => Error = () => new StartTimeoutError()): Promise<T> {
	return new Promise<T>((resolve, reject) => {
		const timer = setTimeout(() => {
			markStartedOffline()
			reject(timeoutError())
		}, START_TIMEOUT_MS)
		promise.then(
			value => {
				clearTimeout(timer)
				resolve(value)
			},
			reason => {
				clearTimeout(timer)
				reject(reason)
			},
		)
	})
}

/** A token renewal that gives up like a network failure on a weak signal. */
export function renewWithinStartTimeout<T>(promise: Promise<T>): Promise<T> {
	return withinStartTimeout(promise, () => new RefreshTokenError({kind: 'network'}, 'timed out'))
}

/** Someone has logged in on this device, so there's a copy to open offline. */
export function canStartOffline(): boolean {
	return getToken() !== null
}

/**
 * The login token expired and couldn't be renewed for lack of signal: keep
 * the session (offline, it only shows the phone's copy) instead of logging out.
 */
export function keepSessionOffline(failure: RefreshFailure | undefined): boolean {
	if (failure?.kind !== 'network') return false
	markStartedOffline()
	return true
}

/** The server couldn't be reached at all (as opposed to answering with an error). */
export function isUnreachable(e: unknown): boolean {
	const status = (e as {status?: unknown} | null)?.status
	const unreachable = typeof status === 'number'
		// The proxy answered for a server that's down.
		? status === 0 || status === 502 || status === 503 || status === 504
		// fetch throws a TypeError without a connection; aborts and our timeout are DOMException/StartTimeoutError.
		: e instanceof TypeError || e instanceof StartTimeoutError || (e instanceof DOMException && ['AbortError', 'TimeoutError'].includes(e.name))
	if (unreachable) markStartedOffline()
	return unreachable
}

/**
 * Opening the app on its home page without signal goes to the shopping list
 * instead: the task overview needs the server, the list doesn't.
 */
export function registerOfflineStart(router: Router) {
	router.beforeEach((to, from) => {
		if (from === START_LOCATION && to.name === 'home' && startedOffline) {
			return {name: 'shopping'}
		}
	})
}

/**
 * On the shopping list, "couldn't reach the server" errors from Vikunja's own
 * background requests (projects, notifications) aren't toasted: the sync
 * badge already says the app is offline. Elsewhere they show as usual.
 */
export function isQuietOfflineError(e: unknown): boolean {
	return window.location.pathname.includes('/shopping') && isUnreachable(e)
}
