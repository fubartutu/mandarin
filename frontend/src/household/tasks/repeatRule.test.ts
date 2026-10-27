import {describe, expect, it} from 'vitest'

import {extractRepeatRule, firstOccurrence, fromRRule, standInSeconds, toRRule, type RepeatRule} from './repeatRule'

const DAY = 24 * 3600

describe('RRULE round trip', () => {
	const cases: [RepeatRule, string][] = [
		[{kind: 'weekly', interval: 1, days: ['TH', 'MO']}, 'FREQ=WEEKLY;BYDAY=MO,TH'],
		[{kind: 'weekly', interval: 2, days: ['FR']}, 'FREQ=WEEKLY;INTERVAL=2;BYDAY=FR'],
		[{kind: 'monthDay', interval: 1, day: 15}, 'FREQ=MONTHLY;BYMONTHDAY=15'],
		[{kind: 'monthDay', interval: 1, day: -1}, 'FREQ=MONTHLY;BYMONTHDAY=-1'],
		[{kind: 'monthWeekday', interval: 1, nth: 2, weekday: 'TU'}, 'FREQ=MONTHLY;BYDAY=2TU'],
		[{kind: 'monthWeekday', interval: 1, nth: -1, weekday: 'FR'}, 'FREQ=MONTHLY;BYDAY=-1FR'],
		[{kind: 'months', interval: 2}, 'FREQ=MONTHLY;INTERVAL=2'],
		[{kind: 'years', interval: 1}, 'FREQ=YEARLY'],
	]
	it.each(cases)('%o <-> %s', (rule, text) => {
		expect(toRRule(rule)).toBe(text)
		const back = fromRRule(text)
		expect(back).toEqual(rule.kind === 'weekly' ? {...rule, days: [...rule.days].sort((a, b) => 'MOTUWETHFRSASU'.indexOf(a) - 'MOTUWETHFRSASU'.indexOf(b))} : rule)
	})

	it('reads nothing it cannot show', () => {
		expect(fromRRule('')).toBeNull()
		expect(fromRRule('FREQ=MONTHLY;BYSETPOS=2;BYDAY=TU')).toBeNull()
		expect(fromRRule('FREQ=DAILY')).toBeNull()
		expect(fromRRule('FREQ=MONTHLY;BYDAY=5TU')).toBeNull()
	})

	it('stands in with a week, a 30-day month or a 365-day year', () => {
		expect(standInSeconds({kind: 'weekly', interval: 2, days: ['MO']})).toBe(14 * DAY)
		expect(standInSeconds({kind: 'monthWeekday', interval: 1, nth: 2, weekday: 'TU'})).toBe(30 * DAY)
		expect(standInSeconds({kind: 'years', interval: 1})).toBe(365 * DAY)
	})
})

describe('firstOccurrence', () => {
	// Tuesday 27 October 2026, 17:00.
	const tuesday = new Date(2026, 9, 27, 17, 0)

	it('takes today when it matches, keeping the time', () => {
		expect(firstOccurrence({kind: 'weekly', interval: 1, days: ['TU', 'FR']}, tuesday)).toEqual(tuesday)
	})
	it('finds the next matching weekday', () => {
		expect(firstOccurrence({kind: 'weekly', interval: 1, days: ['MO', 'TH']}, tuesday)).toEqual(new Date(2026, 9, 29, 17, 0))
		expect(firstOccurrence({kind: 'weekly', interval: 1, days: ['MO']}, tuesday)).toEqual(new Date(2026, 10, 2, 17, 0))
	})
	it('finds the nth weekday this month or next', () => {
		// 2nd Tuesday of October was the 13th, so November's: the 10th.
		expect(firstOccurrence({kind: 'monthWeekday', interval: 1, nth: 2, weekday: 'TU'}, tuesday)).toEqual(new Date(2026, 10, 10, 17, 0))
		expect(firstOccurrence({kind: 'monthWeekday', interval: 1, nth: -1, weekday: 'FR'}, tuesday)).toEqual(new Date(2026, 9, 30, 17, 0))
	})
	it('finds a day of the month, skipping months without it', () => {
		expect(firstOccurrence({kind: 'monthDay', interval: 1, day: -1}, tuesday)).toEqual(new Date(2026, 9, 31, 17, 0))
		expect(firstOccurrence({kind: 'monthDay', interval: 1, day: 15}, tuesday)).toEqual(new Date(2026, 10, 15, 17, 0))
		expect(firstOccurrence({kind: 'monthDay', interval: 1, day: 31}, new Date(2026, 10, 1, 9, 0))).toEqual(new Date(2026, 11, 31, 9, 0))
	})
	it('keeps the date for month and year intervals', () => {
		expect(firstOccurrence({kind: 'months', interval: 2}, tuesday)).toEqual(tuesday)
	})
})

describe('extractRepeatRule', () => {
	const cases: [string, string, RepeatRule][] = [
		['bins every mon, thu', 'bins weekly', {kind: 'weekly', interval: 1, days: ['MO', 'TH']}],
		['bins every monday and thursday at 7pm', 'bins weekly at 7pm', {kind: 'weekly', interval: 1, days: ['MO', 'TH']}],
		['gym every mon, wed & fri', 'gym weekly', {kind: 'weekly', interval: 1, days: ['MO', 'WE', 'FR']}],
		['standup every weekday', 'standup weekly', {kind: 'weekly', interval: 1, days: ['MO', 'TU', 'WE', 'TH', 'FR']}],
		['lie in every weekend', 'lie in weekly', {kind: 'weekly', interval: 1, days: ['SA', 'SU']}],
		['call nan every sunday', 'call nan weekly', {kind: 'weekly', interval: 1, days: ['SU']}],
		['recycling every other friday', 'recycling weekly', {kind: 'weekly', interval: 2, days: ['FR']}],
		['payroll every 2 weeks on thursday', 'payroll weekly', {kind: 'weekly', interval: 2, days: ['TH']}],
		['book club every 2nd tuesday', 'book club monthly', {kind: 'monthWeekday', interval: 1, nth: 2, weekday: 'TU'}],
		['market every last saturday of the month', 'market monthly', {kind: 'monthWeekday', interval: 1, nth: -1, weekday: 'SA'}],
		['pay rent every 15th', 'pay rent monthly', {kind: 'monthDay', interval: 1, day: 15}],
		['pay rent monthly on the 1st', 'pay rent monthly', {kind: 'monthDay', interval: 1, day: 1}],
		['invoice every last day of the month', 'invoice monthly', {kind: 'monthDay', interval: 1, day: -1}],
		['water bill every 3 months', 'water bill monthly', {kind: 'months', interval: 3}],
		['car service every 6 months until dec 2027', 'car service monthly until dec 2027', {kind: 'months', interval: 6}],
		['rego yearly', 'rego yearly', {kind: 'years', interval: 1}],
		['passport every 10 years', 'passport yearly', {kind: 'years', interval: 10}],
		['dentist biannually', 'dentist monthly', {kind: 'months', interval: 6}],
	]
	it.each(cases)('%s', (title, text, rule) => {
		const result = extractRepeatRule(title)
		expect(result.text).toBe(text)
		expect(result.rule).toEqual(rule)
	})

	it('leaves plain intervals to Vikunja', () => {
		for (const title of ['water plants daily', 'clean every 2 weeks', 'rent monthly', 'rent every month', 'weekly shop']) {
			expect(extractRepeatRule(title)).toEqual({text: title, rule: null})
		}
	})

	it('does not read weekdays inside other words', () => {
		expect(extractRepeatRule('everyone monday meeting').rule).toBeNull()
		expect(extractRepeatRule('every sunday-ish').rule).toBeNull()
	})
})
