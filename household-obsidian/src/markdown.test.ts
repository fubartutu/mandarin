import {describe, expect, it} from 'vitest'

import type {Recipe} from './api'
import {
	blockFilter,
	dueTimestamp,
	fillTemplate,
	finishedFilter,
	finishedList,
	lineToTask,
	linkedTask,
	noteLink,
	parseBlock,
	recipeMarkdown,
	safeFileName,
	withFinishedTasks,
	withSubProjects,
	withTaskLink,
} from './markdown'

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

describe('finished tasks → daily note', () => {
	it('counts sub-projects, however deep', () => {
		const projects = [
			{id: 1, title: 'Home'},
			{id: 2, title: 'Garden', parent_project_id: 1},
			{id: 3, title: 'Veggies', parent_project_id: 2},
			{id: 4, title: 'Work'},
		]
		expect(withSubProjects(projects, [1]).sort()).toEqual([1, 2, 3])
		expect(withSubProjects(projects, [4])).toEqual([4])
	})

	it('filters on the day finished and the projects', () => {
		expect(finishedFilter('2026-11-05', '2026-11-06', [1, 2]))
			.toBe('done_at >= 2026-11-05 && done_at < 2026-11-06 && project in 1, 2')
	})

	it('lists tasks, with project headings only when there are several', () => {
		const url = 'https://m.example/tasks/1'
		expect(finishedList([{project: 'Home', tasks: [{title: 'Fix [the] tap', url}]}, {project: 'Work', tasks: []}]))
			.toBe(`- [x] [Fix the tap](${url})`)
		expect(finishedList([{project: 'Home', tasks: [{title: 'A', url}]}, {project: 'Work', tasks: [{title: 'B', url}]}]))
			.toBe(`### Home\n- [x] [A](${url})\n\n### Work\n- [x] [B](${url})`)
		expect(finishedList([{project: 'Home', tasks: []}])).toBe('- Nothing finished.')
	})

	const heading = '## ✅ Done in Mandarin'
	const start = '%% mandarin-finished: start %%'
	const end = '%% mandarin-finished: end %%'

	it('adds the list at the end of a note without the heading', () => {
		expect(withFinishedTasks('# Today\n\nNotes\n\n', heading, '- [x] A'))
			.toBe(`# Today\n\nNotes\n\n${heading}\n${start}\n- [x] A\n${end}\n`)
		expect(withFinishedTasks('', heading, '- [x] A')).toBe(`${heading}\n${start}\n- [x] A\n${end}\n`)
	})

	it('puts the list under the heading when the note has it, keeping what follows', () => {
		const note = `# Today\n${heading}\n\n___\nMore`
		expect(withFinishedTasks(note, heading, '- [x] A')).toBe(`# Today\n${heading}\n${start}\n- [x] A\n${end}\n\n___\nMore`)
	})

	it('replaces only what is between the markers the next time', () => {
		const once = withFinishedTasks(`# Today\n${heading}\nafter`, heading, '- [x] A')
		const twice = withFinishedTasks(once, heading, '- [x] A\n- [x] B')
		expect(twice).toBe(`# Today\n${heading}\n${start}\n- [x] A\n- [x] B\n${end}\nafter`)
		expect(withFinishedTasks(twice, heading, '- [x] A\n- [x] B')).toBe(twice)
	})

	it('fills a daily-note template like Obsidian', () => {
		const format = (pattern: string) => `<${pattern}>`
		expect(fillTemplate('# {{date:dddd, D MMMM YYYY}} {{title}} {{date}} {{ time }} {{time:H}}', '2026-11-05', format))
			.toBe('# <dddd, D MMMM YYYY> 2026-11-05 <YYYY-MM-DD> <HH:mm> <H>')
	})
})
