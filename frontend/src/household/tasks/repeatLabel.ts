// Plain-words text for a repeat rule and its end, e.g. "Every Mon, Thu" and
// "until 1 Mar". Pure functions that take vue-i18n's `t` and the locale, so
// the tests need no component; RepeatSettingsField and the task page's
// collapsed Repeat row (useRepeatSummary) both use them.

import {WEEKDAYS, isWorkdays, type RepeatRule, type Weekday} from './repeatRule'
import type {RepeatSettings} from './queries'

/** The part of vue-i18n's `t` used here: key, named values, plural count. */
export type Translate = (key: string, named: Record<string, unknown>, plural: number) => string

export function weekdayName(day: Weekday, style: 'long' | 'short' | 'narrow', locale: string): string {
	// 5 January 2026 is a Monday.
	return new Date(2026, 0, 5 + WEEKDAYS.indexOf(day)).toLocaleDateString(locale, {weekday: style})
}

/** "Every Mon, Thu", "Every weekday", "Every 2nd Tuesday", "Every 2 months on day 15". */
export function ruleLabel(rule: RepeatRule, t: Translate, locale: string): string {
	const n = rule.interval
	switch (rule.kind) {
		case 'weekly': {
			// A 'unit' list joins with commas only ("Mon, Thu"), which reads
			// better in a short label than "Mon and Thu".
			const names = isWorkdays(rule.days)
				? t('household.tasks.repeat.weekdays', {}, 1)
				: new Intl.ListFormat(locale, {type: 'unit', style: 'short'})
					.format(rule.days.map(day => weekdayName(day, 'short', locale)))
			return t('household.tasks.repeat.summary.weekly', {days: names, n}, n)
		}
		case 'monthDay':
			return rule.day === -1
				? t('household.tasks.repeat.summary.lastDay', {n}, n)
				: t('household.tasks.repeat.summary.monthDay', {day: rule.day, n}, n)
		case 'monthWeekday':
			return t('household.tasks.repeat.summary.monthWeekday', {
				nth: t(`household.tasks.repeat.nth.${rule.nth}`, {}, 1).toLowerCase(),
				weekday: weekdayName(rule.weekday, 'long', locale),
				n,
			}, n)
		case 'months':
			return t('household.tasks.repeat.summary.months', {n}, n)
		case 'years':
			return t('household.tasks.repeat.summary.years', {n}, n)
	}
}

/** "until 1 Mar", "3 times left", "until 1 Mar or 3 times left"; '' when it never ends. */
export function endLabel(settings: Pick<RepeatSettings, 'until' | 'repeatsLeft'>, t: Translate, locale: string, now = new Date()): string {
	const parts: string[] = []
	if (settings.until) {
		const date = settings.until.toLocaleDateString(locale, {
			day: 'numeric',
			month: 'short',
			// Only name the year when it isn't this one.
			year: settings.until.getFullYear() === now.getFullYear() ? undefined : 'numeric',
		})
		parts.push(t('household.tasks.repeatEnd.short.until', {date}, 1))
	}
	if (settings.repeatsLeft) {
		parts.push(t('household.tasks.repeatEnd.short.count', {count: settings.repeatsLeft}, settings.repeatsLeft))
	}
	return parts.join(t('household.tasks.repeatEnd.short.or', {}, 1))
}

/** The collapsed row's text: the rule (or Vikunja's own label), then the end. */
export function repeatLabel(
	vikunjaLabel: string,
	rule: RepeatRule | null,
	settings: Pick<RepeatSettings, 'until' | 'repeatsLeft'> | null,
	t: Translate,
	locale: string,
	now = new Date(),
): string {
	const main = rule ? ruleLabel(rule, t, locale) : vikunjaLabel
	const end = settings ? endLabel(settings, t, locale, now) : ''
	return end ? t('household.tasks.repeatEnd.short.withEnd', {repeat: main, end}, 1) : main
}
