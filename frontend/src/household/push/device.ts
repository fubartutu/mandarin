// This device's side of push notifications: whether it can have them,
// subscribing with the browser's push service, and telling Mandarin.

import {pushDevicesRemove, pushDevicesSave, pushKey} from '@/client/generated'

export type DeviceState =
	| 'unsupported' // no Web Push in this browser (or plain http)
	| 'needs-home-screen' // iPhone/iPad: only the home-screen app gets pushes
	| 'blocked' // the person said no; only the browser's settings can undo it
	| 'off'
	| 'on'

function isIos(): boolean {
	// iPadOS reports itself as a Mac; touch gives it away.
	return /iPhone|iPad|iPod/.test(navigator.userAgent) ||
		(navigator.userAgent.includes('Macintosh') && navigator.maxTouchPoints > 1)
}

function isHomeScreenApp(): boolean {
	return window.matchMedia('(display-mode: standalone)').matches ||
		(navigator as Navigator & {standalone?: boolean}).standalone === true
}

function supported(): boolean {
	return 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window
}

/** What this device can do now. */
export async function deviceState(): Promise<DeviceState> {
	if (isIos() && !isHomeScreenApp()) return 'needs-home-screen'
	if (!supported()) return 'unsupported'
	if (Notification.permission === 'denied') return 'blocked'
	const registration = await navigator.serviceWorker.ready
	const subscription = await registration.pushManager.getSubscription()
	return subscription && Notification.permission === 'granted' ? 'on' : 'off'
}

/** "iPhone · Safari", "Mac · Chrome": how the device shows in the list. */
export function deviceName(): string {
	const ua = navigator.userAgent
	const device = /iPhone/.test(ua) ? 'iPhone'
		: /iPad/.test(ua) || (ua.includes('Macintosh') && navigator.maxTouchPoints > 1) ? 'iPad'
			: /Android/.test(ua) ? 'Android'
				: /Macintosh/.test(ua) ? 'Mac'
					: /Windows/.test(ua) ? 'Windows'
						: /Linux/.test(ua) ? 'Linux' : 'Device'
	const browser = /Edg\//.test(ua) ? 'Edge'
		: /Firefox\//.test(ua) ? 'Firefox'
			: /Chrome\//.test(ua) ? 'Chrome'
				: /Safari\//.test(ua) ? 'Safari' : ''
	return browser ? `${device} · ${browser}` : device
}

/** The server key (base64url) as the bytes PushManager.subscribe wants. */
export function keyBytes(base64url: string): Uint8Array<ArrayBuffer> {
	const base64 = (base64url + '='.repeat((4 - base64url.length % 4) % 4)).replace(/-/g, '+').replace(/_/g, '/')
	const raw = atob(base64)
	const bytes = new Uint8Array(new ArrayBuffer(raw.length))
	for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i)
	return bytes
}

async function register(subscription: PushSubscription) {
	const json = subscription.toJSON()
	await pushDevicesSave({body: {
		endpoint: subscription.endpoint,
		keys: {p256dh: json.keys?.p256dh ?? '', auth: json.keys?.auth ?? ''},
		name: deviceName(),
	}})
}

/**
 * Turns push on for this device. Must be called from a tap: iOS only shows
 * the permission prompt then. Returns the state afterwards.
 */
export async function turnOn(): Promise<DeviceState> {
	const permission = await Notification.requestPermission()
	if (permission === 'denied') return 'blocked'
	if (permission !== 'granted') return 'off'

	const registration = await navigator.serviceWorker.ready
	const key = keyBytes((await pushKey()).data.public_key ?? '')
	let subscription = await registration.pushManager.getSubscription()
	// Subscribed with another server key (e.g. a restored older database): start over.
	if (subscription && subscription.options.applicationServerKey &&
		!sameBytes(new Uint8Array(subscription.options.applicationServerKey), key)) {
		await subscription.unsubscribe()
		subscription = null
	}
	subscription ??= await registration.pushManager.subscribe({userVisibleOnly: true, applicationServerKey: key})
	await register(subscription)
	return 'on'
}

/** Turns push off for this device, here and on the server. */
export async function turnOff(): Promise<DeviceState> {
	const registration = await navigator.serviceWorker.ready
	const subscription = await registration.pushManager.getSubscription()
	if (subscription) {
		await pushDevicesRemove({body: {endpoint: subscription.endpoint}}).catch(() => undefined)
		await subscription.unsubscribe()
	}
	return 'off'
}

/**
 * Re-sends this device's subscription when the app opens: the browser may
 * have replaced it, or another person may have signed in on this device (it
 * then becomes theirs). Quiet: does nothing without permission or signal.
 */
export async function refreshSubscription(): Promise<void> {
	try {
		if (!supported() || Notification.permission !== 'granted') return
		const registration = await navigator.serviceWorker.ready
		const subscription = await registration.pushManager.getSubscription()
		if (subscription) await register(subscription)
	} catch {
		// Offline, or push unavailable: try again next time.
	}
}

function sameBytes(a: Uint8Array, b: Uint8Array): boolean {
	return a.length === b.length && a.every((v, i) => v === b[i])
}
