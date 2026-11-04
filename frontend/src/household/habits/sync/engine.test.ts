import {describe, expect, it, vi} from 'vitest'

import {createSyncEngine} from '../../sync/engine'
import {MemoryDocumentStorage} from '../../sync/idb'
import {normalizeHabit} from '../queries'
import {habitsModel, tickOp, type HabitOp, type HabitsData, type HabitsResult} from './model'

const TODAY = '2026-11-04'
const water = normalizeHabit({id: 1, title: 'Water', kind: 'daily_count', goal: 2, progress: {today: TODAY, goal: 2, due_today: true}})

function answer(habits = [water], rejected: HabitsResult['rejected'] = []): HabitsResult {
	return {cursor: 0, habits, groups: [], rejected}
}

describe('the habits engine', () => {
	it('keeps ticks made offline and sends them when there is signal', async () => {
		const storage = new MemoryDocumentStorage<HabitsData, HabitOp, HabitsResult>(habitsModel)
		storage.data = habitsModel.applyResult(habitsModel.empty(), answer())
		let online = false
		const transport = vi.fn(async (_body: {cursor: number, ops: HabitOp[]}) => {
			if (!online) throw new TypeError('Failed to fetch')
			return answer([{...water, progress: {...water.progress, today_count: 1}}])
		})
		const engine = createSyncEngine(habitsModel, storage, transport)
		await engine.loaded

		await engine.change(tickOp(water, TODAY))
		expect(engine.data.value.habits[0].progress.today_count).toBe(1)
		expect(storage.outbox).toHaveLength(1)

		await engine.sync()
		expect(engine.status.value).toBe('offline')
		expect(engine.pendingCount.value).toBe(1)

		online = true
		await engine.sync()
		expect(engine.status.value).toBe('synced')
		expect(engine.pendingCount.value).toBe(0)
		expect(storage.outbox).toHaveLength(0)
		const sent = transport.mock.calls.at(-1)![0]
		expect(sent.ops[0]).toMatchObject({type: 'tick', habit_id: 1, day: TODAY})
		expect(engine.data.value.habits[0].progress.today_count).toBe(1)
		engine.stop()
	})

	it('reports changes the server refused', async () => {
		const storage = new MemoryDocumentStorage<HabitsData, HabitOp, HabitsResult>(habitsModel)
		const onRejected = vi.fn()
		const op = tickOp(water, TODAY)
		const engine = createSyncEngine(habitsModel, storage, async () => answer([], [{opId: op.id, reason: 'gone'}]), {onRejected})
		await engine.change(op)
		await engine.sync()
		expect(onRejected).toHaveBeenCalledWith([{op, reason: 'gone'}])
		expect(engine.data.value.habits).toEqual([])
		engine.stop()
	})
})
