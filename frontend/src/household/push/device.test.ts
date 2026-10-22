import {describe, expect, it} from 'vitest'

import {keyBytes} from './device'

describe('keyBytes', () => {
	it('decodes the server key from base64url', () => {
		// "hello" in base64url is aGVsbG8 (no padding).
		expect([...keyBytes('aGVsbG8')]).toEqual([104, 101, 108, 108, 111])
		expect([...keyBytes('-_8')]).toEqual([251, 255])
	})
})
