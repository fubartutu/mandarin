import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {createPinia, setActivePinia} from 'pinia'

import {parseTaskText} from '@/modules/quickAddMagic'
import {extractRepeatEnd} from './repeatEnd'

// Tuesday 27 October 2026, 10:00 local time.
const now = new Date(2026, 9, 27, 10, 0)

// Both halves of quick add: our end, then Vikunja's magic on what's left.
function quickAdd(title: string) {
	const {text, end} = extractRepeatEnd(title, now)
	const parsed = parseTaskText(text, undefined, now)
	return {text, end, parsed}
}

describe('extractRepeatEnd', () => {
	beforeEach(() => {
		vi.useFakeTimers()
		vi.setSystemTime(now)
		setActivePinia(createPinia()) // the date parser reads the user's default time
	})
	afterEach(() => {
		vi.useRealTimers()
	})

	it('reads "until <date>" and leaves the time for the due date', () => {
		const {text, end, parsed} = quickAdd('do this task at 5pm daily until 5th November')
		expect(end?.until).toEqual(new Date(2026, 10, 5, 23, 59, 59))
		expect(end?.repeatsLeft).toBe(0)
		expect(text).toBe('do this task at 5pm daily')
		expect(parsed.text).toBe('do this task')
		expect(parsed.repeats).toEqual({amount: 1, type: 'days'})
		expect(parsed.date?.getHours()).toBe(17)
		expect(parsed.date?.getDate()).not.toBe(5)
	})

	it('moves a time typed after the end date back to the task', () => {
		const {text, end} = quickAdd('water plants every 2 days until nov 20 at 7am')
		expect(end?.until).toEqual(new Date(2026, 10, 20, 23, 59, 59))
		expect(text).toBe('water plants every 2 days at 7am')
	})

	it('reads a weekday as the end date', () => {
		const {end} = quickAdd('stretch daily until friday')
		expect(end?.until).toEqual(new Date(2026, 9, 30, 23, 59, 59))
	})

	it('reads a number of times', () => {
		expect(quickAdd('take tablets daily 10 times').end).toEqual({until: null, repeatsLeft: 10})
		expect(quickAdd('take tablets daily for five times').end).toEqual({until: null, repeatsLeft: 5})
		expect(quickAdd('physio weekly x6').text).toBe('physio weekly')
	})

	it('reads "for <span>" as an end date', () => {
		const {text, end} = quickAdd('practice piano daily for 3 weeks')
		expect(text).toBe('practice piano daily')
		expect(end?.until).toEqual(new Date(2026, 10, 16, 23, 59, 59))
	})

	it('takes both an end date and a count', () => {
		const {end} = quickAdd('call mum weekly until 1 dec 4 times')
		expect(end).toEqual({until: new Date(2026, 11, 1, 23, 59, 59), repeatsLeft: 4})
	})

	it('leaves titles that do not repeat alone', () => {
		expect(extractRepeatEnd('read until chapter 5', now)).toEqual({text: 'read until chapter 5', end: null})
		expect(extractRepeatEnd('buy 10 times tables book', now).end).toBeNull()
	})

	it('leaves "until" without a date in the title', () => {
		expect(extractRepeatEnd('daily walk until dark', now)).toEqual({text: 'daily walk until dark', end: null})
	})
})
