// Keeps the screen on while cooking mode is open (Screen Wake Lock API).
// Browsers without it (older Safari, Firefox before 126) just skip it, and a
// refused request (low battery, no permission) is ignored: cooking mode works
// the same, the screen may only dim.

export interface WakeLockEnv {
	navigator: Pick<Navigator, 'wakeLock'> | Record<string, never>
	document: Pick<Document, 'visibilityState' | 'addEventListener' | 'removeEventListener'>
}

/** Holds a screen wake lock until the returned function is called. */
export function holdScreenAwake(env: WakeLockEnv = {navigator, document}): () => void {
	const wakeLock = 'wakeLock' in env.navigator ? env.navigator.wakeLock : undefined
	if (!wakeLock) return () => {}

	let sentinel: WakeLockSentinel | undefined
	let held = true
	let asking = false

	function request() {
		if (asking) return
		asking = true
		wakeLock!.request('screen').then(lock => {
			// Let go straight away if cooking mode closed while we waited.
			if (held) sentinel = lock
			else lock.release().catch(() => {})
		}).catch(() => {}).finally(() => {
			asking = false
		})
	}

	// The browser drops the lock whenever the page is hidden (another app,
	// screen locked), so take it again when the page is back.
	function onVisibilityChange() {
		if (env.document.visibilityState === 'visible' && (!sentinel || sentinel.released)) request()
	}

	request()
	env.document.addEventListener('visibilitychange', onVisibilityChange)
	return () => {
		held = false
		env.document.removeEventListener('visibilitychange', onVisibilityChange)
		sentinel?.release().catch(() => {})
		sentinel = undefined
	}
}
