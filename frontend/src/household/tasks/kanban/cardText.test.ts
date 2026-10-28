import {describe, expect, it} from 'vitest'

import {commentText, firstLine} from './cardText'

describe('firstLine', () => {
	it('takes the first paragraph', () => {
		expect(firstLine('<p>Bins out by 7pm</p><p>Recycling too</p>')).toBe('Bins out by 7pm')
	})
	it('skips empty blocks and squashes spaces', () => {
		expect(firstLine('<p></p><p>  Call   the\nplumber </p>')).toBe('Call the plumber')
	})
	it('reads headings, list and checklist items', () => {
		expect(firstLine('<h2>Before Friday</h2><p>…</p>')).toBe('Before Friday')
		expect(firstLine('<ul data-type="taskList"><li data-checked="false"><p>Buy filters</p></li></ul>')).toBe('Buy filters')
	})
	it('takes a list item\'s own text, not its nested list', () => {
		expect(firstLine('<ul><li><p>Garden</p><ul><li>weeds</li></ul></li></ul>')).toBe('Garden')
	})
	it('is empty for no or empty descriptions', () => {
		expect(firstLine('')).toBe('')
		expect(firstLine(null)).toBe('')
		expect(firstLine('<p></p>')).toBe('')
	})
	it('handles plain text', () => {
		expect(firstLine('just some text')).toBe('just some text')
	})
})

describe('commentText', () => {
	it('strips the HTML', () => {
		expect(commentText('<p>Done the <strong>front</strong> yard</p>')).toBe('Done the front yard')
	})
	it('shortens long comments', () => {
		const long = '<p>' + 'word '.repeat(60) + '</p>'
		const text = commentText(long, 40)
		expect(text.length).toBeLessThanOrEqual(40)
		expect(text.endsWith('…')).toBe(true)
	})
})
