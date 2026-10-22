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

// ---- Push notifications ----
// The server (pkg/household/push) sends {title, body, url, tag}. A push must
// always show a notification (Safari stops delivering otherwise).

interface PushMessage {
	title?: string
	body?: string
	url?: string
	tag?: string
}

export function registerPush() {
	self.addEventListener('push', event => {
		let msg: PushMessage
		try {
			msg = event.data?.json() ?? {}
		} catch {
			msg = {body: event.data?.text()}
		}
		const icon = new URL('images/household/android-chrome-192x192.png', self.registration.scope).href
		event.waitUntil(self.registration.showNotification(msg.title || 'Mandarin', {
			body: msg.body,
			tag: msg.tag,
			icon,
			data: {mandarinUrl: msg.url || '/'},
		}))
	})

	// Tapping a notification opens its page: in an open Mandarin window if
	// there is one, else a new one. (Vikunja's own notificationclick handler
	// in sw.ts only reacts to its "show-task" action, which ours don't have.)
	self.addEventListener('notificationclick', event => {
		const path = event.notification.data?.mandarinUrl
		if (typeof path !== 'string') return
		event.notification.close()
		const target = new URL(path.replace(/^\//, ''), self.registration.scope).href
		event.waitUntil((async () => {
			const windows = await self.clients.matchAll({type: 'window', includeUncontrolled: true})
			const open = windows.find(w => new URL(w.url).origin === self.location.origin)
			if (open) {
				await open.focus()
				await open.navigate(target).catch(() => self.clients.openWindow(target))
				return
			}
			await self.clients.openWindow(target)
		})())
	})
}
