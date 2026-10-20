// Runs inside the service worker (imported from src/sw.ts, marked FORK).
// Vikunja's worker caches the app's files but not its pages, so opening
// /shopping without signal failed. Page loads now try the network for a few
// seconds, then fall back to the cached app, which opens the phone's copy.

import {NavigationRoute, registerRoute} from 'workbox-routing'
import {NetworkFirst} from 'workbox-strategies'

declare let self: ServiceWorkerGlobalScope

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
