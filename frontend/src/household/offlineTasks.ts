// Tasks readable offline: the service worker keeps the server's answers (see
// sw.ts). Here, in the app: the kept answers are deleted on logout and when a
// different user signs in, so nobody reads someone else's tasks offline, and
// a banner says when what's on screen comes from that copy.

import {createApp} from 'vue'
import type {Router} from 'vue-router'

import OfflineBanner from './OfflineBanner.vue'

// Same name as API_CACHE in sw.ts (that file runs in the worker, so isn't imported here).
const API_CACHE = 'mandarin-api'
const USER_KEY = 'mandarin.apiCacheUser'

/** Deletes the kept server answers. Called from the auth store's logout (FORK). */
export async function clearOfflineApiCache(): Promise<void> {
	try {
		await globalThis.caches?.delete(API_CACHE)
	} catch {
		// No Cache Storage (plain http, private mode): nothing was kept.
	}
}

/**
 * Clears the kept answers when the signed-in user isn't the one they were
 * kept for. Logout clears them too, but a login can also happen after an
 * expired session, without one.
 */
export async function keepOfflineApiCacheFor(userId: number): Promise<void> {
	if (!userId) return
	const previous = localStorage.getItem(USER_KEY)
	if (previous !== String(userId)) {
		await clearOfflineApiCache()
		localStorage.setItem(USER_KEY, String(userId))
	}
}

/** Shows the offline banner on every page; it hides itself while online. */
export function mountOfflineBanner(router: Router): void {
	const el = document.createElement('div')
	document.body.appendChild(el)
	createApp(OfflineBanner, {router}).mount(el)
}
