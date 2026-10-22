import {describe, expect, it} from 'vitest'

import type {Recipe} from './api'
import {blockFilter, dueTimestamp, lineToTask, linkedTask, noteLink, parseBlock, recipeMarkdown, safeFileName, withTaskLink} from './markdown'

describe('note line → task', () => {
	it('drops list markers and checkboxes', () => {
		expect(lineToTask('- [ ] Ring the plumber')).toEqual({title: 'Ring the plumber', due: null})
		expect(lineToTask('  1. Book flights')).toEqual({title: 'Book flights', due: null})
		expect(lineToTask('Plain line')).toEqual({title: 'Plain line', due: null})
		expect(lineToTask('- [ ]   ')).toBeNull()
	})

	it('reads a due date in Tasks or Dataview style', () => {
		expect(lineToTask('- [ ] Renew rego 📅 2026-11-03')).toEqual({title: 'Renew rego', due: '2026-11-03'})
		expect(lineToTask('- Pay rates due:: 2026-11-15 urgently')).toEqual({title: 'Pay rates urgently', due: '2026-11-15'})
	})

	it('spots a line that already has a task, and adds the link', () => {
		const line = withTaskLink('- [ ] Ring the plumber  ', 'https://mandarin.example.com/tasks/12')
		expect(line).toBe('- [ ] Ring the plumber [↗ Mandarin](https://mandarin.example.com/tasks/12)')
		expect(linkedTask(line)).toBe('https://mandarin.example.com/tasks/12')
		expect(linkedTask('- [ ] Nothing here')).toBeNull()
	})

	it('builds due times and note links', () => {
		const iso = dueTimestamp('2026-11-03', '09:30')
		const d = new Date(iso)
		expect([d.getFullYear(), d.getMonth() + 1, d.getDate(), d.getHours(), d.getMinutes()]).toEqual([2026, 11, 3, 9, 30])
		expect(noteLink('HOME', 'Projects/Car.md')).toBe('obsidian://open?vault=HOME&file=Projects%2FCar.md')
	})
})

describe('mandarin-tasks block', () => {
	it('reads options and builds the filter', () => {
		const options = parseBlock('project: Inbox\nfilter: due_date < now+7d  # this week\nlimit: 5\n')
		expect(options).toEqual({project: 'Inbox', filter: 'due_date < now+7d', limit: 5, showDone: false})
		expect(blockFilter(options, 3)).toBe('done = false && project = 3 && (due_date < now+7d)')
		expect(blockFilter(parseBlock('show-done: yes'), null), 'done tasks too: no filter at all').toBe('')
		expect(blockFilter(parseBlock(''), null)).toBe('done = false')
	})
})

describe('recipe → note', () => {
	const recipe: Recipe = {
		id: 7, title: 'Chop Suey: Chicken Stir Fry', source_url: 'https://www.recipetineats.com/chicken-stir-fry/',
		description: 'Quick and veggie loaded.', servings: 2, yields: '2 servings', prep_minutes: 9, cook_minutes: 6, total_minutes: 75,
		ingredients: [{group: '', text: '180g chicken breast'}, {group: 'Sauce', text: '1 tbsp cornflour'}, {group: 'Sauce', text: '1 tbsp oyster sauce'}],
		steps: ['Prep:', 'Slice the chicken.', 'Cook:', 'Stir fry for 1 minute.'], tags: ['Mains', 'American Chinese'], notes: 'Double the sauce.', image_ids: [3],
	}

	it('writes frontmatter, ingredients by group and a numbered method', () => {
		expect(recipeMarkdown(recipe, 'https://mandarin.example.com/recipes/7', '![[Recipes/attachments/chop-suey.jpg]]')).toBe(`---
mandarin: "https://mandarin.example.com/recipes/7"
source: "https://www.recipetineats.com/chicken-stir-fry/"
servings: 2
yields: "2 servings"
prep: "9 min"
cook: "6 min"
total: "1 h 15 min"
tags: [recipe, mains, american-chinese]
---

# Chop Suey: Chicken Stir Fry

![[Recipes/attachments/chop-suey.jpg]]

Quick and veggie loaded.

## Ingredients

- 180g chicken breast

### Sauce

- 1 tbsp cornflour
- 1 tbsp oyster sauce

## Method

### Prep

1. Slice the chicken.

### Cook

2. Stir fry for 1 minute.

## Notes

Double the sauce.
`)
	})

	it('makes file names safe', () => {
		expect(safeFileName('Chop Suey: Chicken / Stir Fry?')).toBe('Chop Suey Chicken Stir Fry')
	})
})
