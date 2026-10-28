import {describe, expect, it} from 'vitest'

import {isAppleTouch, withMaximumScale} from './ios'

describe('isAppleTouch', () => {
	const nav = (userAgent: string, maxTouchPoints = 0) => ({userAgent, maxTouchPoints})

	it('finds iPhones and iPads, also an iPad asking for the desktop site', () => {
		expect(isAppleTouch(nav('Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 Version/26.0 Mobile/15E148 Safari/604.1', 5))).toBe(true)
		expect(isAppleTouch(nav('Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) AppleWebKit/605.1.15', 5))).toBe(true)
		expect(isAppleTouch(nav('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/18.0 Safari/605.1.15', 5))).toBe(true)
	})

	it('leaves Android and Macs without touch alone', () => {
		expect(isAppleTouch(nav('Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 Chrome/141.0 Mobile Safari/537.36', 5))).toBe(false)
		expect(isAppleTouch(nav('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/18.0 Safari/605.1.15', 0))).toBe(false)
	})
})

describe('withMaximumScale', () => {
	it('adds maximum-scale once', () => {
		expect(withMaximumScale('width=device-width,initial-scale=1.0')).toBe('width=device-width,initial-scale=1.0,maximum-scale=1')
		expect(withMaximumScale('width=device-width,maximum-scale=2')).toBe('width=device-width,maximum-scale=2')
		expect(withMaximumScale('')).toBe('width=device-width,initial-scale=1,maximum-scale=1')
	})
})
