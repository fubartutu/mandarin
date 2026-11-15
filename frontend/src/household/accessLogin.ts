// Mandarin sits behind Cloudflare Zero Trust (Access). When its login cookie
// expires, Cloudflare answers every request with a redirect to its login page.
// A page load follows that redirect, but the app's own requests can't (it's
// another site), so they fail like "no signal" and the app shows the phone's
// copy as if offline. On iPhone a home-screen app is almost never reloaded:
// it's resumed where it was, so it stayed "offline" until it was killed.
//
// So when the app opens, comes back to the front, or the phone gets signal,
// it asks the server for a page without following redirects. A redirect means
// the Cloudflare login expired, and the app reloads itself so Cloudflare can
// show its login (and sends you back here after).

import {getFullBaseUrl} from '@/helpers/getFullBaseUrl'

/** Added to the reload's URL so the service worker skips its cached copy (sw.ts). */
export const RELOGIN_PARAM = 'mandarin-relogin'

// Don't ask more often than this (a resume and an "online" often come together).
const CHECK_EVERY_MS = 30_000
// After a reload for the login, don't reload again for a while: if the login
// page didn't come up, a loop would make the app unusable.
const RELOGIN_PAUSE_MS = 2 * 60_000
const RELOGIN_AT_KEY = 'mandarin.reloginAt'

/**
 * True when Cloudflare redirects instead of letting the request through.
 * No signal, or any answer from Mandarin itself, is false.
 */
export async function accessLoginExpired(fetchFn: typeof fetch = fetch, base: string = getFullBaseUrl()): Promise<boolean> {
	try {
		// The query keeps the service worker out of it (no route or precached
		// file matches), so the answer comes straight from the network.
		const res = await fetchFn(`${base}?mandarin-access-check=${Date.now()}`, {
			redirect: 'manual', // a redirect comes back as type "opaqueredirect" instead of being followed
			cache: 'no-store',
			credentials: 'same-origin', // the Cloudflare Access cookie
		})
		return res.type === 'opaqueredirect'
	} catch {
		return false
	}
}

/** The current page's URL, marked so the reload goes to the network. */
export function reloginUrl(href: string): string {
	const url = new URL(href)
	url.searchParams.set(RELOGIN_PARAM, '1')
	return url.href
}

/** The URL without the mark, or null when it has none. */
export function withoutReloginParam(href: string): string | null {
	const url = new URL(href)
	if (!url.searchParams.has(RELOGIN_PARAM)) return null
	url.searchParams.delete(RELOGIN_PARAM)
	return url.href
}

function readNumber(key: string): number {
	try {
		return Number(sessionStorage.getItem(key)) || 0
	} catch {
		return 0
	}
}

function writeNumber(key: string, value: number) {
	try {
		sessionStorage.setItem(key, String(value))
	} catch {
		// Private mode: no loop guard, but the check still works.
	}
}

/** Checks the Cloudflare login now and then, and reloads for it when it expired. */
export function registerAccessLoginCheck() {
	// Back from Cloudflare's login: tidy the address before the router reads it.
	const clean = withoutReloginParam(window.location.href)
	if (clean) window.history.replaceState(window.history.state, '', clean)

	let lastCheck = 0
	let checking = false
	async function check() {
		const now = Date.now()
		if (checking || now - lastCheck < CHECK_EVERY_MS || !navigator.onLine) return
		if (now - readNumber(RELOGIN_AT_KEY) < RELOGIN_PAUSE_MS) return
		checking = true
		lastCheck = now
		try {
			if (await accessLoginExpired()) {
				writeNumber(RELOGIN_AT_KEY, Date.now())
				window.location.replace(reloginUrl(window.location.href))
			}
		} finally {
			checking = false
		}
	}

	check()
	document.addEventListener('visibilitychange', () => {
		if (document.visibilityState === 'visible') check()
	})
	window.addEventListener('online', () => check())
}
