import {describe, expect, it} from 'vitest'

import {normalizeHabit, type HabitResponse} from '../queries'
import {applyOp, emptyData, normalizeSyncResult, rollToDay, skipOp, tickOp, untickOp, type HabitsData} from './model'

// 2026-11-04 is a Wednesday.
const TODAY = '2026-11-04'

function habit(fields: Partial<Omit<HabitResponse, 'progress'>> & {progress?: Partial<HabitResponse['progress']>} = {}): HabitResponse {
	const h = normalizeHabit({id: 1, title: 'Stretch', kind: 'daily', progress: {today: TODAY, goal: 1, due_today: true}})
	return {...h, ...fields, progress: {...h.progress, ...fields.progress}}
}

const data = (...habits: HabitResponse[]): HabitsData => ({...emptyData(), habits, loaded: true})

describe('applyOp', () => {
	it('shows a tick at once and takes it off again', () => {
		const h = habit()
		const tick = tickOp(h, TODAY)
		let d = applyOp(data(h), tick)
		expect(d.habits[0].progress).toMatchObject({today_count: 1, done_today: true, due_today: false, today_entries: [tick.uid]})
		d = applyOp(d, untickOp(h, tick.uid!))
		expect(d.habits[0].progress).toMatchObject({today_count: 0, done_today: false, due_today: true})
	})

	it('counts up to the goal', () => {
		const h = habit({kind: 'daily_count', progress: {goal: 2}})
		let d = applyOp(data(h), tickOp(h, TODAY))
		expect(d.habits[0].progress).toMatchObject({today_count: 1, done_today: false, due_today: true})
		d = applyOp(d, tickOp(h, TODAY))
		expect(d.habits[0].progress.done_today).toBe(true)
	})

	it('never makes a bad habit done, and counts the week', () => {
		const h = habit({kind: 'bad', progress: {goal: 0, due_today: false, this_week: 2}})
		const d = applyOp(data(h), tickOp(h, TODAY))
		expect(d.habits[0].progress).toMatchObject({today_count: 1, done_today: false, this_week: 3})
	})

	it('leaves today alone for a late tick on an earlier day', () => {
		const h = habit()
		expect(applyOp(data(h), tickOp(h, '2026-11-02')).habits[0]).toEqual(h)
	})

	it('applies the same tick once', () => {
		const h = habit({kind: 'daily_count', progress: {goal: 5}})
		const tick = tickOp(h, TODAY)
		const d = applyOp(applyOp(data(h), tick), tick)
		expect(d.habits[0].progress.today_count).toBe(1)
	})

	it('skips and un-skips today', () => {
		const h = habit()
		let d = applyOp(data(h), skipOp(h, TODAY, true))
		expect(d.habits[0].progress).toMatchObject({skipped: true, due_today: false})
		d = applyOp(d, skipOp(h, TODAY, false))
		expect(d.habits[0].progress).toMatchObject({skipped: false, due_today: true})
	})

	it('only changes the habit the op is for', () => {
		const a = habit()
		const b = habit({id: 2})
		expect(applyOp(data(a, b), tickOp(a, TODAY)).habits[1]).toBe(b)
	})
})

describe('rollToDay', () => {
	it('starts a new day empty, on the habit\'s own days', () => {
		const done = habit({progress: {today_count: 1, done_today: true, due_today: false, today_entries: ['x']}})
		expect(rollToDay(done, '2026-11-05').progress).toMatchObject({
			today: '2026-11-05', today_count: 0, today_entries: [], done_today: false, due_today: true,
		})
		const weekdaysOnly = habit({weekdays: '1111100'})
		expect(rollToDay(weekdaysOnly, '2026-11-07').progress.due_today).toBe(false) // Saturday
	})

	it('keeps a weekly habit\'s ticks until Monday', () => {
		const weekly = habit({kind: 'weekly', progress: {goal: 3, today_count: 2, today_entries: ['a', 'b']}})
		expect(rollToDay(weekly, '2026-11-06').progress.today_count).toBe(2)
		expect(rollToDay(weekly, '2026-11-09').progress.today_count).toBe(0)
	})

	it('keeps paused habits off', () => {
		expect(rollToDay(habit({paused: true}), '2026-11-05').progress.due_today).toBe(false)
	})

	it('changes nothing on the same day', () => {
		const h = habit()
		expect(rollToDay(h, TODAY)).toBe(h)
	})

	it('moves the copy on when a tick is for a later day', () => {
		const h = habit({progress: {today_count: 1, done_today: true, due_today: false}})
		const d = applyOp(data(h), tickOp(h, '2026-11-05'))
		expect(d.habits[0].progress).toMatchObject({today: '2026-11-05', today_count: 1, done_today: true})
	})
})

describe('normalizeSyncResult', () => {
	it('reads the answer', () => {
		const r = normalizeSyncResult({
			habits: [{id: 3, title: 'Water'}],
			groups: [{id: 1, title: 'Morning'}],
			rejected: [{op_id: 'op1', reason: 'gone'}],
		})
		expect(r.cursor).toBe(0)
		expect(r.habits[0].title).toBe('Water')
		expect(r.groups[0].title).toBe('Morning')
		expect(r.rejected).toEqual([{opId: 'op1', reason: 'gone'}])
	})
})
