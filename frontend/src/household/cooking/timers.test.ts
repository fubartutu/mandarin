import {describe, expect, it} from 'vitest'

import {findTimers, formatRemaining} from './timers'

describe('findTimers', () => {
	it('finds minutes written in full', () => {
		expect(findTimers('Simmer for 20 minutes.')).toEqual([
			{seconds: 1200, label: '20 minutes', index: 11},
		])
	})

	// [text, seconds, label]
	it.each([
		['Rest 1 minute.', 60, '1 minute'],
		['Cook 5 min, then serve.', 300, '5 min'],
		['Cook 5 mins.', 300, '5 mins'],
		['Bake 1 hour.', 3600, '1 hour'],
		['Bake 2 Hours.', 7200, '2 Hours'],
		['Braise 3 hrs.', 10800, '3 hrs'],
		['Braise 1 hr', 3600, '1 hr'],
		['Prove 2h somewhere warm.', 7200, '2h'],
		['Blitz 30 seconds.', 30, '30 seconds'],
		['Blitz 10 sec.', 10, '10 sec'],
		['Microwave 45s.', 45, '45s'],
		// fractions and decimals
		['Roast 1½ hours.', 5400, '1½ hours'],
		['Roast 1 1/2 hours.', 5400, '1 1/2 hours'],
		['Roast 1.5 hours.', 5400, '1.5 hours'],
		['Roast 2,5 h.', 9000, '2,5 h'],
		['Steam ¾ hour.', 2700, '¾ hour'],
		['Cook 1/2 hour.', 1800, '1/2 hour'],
		// compounds are one timer
		['Bake 1 hour 30 minutes.', 5400, '1 hour 30 minutes'],
		['Bake 1 hr 15 min.', 4500, '1 hr 15 min'],
		['Bake 1 hour and 10 minutes.', 4200, '1 hour and 10 minutes'],
		['Bake 2h30 until golden.', 9000, '2h30'],
		['Pulse 1 minute 30 seconds.', 90, '1 minute 30 seconds'],
		// ranges start at the lower bound
		['Simmer 20–25 minutes.', 1200, '20–25 minutes'],
		['Simmer 20-25 min.', 1200, '20-25 min'],
		['Simmer 20 to 25 minutes.', 1200, '20 to 25 minutes'],
		['Roast 1 – 1½ hours.', 3600, '1 – 1½ hours'],
		// words
		['Rest for an hour.', 3600, 'an hour'],
		['Chill for half an hour.', 1800, 'half an hour'],
		['Stir for a minute.', 60, 'a minute'],
		['Leave for an hour and a half.', 5400, 'an hour and a half'],
		['Simmer ten minutes.', 600, 'ten minutes'],
		['Rest for a half-hour.', 1800, 'a half-hour'],
		['Marinate two to three hours.', 7200, 'two to three hours'],
		['Go for a 20-minute walk.', 1200, '20-minute'],
	])('reads %j as %i seconds', (text, seconds, label) => {
		expect(findTimers(text)).toEqual([{seconds, label, index: text.indexOf(label)}])
	})

	it.each([
		'Heat the oven to 180°C.',
		'Preheat to 350 °F (175 C).',
		'Add 2 cups of stock and 1 tbsp oil.',
		'Stir in 2 heaped tablespoons of flour.',
		'Add 10 small potatoes and 4 shallots.',
		'Scatter over 3 mint leaves and 2 handfuls of spinach.',
		'Use a 20cm tin; cut into 8 slices.',
		'Cook for a few minutes until soft.',
		'Leave for minutes rather than hours.',
		'Season. Serves 4.',
		'Add a handful of herbs and a splash of milk.',
		'Beat in a second egg.',
		'Add the pizza minutely sliced.',
		'Add one to the pan at a time.',
	])('finds no time in %j', text => {
		expect(findTimers(text)).toEqual([])
	})

	it('offers each distinct time in a step, in order', () => {
		const text = 'Simmer 20 minutes, then bake at 200°C for 1 hour 10 min. Rest 5 min.'
		expect(findTimers(text)).toEqual([
			{seconds: 1200, label: '20 minutes', index: 7},
			{seconds: 4200, label: '1 hour 10 min', index: 42},
			{seconds: 300, label: '5 min', index: 62},
		])
	})

	it('offers a repeated time once', () => {
		expect(findTimers('Fry 5 minutes, turn and fry 5 min more.')).toEqual([
			{seconds: 300, label: '5 minutes', index: 4},
		])
	})
})

describe('formatRemaining', () => {
	it.each([
		[0, '00:00'],
		[9, '00:09'],
		[65, '01:05'],
		[1200, '20:00'],
		[3599, '59:59'],
		[3600, '1:00:00'],
		[5405, '1:30:05'],
	])('shows %i seconds as %s', (seconds, shown) => {
		expect(formatRemaining(seconds)).toBe(shown)
	})
})
