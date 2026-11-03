import {describe, expect, it} from 'vitest'

import {normalizeGroup, normalizeHabit, type HabitResponse} from './queries'
import {badHabits, everyDay, habitSections, habitState, homeHabits, weekdayIndexes} from './today'

let nextId = 1
function habit(fields: Partial<Omit<HabitResponse, 'progress'>> & {progress?: Partial<HabitResponse['progress']>} = {}): HabitResponse {
	const h = normalizeHabit({id: nextId++, title: 'Habit', kind: 'daily'})
	return {...h, ...fields, progress: {...h.progress, ...fields.progress}}
}

const group = (id: number, position: number) => normalizeGroup({id, title: `Group ${id}`, position})

describe('habitState', () => {
	it('tells due from done', () => {
		expect(habitState(habit({progress: {due_today: true}}))).toBe('due')
		expect(habitState(habit({progress: {done_today: true}}))).toBe('done')
	})

	it('puts paused before anything else, and rest when nothing applies', () => {
		expect(habitState(habit({paused: true, progress: {due_today: true}}))).toBe('paused')
		expect(habitState(habit({progress: {skipped: true}}))).toBe('skipped')
		expect(habitState(habit())).toBe('rest')
	})

	it('marks bad habits as their own kind', () => {
		expect(habitState(habit({kind: 'bad', progress: {due_today: true}}))).toBe('bad')
	})
})

describe('habitSections', () => {
	it('orders routines and habits by position, ungrouped last', () => {
		const a = habit({groupId: 2, position: 2})
		const b = habit({groupId: 2, position: 1})
		const c = habit({groupId: 1})
		const loose = habit()
		const sections = habitSections([a, b, c, loose], [group(1, 20), group(2, 10)])
		expect(sections.map(s => s.group?.id)).toEqual([2, 1, undefined])
		expect(sections[0].habits).toEqual([b, a])
	})

	it('leaves out bad habits and empty routines', () => {
		const bad = habit({kind: 'bad', groupId: 1})
		expect(habitSections([bad], [group(1, 1)])).toEqual([])
		expect(badHabits([bad, habit()])).toEqual([bad])
	})

	it('treats a habit in a routine it cannot see as ungrouped', () => {
		const h = habit({groupId: 99})
		expect(habitSections([h], [])).toEqual([{habits: [h]}])
	})
})

describe('homeHabits', () => {
	it('shows due habits first, then done ones, and nothing else', () => {
		const done = habit({position: 1, progress: {done_today: true}})
		const due = habit({position: 2, progress: {due_today: true}})
		const rest = habit({position: 3})
		const paused = habit({paused: true, progress: {due_today: true}})
		expect(homeHabits([done, due, rest, paused], [])).toEqual([due, done])
	})
})

describe('weekdays', () => {
	it('lists the chosen days, Monday first', () => {
		expect(weekdayIndexes('1010100')).toEqual([0, 2, 4])
		expect(everyDay('1111111')).toBe(true)
		expect(everyDay('1111100')).toBe(false)
	})
})
