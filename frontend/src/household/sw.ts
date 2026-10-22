// Runs inside the service worker (imported from src/sw.ts, marked FORK).
// Vikunja's worker caches the app's files but not its pages, so opening
// /shopping without signal failed. Page loads now try the network for a few
// seconds, then fall back to the cached app, which opens the phone's copy.
//
// Tasks are readable offline the same way: every answer the server gives
// (projects, tasks, labels, settings…) is kept, and served when there's no
// connection. So anything opened before can be read without signal.

import {NavigationRoute, registerRoute} from 'workbox-routing'
import {NetworkFirst} from 'workbox-strategies'
import type {WorkboxPlugin} from 'workbox-core'

declare let self: ServiceWorkerGlobalScope

/** The cache of server answers. The app deletes it on logout (offlineTasks.ts). */
export const API_CACHE = 'mandarin-api'
const API_CACHE_MAX_ENTRIES = 500

export function registerOfflinePages() {
	// NetworkFirst: online, the server's page (it carries live config); after
	// networkTimeoutSeconds, or with no network, the copy cached last time.
	const pages = new NetworkFirst({cacheName: 'mandarin-pages', networkTimeoutSeconds: 4})

	registerRoute(new NavigationRoute(async options => {
		try {
			return await pages.handle(options)
		} catch {
			// A page never opened before: every page of the app is the same
			// index.html, which Vikunja's worker precaches (stored with a
			// ?__WB_REVISION__=… suffix, hence ignoreSearch).
			const index = new URL('index.html', self.registration.scope).href
			return (await caches.match(index, {ignoreSearch: true})) ?? Response.error()
		}
	}, {
		// Server-rendered or API URLs are never the app.
		denylist: [/\/api\//, /\/dav\//, /\/\.well-known\//],
	}))
}

// Not worth keeping offline, or must never come from a cache: logins and
// tokens, the shopping sync (it has its own offline copy), and pictures and
// files (only JSON is kept anyway; see keepJsonOnly).
const NOT_CACHED = [/\/token\b/, /\/login\b/, /\/auth\//, /\/logout\b/, /\/shopping\/sync\b/, /\/websocket\b/]

const keepJsonOnly: WorkboxPlugin = {
	// Only successful JSON answers are kept: not errors, not file downloads.
	cacheWillUpdate: async ({response}) =>
		response.ok && response.headers.get('content-type')?.includes('json') ? response : null,
	// Keep the cache from growing forever: drop the oldest entries.
	cacheDidUpdate: async ({cacheName}) => {
		const cache = await caches.open(cacheName)
		const keys = await cache.keys() // oldest first
		await Promise.all(keys.slice(0, Math.max(0, keys.length - API_CACHE_MAX_ENTRIES)).map(key => cache.delete(key)))
	},
}

export function registerOfflineApi() {
	// NetworkFirst: always the server's answer when it comes; the kept copy
	// only without a connection, or when the server takes longer than
	// networkTimeoutSeconds (a dead zone), so stale data is rare.
	const api = new NetworkFirst({
		cacheName: API_CACHE,
		networkTimeoutSeconds: 8,
		plugins: [keepJsonOnly],
		fetchOptions: {cache: 'no-store'}, // like Vikunja's own API route: bypass the browser's HTTP cache
	})
	// Registered before Vikunja's NetworkOnly route for /api/v1, so it wins.
	registerRoute(
		({url, request}) => url.origin === self.location.origin &&
			/\/api\/v[12]\//.test(url.pathname) &&
			!NOT_CACHED.some(pattern => pattern.test(url.pathname)) &&
			!request.headers.get('accept')?.includes('text/event-stream'),
		api,
		'GET',
	)
}
