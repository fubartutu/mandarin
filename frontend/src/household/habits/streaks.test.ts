import {describe, expect, it} from 'vitest'

import {addDays} from '../menu/week'
import type {PeriodResponse, PeriodStatus} from './queries'
import {monthStarts, streakRuns, timelineSpan} from './streaks'

// One letter per day from 2026-11-01: d done, m missed, p partial, s skipped,
// r rest, t pending (today).
const letters: Record<string, PeriodStatus> = {d: 'done', m: 'missed', p: 'partial', s: 'skipped', r: 'rest', t: 'pending'}
function days(pattern: string): PeriodResponse[] {
	return [...pattern].map((c, i) => ({start: addDays('2026-11-01', i), status: letters[c], count: c === 'd' ? 1 : 0}))
}
const day = (i: number) => addDays('2026-11-01', i)

describe('streakRuns', () => {
	it('splits runs at two misses in a row', () => {
		const runs = streakRuns(days('ddmmddd'), 7)
		expect(runs.map(r => [r.start, r.end, r.length, r.ongoing])).toEqual([
			[day(0), day(1), 2, false],
			[day(4), day(6), 3, true],
		])
	})

	it('forgives one miss a week inside a run', () => {
		const [run] = streakRuns(days('dddmddd'), 7)
		expect(run.length).toBe(6)
		expect(run.forgiven).toEqual([day(3)])
	})

	it('ends the run at a second miss within the grace window', () => {
		const runs = streakRuns(days('ddmddmdd'), 7)
		expect(runs.map(r => r.length)).toEqual([4, 2])
		expect(runs[0].forgiven).toEqual([day(2)])
	})

	it('passes over skipped, rest and pending days', () => {
		const runs = streakRuns(days('ddsrdt'), 7)
		expect(runs).toEqual([{start: day(0), end: day(4), length: 3, forgiven: [], ongoing: true}])
	})

	it('counts partial as missed, and drops a forgiven miss that ended a run', () => {
		const runs = streakRuns(days('ddpm'), 7)
		expect(runs).toEqual([{start: day(0), end: day(1), length: 2, forgiven: [], ongoing: false}])
	})

	it('keeps a forgiven miss at the end of a run still going', () => {
		const [run] = streakRuns(days('ddm'), 7)
		expect(run.ongoing).toBe(true)
		expect(run.forgiven).toEqual([day(2)])
	})
})

describe('timelineSpan', () => {
	it('places and clips spans', () => {
		expect(timelineSpan('2026-11-01', '2026-11-10', '2026-11-01', '2026-11-05')).toEqual({left: 0, width: 50, clipped: false})
		expect(timelineSpan('2026-11-01', '2026-11-10', '2026-10-20', '2026-11-01')).toEqual({left: 0, width: 10, clipped: true})
		expect(timelineSpan('2026-11-01', '2026-11-10', '2026-10-01', '2026-10-05')).toBeNull()
	})
})

describe('monthStarts', () => {
	it('lists each 1st on the timeline, across a year end', () => {
		expect(monthStarts('2026-11-15', '2027-02-03')).toEqual(['2026-12-01', '2027-01-01', '2027-02-01'])
		expect(monthStarts('2026-11-01', '2026-11-30')).toEqual(['2026-11-01'])
	})
})
