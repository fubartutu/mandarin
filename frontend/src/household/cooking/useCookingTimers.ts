// Cooking mode's running timers. CookingView owns them (one call per view),
// so they live exactly as long as cooking mode is open.
//
// The state is plain, serialisable data: a timer stores when it ends (epoch
// ms), not a counter, so the time left is always worked out from the clock.
// A phone that slept for four minutes shows the right time on the next tick,
// and the state can be saved and restored as it is.

import {computed, onScopeDispose, ref, watch} from 'vue'

import {createTimerAlarm} from './alarm'
import type {DetectedTimer} from './timers'

export interface CookingTimer {
	id: string
	stepNumber: number
	label: string // as written in the step, e.g. "20–25 minutes"
	seconds: number // the full length
	endsAt: number // epoch ms
}

export interface RunningTimer extends CookingTimer {
	remaining: number // whole seconds left, rounded up
}

/** What a finished timer does to get attention. */
export interface TimerAlarm {
	/** Called from the tap that starts a timer: browsers only allow sound after one. */
	prepare(): void
	vibrate(): void
	startRinging(): void
	stopRinging(): void
	/** Cooking mode closed: stop and let go of the audio. */
	dispose(): void
}

let lastId = 0

export function useCookingTimers(options: {alarm?: TimerAlarm} = {}) {
	const alarm = options.alarm ?? createTimerAlarm()

	const timers = ref<CookingTimer[]>([]) // running
	const finished = ref<CookingTimer[]>([]) // done, banner not yet dismissed
	const now = ref(Date.now())

	function remaining(timer: CookingTimer): number {
		return Math.max(0, Math.ceil((timer.endsAt - now.value) / 1000))
	}

	const running = computed<RunningTimer[]>(() => timers.value
		.map(timer => ({...timer, remaining: remaining(timer)}))
		.sort((a, b) => a.endsAt - b.endsAt))

	function tick() {
		now.value = Date.now()
		const done = timers.value.filter(timer => timer.endsAt <= now.value)
		if (done.length === 0) return
		timers.value = timers.value.filter(timer => timer.endsAt > now.value)
		finished.value = [...finished.value, ...done.sort((a, b) => a.endsAt - b.endsAt)]
		alarm.vibrate()
	}

	// Tick once a second while anything is running.
	let interval: ReturnType<typeof setInterval> | undefined
	watch(() => timers.value.length > 0, active => {
		if (active && interval === undefined) interval = setInterval(tick, 1000)
		else if (!active && interval !== undefined) {
			clearInterval(interval)
			interval = undefined
		}
	}, {immediate: true, flush: 'sync'})

	// Ring while any banner is up.
	watch(() => finished.value.length > 0, ringing => {
		if (ringing) alarm.startRinging()
		else alarm.stopRinging()
	}, {flush: 'sync'})

	function runningOn(stepNumber: number, label: string): CookingTimer | undefined {
		return timers.value.find(timer => timer.stepNumber === stepNumber && timer.label === label)
	}

	function start(stepNumber: number, detected: Pick<DetectedTimer, 'label' | 'seconds'>): CookingTimer {
		alarm.prepare()
		const existing = runningOn(stepNumber, detected.label)
		if (existing) return existing
		now.value = Date.now()
		const timer: CookingTimer = {
			id: `${now.value.toString(36)}-${++lastId}`,
			stepNumber,
			label: detected.label,
			seconds: detected.seconds,
			endsAt: now.value + detected.seconds * 1000,
		}
		timers.value = [...timers.value, timer]
		return timer
	}

	function cancel(id: string) {
		timers.value = timers.value.filter(timer => timer.id !== id)
	}

	function dismiss(id: string) {
		finished.value = finished.value.filter(timer => timer.id !== id)
	}

	function clear() {
		timers.value = []
		finished.value = []
	}

	// Closing cooking mode ends every timer (no push, no service worker).
	onScopeDispose(() => {
		clear()
		if (interval !== undefined) clearInterval(interval)
		interval = undefined
		alarm.dispose()
	})

	return {timers, finished, running, remaining, runningOn, start, cancel, dismiss, clear}
}
