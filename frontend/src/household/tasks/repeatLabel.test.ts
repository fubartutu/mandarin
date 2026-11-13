import {describe, expect, it} from 'vitest'
import {createI18n} from 'vue-i18n'

import messages from '../i18n/en.json'
import {fromRRule} from './repeatRule'
import {endLabel, repeatLabel, ruleLabel, type Translate} from './repeatLabel'

// The real household strings, so a reworded en.json shows up here.
const i18n = createI18n({legacy: false, locale: 'en', messages: {en: messages}})
const t = i18n.global.t as unknown as Translate
const locale = 'en-GB'
const now = new Date(2026, 10, 13)

const rule = (text: string) => {
	const parsed = fromRRule(text)
	if (!parsed) throw new Error(`unreadable rule ${text}`)
	return parsed
}

describe('ruleLabel', () => {
	it.each([
		['FREQ=WEEKLY;BYDAY=MO,TH', 'Every Mon, Thu'],
		['FREQ=WEEKLY;BYDAY=FR', 'Every Fri'],
		['FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR', 'Every weekday'],
		['FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE', 'Every 2 weeks on Mon, Wed'],
		['FREQ=MONTHLY;BYDAY=2TU', 'Every 2nd Tuesday'],
		['FREQ=MONTHLY;BYDAY=-1FR', 'Every last Friday'],
		['FREQ=MONTHLY;INTERVAL=3;BYDAY=1MO', 'Every 3 months on the 1st Monday'],
		['FREQ=MONTHLY;BYMONTHDAY=15', 'Monthly on day 15'],
		['FREQ=MONTHLY;BYMONTHDAY=-1', 'Monthly on the last day'],
		['FREQ=MONTHLY;INTERVAL=3', 'Every 3 months on the due date\'s day'],
		['FREQ=YEARLY', 'Every year on the due date'],
	])('%s reads "%s"', (text, label) => {
		expect(ruleLabel(rule(text), t, locale)).toBe(label)
	})
})

describe('endLabel', () => {
	it('is empty when the repeat never ends', () => {
		expect(endLabel({until: null, repeatsLeft: 0}, t, locale, now)).toBe('')
	})

	it('names the last day, with the year only when it is not this one', () => {
		expect(endLabel({until: new Date(2026, 11, 1), repeatsLeft: 0}, t, locale, now)).toBe('until 1 Dec')
		expect(endLabel({until: new Date(2027, 2, 1), repeatsLeft: 0}, t, locale, now)).toBe('until 1 Mar 2027')
	})

	it('counts the open occurrence in', () => {
		expect(endLabel({until: null, repeatsLeft: 3}, t, locale, now)).toBe('3 times left')
		expect(endLabel({until: null, repeatsLeft: 1}, t, locale, now)).toBe('last time')
	})

	it('gives both when both are set', () => {
		expect(endLabel({until: new Date(2026, 11, 1), repeatsLeft: 3}, t, locale, now)).toBe('until 1 Dec or 3 times left')
	})
})

describe('repeatLabel', () => {
	it('shows the rule instead of the stand-in interval', () => {
		expect(repeatLabel('Every 7 days', rule('FREQ=WEEKLY;BYDAY=MO,TH'), {until: null, repeatsLeft: 0}, t, locale, now))
			.toBe('Every Mon, Thu')
	})

	it('adds the end', () => {
		expect(repeatLabel('Every 30 days', rule('FREQ=MONTHLY;BYDAY=2TU'), {until: null, repeatsLeft: 3}, t, locale, now))
			.toBe('Every 2nd Tuesday, 3 times left')
	})

	it('keeps Vikunja\'s label without a rule, with the end when there is one', () => {
		expect(repeatLabel('Every 3 days', null, null, t, locale, now)).toBe('Every 3 days')
		expect(repeatLabel('Every 3 days', null, {until: new Date(2026, 11, 1), repeatsLeft: 0}, t, locale, now))
			.toBe('Every 3 days, until 1 Dec')
	})
})
