import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {effectScope} from 'vue'

import {useCookingTimers, type TimerAlarm} from './useCookingTimers'

function fakeAlarm(): TimerAlarm & {ringing: boolean} {
	const alarm = {
		ringing: false,
		prepare: vi.fn(),
		vibrate: vi.fn(),
		startRinging: vi.fn(() => { alarm.ringing = true }),
		stopRinging: vi.fn(() => { alarm.ringing = false }),
		dispose: vi.fn(() => { alarm.ringing = false }),
	}
	return alarm
}

// Cooking mode owns the timers: its effect scope stands in for the view, and
// stopping the scope is closing cooking mode.
function openCookingMode() {
	const alarm = fakeAlarm()
	const scope = effectScope()
	const timers = scope.run(() => useCookingTimers({alarm}))!
	return {alarm, timers, close: () => scope.stop()}
}

const START = new Date('2026-10-24T18:00:00Z').getTime()

describe('useCookingTimers', () => {
	beforeEach(() => {
		vi.useFakeTimers()
		vi.setSystemTime(START)
	})
	afterEach(() => vi.useRealTimers())

	it('starts a timer as plain data that counts down once a second', () => {
		const {timers} = openCookingMode()
		const timer = timers.start(3, {label: '20 min', seconds: 1200})

		expect(timer).toEqual({
			id: expect.any(String),
			stepNumber: 3,
			label: '20 min',
			seconds: 1200,
			endsAt: START + 1_200_000,
		})
		expect(JSON.parse(JSON.stringify(timers.timers.value))).toEqual([timer])
		expect(timers.running.value.map(t => t.remaining)).toEqual([1200])

		vi.advanceTimersByTime(5000)
		expect(timers.running.value.map(t => t.remaining)).toEqual([1195])
	})

	it('works out the time left from the clock, so a sleeping phone catches up', () => {
		const {timers} = openCookingMode()
		timers.start(1, {label: '10 minutes', seconds: 600})

		// No ticks for four minutes (screen off, interval throttled), then one.
		vi.setSystemTime(START + 240_000)
		vi.advanceTimersByTime(1000)
		expect(timers.running.value[0].remaining).toBe(359)
	})

	it('runs several at once, the one finishing first on top', () => {
		const {timers} = openCookingMode()
		timers.start(2, {label: '1 hour', seconds: 3600})
		timers.start(5, {label: '5 min', seconds: 300})
		timers.start(4, {label: '20 minutes', seconds: 1200})

		expect(timers.running.value.map(t => t.label)).toEqual(['5 min', '20 minutes', '1 hour'])
	})

	it('tells a step which of its times are already running, and does not start one twice', () => {
		const {timers} = openCookingMode()
		const first = timers.start(3, {label: '20 min', seconds: 1200})

		expect(timers.runningOn(3, '20 min')?.id).toBe(first.id)
		expect(timers.runningOn(4, '20 min')).toBeUndefined()
		expect(timers.start(3, {label: '20 min', seconds: 1200}).id).toBe(first.id)
		expect(timers.timers.value).toHaveLength(1)
	})

	it('cancels a running timer without an alarm', () => {
		const {timers, alarm} = openCookingMode()
		const timer = timers.start(1, {label: '5 min', seconds: 300})
		timers.cancel(timer.id)

		vi.advanceTimersByTime(400_000)
		expect(timers.running.value).toEqual([])
		expect(timers.finished.value).toEqual([])
		expect(alarm.vibrate).not.toHaveBeenCalled()
		expect(alarm.ringing).toBe(false)
	})

	it('readies the sound on the tap that starts a timer', () => {
		const {timers, alarm} = openCookingMode()
		timers.start(1, {label: '5 min', seconds: 300})
		expect(alarm.prepare).toHaveBeenCalled()
	})

	it('finishes with a vibration and rings until the banner is dismissed', () => {
		const {timers, alarm} = openCookingMode()
		const timer = timers.start(3, {label: '20 min', seconds: 1200})

		vi.advanceTimersByTime(1_199_000)
		expect(timers.finished.value).toEqual([])
		expect(alarm.ringing).toBe(false)

		vi.advanceTimersByTime(1000)
		expect(timers.running.value).toEqual([])
		expect(timers.finished.value).toEqual([timer])
		expect(alarm.vibrate).toHaveBeenCalledTimes(1)
		expect(alarm.ringing).toBe(true)

		timers.dismiss(timer.id)
		expect(timers.finished.value).toEqual([])
		expect(alarm.ringing).toBe(false)
	})

	it('keeps ringing while any finished timer is still shown', () => {
		const {timers, alarm} = openCookingMode()
		const short = timers.start(1, {label: '1 min', seconds: 60})
		const long = timers.start(2, {label: '2 min', seconds: 120})

		vi.advanceTimersByTime(120_000)
		expect(timers.finished.value.map(t => t.id)).toEqual([short.id, long.id])
		expect(alarm.vibrate).toHaveBeenCalledTimes(2)

		timers.dismiss(short.id)
		expect(alarm.ringing).toBe(true)
		timers.dismiss(long.id)
		expect(alarm.ringing).toBe(false)
	})

	it('drops every timer and goes quiet when cooking mode closes', () => {
		const {timers, alarm, close} = openCookingMode()
		timers.start(1, {label: '1 min', seconds: 60})
		timers.start(2, {label: '5 min', seconds: 300})
		vi.advanceTimersByTime(60_000)
		expect(alarm.ringing).toBe(true)

		close()
		expect(timers.timers.value).toEqual([])
		expect(timers.finished.value).toEqual([])
		expect(alarm.ringing).toBe(false)

		vi.advanceTimersByTime(300_000)
		expect(alarm.vibrate).toHaveBeenCalledTimes(1)
		expect(vi.getTimerCount()).toBe(0)
	})
})
