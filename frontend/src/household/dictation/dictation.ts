// Dictation for text fields, through the browser's speech recognition
// (Web Speech API). Chrome on Android sends the audio to Google; Safari on
// iPhone uses Apple's Siri dictation. Where neither exists (Firefox), the mic
// button isn't shown and the keyboard's own mic still works.

// TypeScript's DOM types don't include the recognizer itself yet, so this is
// the part of it Mandarin uses.
export interface Recognition {
	lang: string
	interimResults: boolean
	continuous: boolean
	onresult: ((event: {results: SpeechRecognitionResultList}) => void) | null
	onerror: ((event: {error: string}) => void) | null
	onend: (() => void) | null
	start(): void
	stop(): void
	abort(): void
}

type RecognitionConstructor = new () => Recognition

function recognitionConstructor(): RecognitionConstructor | undefined {
	if (typeof window === 'undefined') {
		return undefined
	}
	// Safari and older Chrome only have the prefixed name.
	const w = window as unknown as {SpeechRecognition?: RecognitionConstructor, webkitSpeechRecognition?: RecognitionConstructor}
	return w.SpeechRecognition ?? w.webkitSpeechRecognition
}

export function isDictationSupported(): boolean {
	return recognitionConstructor() !== undefined
}

export function createRecognition(lang: string): Recognition | null {
	const Recognition = recognitionConstructor()
	if (Recognition === undefined) {
		return null
	}
	const recognition = new Recognition()
	recognition.lang = lang
	// Words appear while speaking, and listening stops at the first pause:
	// a task title is one sentence.
	recognition.interimResults = true
	recognition.continuous = false
	return recognition
}

/** What was typed before dictating, followed by what was said. */
export function joinDictated(before: string, said: string): string {
	const spoken = said.trim()
	if (spoken === '') {
		return before
	}
	if (before.trim() === '') {
		return spoken
	}
	return before.replace(/\s+$/, '') + ' ' + spoken
}

/** Everything heard so far in this session, final and interim. */
export function transcriptOf(results: SpeechRecognitionResultList): string {
	let text = ''
	for (let i = 0; i < results.length; i++) {
		text += results[i]![0]!.transcript
	}
	return text
}

/**
 * The household.dictation.error.* key to show for a recognition error, or
 * null when it isn't worth a message (silence, or stopped on purpose).
 */
export function errorKeyFor(code: string): string | null {
	switch (code) {
		case 'no-speech':
		case 'aborted':
			return null
		case 'not-allowed':
		case 'service-not-allowed':
			return 'blocked'
		case 'audio-capture':
			return 'noMicrophone'
		case 'network':
			return 'network'
		default:
			return 'other'
	}
}
