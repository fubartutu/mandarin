import {describe, expect, it} from 'vitest'

import {errorKeyFor, joinDictated, transcriptOf} from './dictation'

describe('joinDictated', () => {
	it('uses what was said when the field is empty', () => {
		expect(joinDictated('', ' buy milk ')).toBe('buy milk')
		expect(joinDictated('  ', 'buy milk')).toBe('buy milk')
	})

	it('adds what was said after what was typed, with one space', () => {
		expect(joinDictated('Call', 'the dentist tomorrow')).toBe('Call the dentist tomorrow')
		expect(joinDictated('Call  ', 'the dentist')).toBe('Call the dentist')
	})

	it('leaves the field alone when nothing was heard', () => {
		expect(joinDictated('Call', '  ')).toBe('Call')
	})
})

describe('transcriptOf', () => {
	it('joins every result, final and interim', () => {
		const result = (transcript: string) => [{transcript}]
		const results = [result('buy milk'), result(' and eggs')] as unknown as SpeechRecognitionResultList
		expect(transcriptOf(results)).toBe('buy milk and eggs')
	})
})

describe('errorKeyFor', () => {
	it('stays quiet for silence and stopping on purpose', () => {
		expect(errorKeyFor('no-speech')).toBeNull()
		expect(errorKeyFor('aborted')).toBeNull()
	})

	it('names the problem otherwise', () => {
		expect(errorKeyFor('not-allowed')).toBe('blocked')
		expect(errorKeyFor('service-not-allowed')).toBe('blocked')
		expect(errorKeyFor('audio-capture')).toBe('noMicrophone')
		expect(errorKeyFor('network')).toBe('network')
		expect(errorKeyFor('language-not-supported')).toBe('other')
	})
})
