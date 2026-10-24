import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

import {createTimerAlarm, type AlarmEnv} from './alarm'

// A stand-in for the browser's Web Audio API that counts the beeps played.
function fakeBrowser(options: {audio?: boolean, vibrate?: boolean} = {}) {
	const beeps: number[] = []
	const contexts: {state: string, resume: () => Promise<void>, close: () => Promise<void>}[] = []
	class FakeAudioContext {
		state = 'suspended'
		currentTime = 0
		destination = {}
		constructor() {
			contexts.push(this)
		}
		resume = vi.fn(async () => { this.state = 'running' })
		close = vi.fn(async () => { this.state = 'closed' })
		createOscillator() {
			return {
				frequency: {value: 0},
				connect: () => {},
				start: () => beeps.push(Date.now()),
				stop: () => {},
			}
		}
		createGain() {
			return {
				gain: {setValueAtTime: () => {}, exponentialRampToValueAtTime: () => {}},
				connect: () => {},
			}
		}
	}
	const vibrate = vi.fn(() => true)
	const env = {
		AudioContext: options.audio === false ? undefined : FakeAudioContext,
		navigator: options.vibrate === false ? {} : {vibrate},
	} as unknown as AlarmEnv
	return {env, beeps, contexts, vibrate}
}

describe('createTimerAlarm', () => {
	beforeEach(() => vi.useFakeTimers())
	afterEach(() => vi.useRealTimers())

	it('vibrates in a long-short-long pattern', () => {
		const browser = fakeBrowser()
		createTimerAlarm(browser.env).vibrate()
		expect(browser.vibrate).toHaveBeenCalledWith([400, 200, 400])
	})

	it('wakes the sound up on the tap that starts a timer', () => {
		const browser = fakeBrowser()
		createTimerAlarm(browser.env).prepare()
		expect(browser.contexts[0].resume).toHaveBeenCalled()
	})

	it('beeps straight away and every 2 seconds until stopped', () => {
		const browser = fakeBrowser()
		const alarm = createTimerAlarm(browser.env)
		alarm.prepare()
		alarm.startRinging()
		const pips = browser.beeps.length
		expect(pips).toBeGreaterThan(0)

		vi.advanceTimersByTime(4000)
		expect(browser.beeps.length).toBe(pips * 3)

		alarm.stopRinging()
		vi.advanceTimersByTime(10_000)
		expect(browser.beeps.length).toBe(pips * 3)
	})

	it('lets the sound go when disposed', () => {
		const browser = fakeBrowser()
		const alarm = createTimerAlarm(browser.env)
		alarm.prepare()
		alarm.startRinging()
		alarm.dispose()
		expect(browser.contexts[0].close).toHaveBeenCalled()
		expect(vi.getTimerCount()).toBe(0)
	})

	it('stays silent, without errors, where there is no sound or vibration', () => {
		const browser = fakeBrowser({audio: false, vibrate: false})
		const alarm = createTimerAlarm(browser.env)
		expect(() => {
			alarm.prepare()
			alarm.vibrate()
			alarm.startRinging()
			vi.advanceTimersByTime(4000)
			alarm.stopRinging()
			alarm.dispose()
		}).not.toThrow()
	})
})
