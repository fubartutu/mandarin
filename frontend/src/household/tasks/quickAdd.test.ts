import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {createPinia, setActivePinia} from 'pinia'

import {parseTaskText, PrefixMode, PREFIXES} from '@/modules/quickAddMagic'
import {buildQuickAddTask} from '@/helpers/task'
import {PRIORITIES} from '@/constants/priorities'
import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'
import {applyHouseholdMagic, readHouseholdMagic} from './quickAdd'

// Tuesday 27 October 2026, 10:00 local time.
const NOW = () => new Date(2026, 9, 27, 10, 0)

// The whole of quick add except the network: ours, Vikunja's magic, the task.
function quickAdd(title: string, mode = PrefixMode.Default) {
	const {text, magic} = readHouseholdMagic(title, mode, NOW())
	const parsed = parseTaskText(text, mode, NOW())
	const task = applyHouseholdMagic(buildQuickAddTask(parsed, {title, project_id: 1}, PREFIXES[mode], []), magic)
	return {text, magic, task}
}

describe('household quick add magic', () => {
	beforeEach(() => {
		vi.useFakeTimers()
		vi.setSystemTime(NOW())
		setActivePinia(createPinia()) // the date parser reads the user's default time
	})
	afterEach(() => {
		vi.useRealTimers()
	})

	describe('every! (repeat from completion)', () => {
		it('repeats from when it is done', () => {
			const {task} = quickAdd('clean filter every! 3 days')
			expect(task.title).toBe('clean filter')
			expect(task.repeat_after).toBe(3 * 24 * 3600)
			expect(task.repeat_mode).toBe(TASK_REPEAT_MODES.REPEAT_MODE_FROM_CURRENT_DATE)
		})

		it('works for "every! month" too', () => {
			const {task} = quickAdd('haircut every! month')
			expect(task.repeat_mode).toBe(TASK_REPEAT_MODES.REPEAT_MODE_FROM_CURRENT_DATE)
			expect(task.repeat_after).toBe(30 * 24 * 3600)
		})

		it('plain "every" keeps repeating from the due date', () => {
			expect(quickAdd('clean filter every 3 days').task.repeat_mode).toBe(TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT)
		})

		it('leaves "every!" without a repeat after it alone', () => {
			expect(quickAdd('say hi to every! one').text).toBe('say hi to every! one')
		})
	})

	describe('starting <date>', () => {
		it('makes the date the first due date', () => {
			const {task} = quickAdd('water plants weekly starting nov 2 at 7am')
			expect(task.title).toBe('water plants')
			const due = new Date(task.due_date!)
			expect([due.getMonth(), due.getDate(), due.getHours()]).toEqual([10, 2, 7])
			expect(task.repeat_after).toBe(7 * 24 * 3600)
		})

		it('works with an end as well', () => {
			const {task, magic} = quickAdd('stretch daily starting monday until nov 30')
			expect(task.title).toBe('stretch')
			expect(new Date(task.due_date!).getDate()).toBe(2) // Monday 2 November
			expect(magic.repeatEnd?.until).toEqual(new Date(2026, 10, 30, 23, 59, 59))
		})

		it('only applies to repeating tasks with a date after it', () => {
			expect(quickAdd('starting line paint').text).toBe('starting line paint')
			expect(quickAdd('daily starting gun check').text).toBe('daily starting gun check')
		})
	})

	describe('p1-p4', () => {
		it('maps Todoist priorities, p1 the most urgent', () => {
			expect(quickAdd('fix the leak p1').task.priority).toBe(PRIORITIES.URGENT)
			expect(quickAdd('fix the leak p2').task.priority).toBe(PRIORITIES.HIGH)
			expect(quickAdd('P3 fix the leak').task.priority).toBe(PRIORITIES.MEDIUM)
			expect(quickAdd('fix the leak p4').task.priority).toBe(PRIORITIES.LOW)
			expect(quickAdd('fix the leak p1').task.title).toBe('fix the leak')
		})

		it('lets Vikunja\'s "!" priority win', () => {
			expect(quickAdd('fix the leak p1 !2').task.priority).toBe(2)
		})

		it('works in Todoist prefix mode', () => {
			const {task} = quickAdd('fix the leak p1 @home', PrefixMode.Todoist)
			expect(task.priority).toBe(PRIORITIES.URGENT)
			expect(task.title).toBe('fix the leak')
		})

		it('leaves words that only start with p alone', () => {
			expect(quickAdd('buy p10 filter').task.priority).toBe(0)
			expect(quickAdd('buy p2x battery').task.title).toBe('buy p2x battery')
		})
	})

	it('does nothing when quick add magic is off', () => {
		expect(readHouseholdMagic('fix it p1 every! day until nov 5', PrefixMode.Disabled, NOW())).toEqual({
			text: 'fix it p1 every! day until nov 5',
			magic: {repeatEnd: null, fromCompletion: 0, priority: null},
		})
	})

	it('a title that is only magic stays literal', () => {
		// Vikunja's rule: nothing is parsed out of it, so p1 must not be either.
		const {text} = readHouseholdMagic('p1', PrefixMode.Default, NOW())
		const parsed = parseTaskText(text, PrefixMode.Default, NOW())
		expect(parsed.text).toBe('')
	})
})
