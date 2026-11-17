import {describe, expect, it} from 'vitest'

import {defaultMode, itemText, moveItem, newItem, parseTyped, totalMinutes} from './breakdown'

describe('defaultMode', () => {
	it('gives chunks to a subtask and steps to anything else', () => {
		expect(defaultMode({related_tasks: {parenttask: [{id: 1, title: 'Lawn'}]}})).toBe('chunks')
		expect(defaultMode({related_tasks: {subtask: [{id: 2, title: 'Scalp'}]}})).toBe('steps')
		expect(defaultMode({})).toBe('steps')
	})
})

describe('parseTyped', () => {
	it('takes one item per line without list markers', () => {
		const text = 'Scalp the lawn\n\n- Dethatch\n2. Core aerate\n * Top dress \n[ ] Fertilise\n[x] Water'
		expect(parseTyped(text).map(i => i.title)).toEqual(['Scalp the lawn', 'Dethatch', 'Core aerate', 'Top dress', 'Fertilise', 'Water'])
	})

	it('gives each item its own key', () => {
		const [a, b] = parseTyped('a\nb')
		expect(a!.key).not.toBe(b!.key)
	})
})

describe('itemText', () => {
	it('keeps the estimate on chunks only', () => {
		const item = newItem(' Rake one strip ', 10)
		expect(itemText(item, 'chunks')).toBe('Rake one strip (~10 min)')
		expect(itemText(item, 'steps')).toBe('Rake one strip')
		expect(itemText(newItem('Rake', 0), 'chunks')).toBe('Rake')
	})

	it("doesn't add a second estimate", () => {
		expect(itemText(newItem('Rake (~5 min)', 10), 'chunks')).toBe('Rake (~5 min)')
	})
})

describe('moveItem', () => {
	it('moves up and down within the list', () => {
		expect(moveItem(['a', 'b', 'c'], 2, -1)).toEqual(['a', 'c', 'b'])
		expect(moveItem(['a', 'b', 'c'], 0, 1)).toEqual(['b', 'a', 'c'])
		expect(moveItem(['a', 'b'], 0, -1)).toEqual(['a', 'b'])
		expect(moveItem(['a', 'b'], 1, 1)).toEqual(['a', 'b'])
	})
})

it('totalMinutes adds up the estimates', () => {
	expect(totalMinutes([newItem('a', 10), newItem('b', 0), newItem('c', 25)])).toBe(35)
})
