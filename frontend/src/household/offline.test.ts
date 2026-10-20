import {describe, expect, it} from 'vitest'

import {ApiError} from '@/client/problem'
import {isUnreachable, StartTimeoutError} from './offline'

describe('isUnreachable', () => {
	it('is true for no connection, timeouts and a down server', () => {
		expect(isUnreachable(new TypeError('Failed to fetch'))).toBe(true)
		expect(isUnreachable(new StartTimeoutError())).toBe(true)
		expect(isUnreachable(new DOMException('timed out', 'TimeoutError'))).toBe(true)
		expect(isUnreachable(new ApiError({status: 503, detail: 'down'}))).toBe(true)
	})

	it('is false when the server answered', () => {
		expect(isUnreachable(new ApiError({status: 401, detail: 'expired'}))).toBe(false)
		expect(isUnreachable({status: 429})).toBe(false)
		expect(isUnreachable(new Error('something else'))).toBe(false)
	})
})
