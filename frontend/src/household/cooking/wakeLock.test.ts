import {describe, expect, it, vi} from 'vitest'

import {holdScreenAwake, type WakeLockEnv} from './wakeLock'

// A stand-in for the browser: a page that can be hidden and shown, and a
// wake lock that hands out sentinels (the lock objects) and records them.
function fakeBrowser(options: {supported?: boolean, refuse?: boolean} = {}) {
	const page = new EventTarget() as EventTarget & {visibilityState: DocumentVisibilityState}
	page.visibilityState = 'visible'
	const sentinels: {type: string, released: boolean, release: () => Promise<void>}[] = []
	const pending: (() => void)[] = []
	const request = vi.fn((type: 'screen') => {
		if (options.refuse) return Promise.reject(new DOMException('Not allowed', 'NotAllowedError'))
		const sentinel = {type, released: false, release: async () => { sentinel.released = true }}
		sentinels.push(sentinel)
		return new Promise(resolve => pending.push(() => resolve(sentinel)))
	})
	const env = {
		navigator: options.supported === false ? {} : {wakeLock: {request}},
		document: page,
	} as unknown as WakeLockEnv
	return {
		env,
		request,
		sentinels,
		/** Lets the browser answer the outstanding requests. */
		async grant() {
			pending.splice(0).forEach(resolve => resolve())
			await Promise.resolve()
			await Promise.resolve()
		},
		/** The browser drops the lock when the page is hidden; showing it fires again. */
		async hideAndShow() {
			for (const s of sentinels) s.released = true
			page.visibilityState = 'hidden'
			page.dispatchEvent(new Event('visibilitychange'))
			page.visibilityState = 'visible'
			page.dispatchEvent(new Event('visibilitychange'))
			await Promise.resolve()
		},
	}
}

describe('holdScreenAwake', () => {
	it('asks for a screen lock and lets it go on release', async () => {
		const browser = fakeBrowser()
		const release = holdScreenAwake(browser.env)
		await browser.grant()
		expect(browser.request).toHaveBeenCalledWith('screen')
		expect(browser.sentinels[0].released).toBe(false)

		release()
		expect(browser.sentinels[0].released).toBe(true)
	})

	it('does nothing, without an error, where the browser has no wake lock', () => {
		const browser = fakeBrowser({supported: false})
		const release = holdScreenAwake(browser.env)
		expect(() => release()).not.toThrow()
	})

	it('takes the lock again when the page comes back after being hidden', async () => {
		const browser = fakeBrowser()
		const release = holdScreenAwake(browser.env)
		await browser.grant()
		await browser.hideAndShow()
		await browser.grant()
		expect(browser.request).toHaveBeenCalledTimes(2)
		expect(browser.sentinels[1].released).toBe(false)

		release()
		expect(browser.sentinels[1].released).toBe(true)
	})

	it('stops asking once released', async () => {
		const browser = fakeBrowser()
		holdScreenAwake(browser.env)()
		await browser.hideAndShow()
		expect(browser.request).toHaveBeenCalledTimes(1)
	})

	it('lets go of a lock granted after cooking mode closed', async () => {
		const browser = fakeBrowser()
		const release = holdScreenAwake(browser.env)
		release()
		await browser.grant()
		expect(browser.sentinels[0].released).toBe(true)
	})

	it('swallows a refused request (e.g. low battery)', async () => {
		const browser = fakeBrowser({refuse: true})
		const release = holdScreenAwake(browser.env)
		await browser.grant()
		expect(() => release()).not.toThrow()
	})
})
