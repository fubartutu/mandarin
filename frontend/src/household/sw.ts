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

import {
	isNotificationAction,
	runNotificationAction,
	type NotificationButton,
} from './push/notificationActions'
import {RELOGIN_PARAM} from './accessLogin'

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
			// The app reloads itself for an expired Cloudflare login
			// (accessLogin.ts): wait for the network, whose redirect to the
			// login page is the point, instead of the cached copy after 4 s.
			if (options.url.searchParams.has(RELOGIN_PARAM)) return await fetch(options.request)
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
// The server (pkg/household/push) sends {title, body, url, tag}, and for a
// single task also Done and Snooze buttons with the token they send back
// (push/notificationActions.ts). A push must always show a notification
// (Safari stops delivering otherwise). Safari shows no buttons; Android, a
// Galaxy Watch it forwards to, and desktop Chrome do.

interface PushMessage {
	title?: string
	body?: string
	url?: string
	tag?: string
	actions?: NotificationButton[]
	action_token?: string
}

// actions and renotify are in the Notification API but not in TypeScript's
// NotificationOptions yet.
type MandarinNotificationOptions = NotificationOptions & {
	actions?: NotificationButton[]
	renotify?: boolean
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
		const hasActions = Boolean(msg.actions?.length && msg.action_token)
		const options: MandarinNotificationOptions = {
			body: msg.body,
			tag: msg.tag,
			icon,
			data: {mandarinUrl: msg.url || '/', mandarinActionToken: msg.action_token},
			...(hasActions ? {
				actions: msg.actions,
				// A snoozed reminder reuses the task's tag: buzz again even if
				// the earlier one is still showing.
				renotify: Boolean(msg.tag),
			} : {}),
		}
		event.waitUntil(self.registration.showNotification(msg.title || 'Mandarin', options))
	})

	// Tapping a notification opens its page: in an open Mandarin window if
	// there is one, else a new one. Its buttons are handled below. (Vikunja's own notificationclick handler
	// in sw.ts only reacts to its "show-task" action, which ours don't have.)
	self.addEventListener('notificationclick', event => {
		const path = event.notification.data?.mandarinUrl
		if (typeof path !== 'string') return
		event.notification.close()

		// Done, Snooze or Skip (habits): done in the background, without opening
		// the app. A snooze is confirmed quietly ("Snoozed until 10:40"); if it
		// fails (no signal, Cloudflare login expired), a notification says so.
		// Tapping either opens the task (or the Habits page).
		const token = event.notification.data?.mandarinActionToken
		const action = event.action
		if (isNotificationAction(action) && typeof token === 'string') {
			const {title, tag} = event.notification
			event.waitUntil((async () => {
				const next = await runNotificationAction(self.registration.scope, action, token, title)
				if (next) {
					await self.registration.showNotification(next.title, {
						body: next.body,
						tag,
						icon: new URL('images/household/android-chrome-192x192.png', self.registration.scope).href,
						data: {mandarinUrl: path},
						silent: next.quiet,
					})
				}
			})())
			return
		}

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
