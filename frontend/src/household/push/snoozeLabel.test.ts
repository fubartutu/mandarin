import {describe, expect, it} from 'vitest'

import {snoozeLabel} from './snoozeLabel'

describe('snoozeLabel', () => {
	const now = new Date(2026, 10, 3, 9, 40) // Tue 3 Nov, local time

	it('is just the time today', () => {
		expect(snoozeLabel(new Date(2026, 10, 3, 10, 40), now, 'en-GB')).toBe('10:40')
	})

	it('adds the day when it comes back tomorrow', () => {
		expect(snoozeLabel(new Date(2026, 10, 4, 0, 15), now, 'en-GB')).toBe('Wed 0:15')
	})
})
