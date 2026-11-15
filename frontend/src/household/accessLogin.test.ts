import {describe, expect, it, vi} from 'vitest'

import {accessLoginExpired, reloginUrl, withoutReloginParam} from './accessLogin'

describe('accessLoginExpired', () => {
	it('is true when Cloudflare redirects to its login', async () => {
		const fetchFn = vi.fn(async () => ({type: 'opaqueredirect'}) as Response)
		expect(await accessLoginExpired(fetchFn, '/')).toBe(true)
		const [url, init] = fetchFn.mock.calls[0] as unknown as [string, RequestInit]
		expect(url).toMatch(/^\/\?mandarin-access-check=\d+$/)
		expect(init.redirect).toBe('manual')
	})

	it('is false when Mandarin answers, or without signal', async () => {
		expect(await accessLoginExpired(async () => ({type: 'basic'}) as Response, '/')).toBe(false)
		expect(await accessLoginExpired(async () => {
			throw new TypeError('Failed to fetch')
		}, '/')).toBe(false)
	})
})

describe('relogin URL', () => {
	it('marks the page and takes the mark off again', () => {
		const marked = reloginUrl('https://mandarin.example/shopping?list=3')
		expect(marked).toBe('https://mandarin.example/shopping?list=3&mandarin-relogin=1')
		expect(withoutReloginParam(marked)).toBe('https://mandarin.example/shopping?list=3')
		expect(withoutReloginParam('https://mandarin.example/shopping')).toBeNull()
	})
})
