// The sound and buzz of a finished cooking timer: a short beep every 2
// seconds made with the Web Audio API (no audio files), and one vibration.
// Both quietly do nothing where the browser can't (iPhones don't vibrate
// from the web).

import type {TimerAlarm} from './useCookingTimers'

export interface AlarmEnv {
	AudioContext?: typeof AudioContext
	navigator: Pick<Navigator, 'vibrate'> | Record<string, never>
}

const RING_EVERY_MS = 2000
const PIPS = [0, 0.2] // two short pips per beep, seconds from now
const PIP_LENGTH = 0.12

function browserEnv(): AlarmEnv {
	// Older Safari only has the prefixed name.
	const legacy = (window as unknown as {webkitAudioContext?: typeof AudioContext}).webkitAudioContext
	return {AudioContext: window.AudioContext ?? legacy, navigator}
}

export function createTimerAlarm(env: AlarmEnv = browserEnv()): TimerAlarm {
	let context: AudioContext | undefined
	let ringing: ReturnType<typeof setInterval> | undefined

	function audio(): AudioContext | undefined {
		if (!context && env.AudioContext) {
			try {
				context = new env.AudioContext()
			} catch {
				// No sound on this device.
			}
		}
		return context
	}

	function beep() {
		const ctx = audio()
		if (!ctx) return
		for (const offset of PIPS) {
			const at = ctx.currentTime + offset
			const oscillator = ctx.createOscillator()
			const gain = ctx.createGain()
			oscillator.frequency.value = 880
			gain.gain.setValueAtTime(0.3, at)
			gain.gain.exponentialRampToValueAtTime(0.001, at + PIP_LENGTH)
			oscillator.connect(gain)
			gain.connect(ctx.destination)
			oscillator.start(at)
			oscillator.stop(at + PIP_LENGTH)
		}
	}

	function stopRinging() {
		if (ringing !== undefined) clearInterval(ringing)
		ringing = undefined
	}

	return {
		// Browsers keep a page silent until a tap; resuming the audio during
		// the tap that starts a timer lets it beep later on its own.
		prepare() {
			const ctx = audio()
			if (ctx?.state === 'suspended') ctx.resume().catch(() => {})
		},
		vibrate() {
			if ('vibrate' in env.navigator && typeof env.navigator.vibrate === 'function') {
				env.navigator.vibrate([400, 200, 400])
			}
		},
		startRinging() {
			if (ringing !== undefined) return
			beep()
			ringing = setInterval(beep, RING_EVERY_MS)
		},
		stopRinging,
		dispose() {
			stopRinging()
			context?.close().catch(() => {})
			context = undefined
		},
	}
}
