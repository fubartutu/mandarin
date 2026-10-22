// Runs the plugin's API code against a real Mandarin server. Skipped unless
// MANDARIN_TEST_URL and MANDARIN_TEST_TOKEN are set (a token with the
// permissions the settings page asks for).

import {describe, expect, it} from 'vitest'

import {DEFAULT_SETTINGS, MandarinApi, type Transport} from './api'
import {dueTimestamp} from './markdown'

const url = process.env.MANDARIN_TEST_URL ?? ''
const token = process.env.MANDARIN_TEST_TOKEN ?? ''

// fetch, shaped like Obsidian's requestUrl.
const transport: Transport = async request => {
	const response = await fetch(request.url, {method: request.method, headers: request.headers, body: request.body})
	const buffer = await response.arrayBuffer()
	return {status: response.status, text: new TextDecoder().decode(buffer), arrayBuffer: buffer}
}

describe.skipIf(!url || !token)('against a Mandarin server', () => {
	const api = new MandarinApi(() => ({...DEFAULT_SETTINGS, url, token}), transport)

	it('lists projects, creates a task from a note line, finds it as due today, ticks it off', async () => {
		const projects = await api.projects()
		expect(projects.length).toBeGreaterThan(0)
		const inbox = projects.find(p => p.title === 'Inbox') ?? projects[0]

		const today = new Date()
		const date = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`
		const task = await api.createTask(inbox.id, {
			title: 'Ring the plumber',
			description: '<p>From <a href="obsidian://open?vault=HOME&file=Home.md">Home</a></p>',
			due_date: dueTimestamp(date, '23:59'),
		})
		expect(task.id).toBeGreaterThan(0)

		const dueToday = await api.tasks('done = false && due_date >= now/d && due_date < now/d+1d')
		expect(dueToday.map(t => t.title)).toContain('Ring the plumber')

		const done = await api.setDone(task.id, true)
		expect(done.done).toBe(true)
		const stillOpen = await api.tasks('done = false && due_date >= now/d && due_date < now/d+1d')
		expect(stillOpen.map(t => t.id)).not.toContain(task.id)
		expect(api.pageUrl(`/tasks/${task.id}`)).toBe(`${url}/tasks/${task.id}`)
	})

	it('reads recipes and downloads a picture', async () => {
		const recipes = await api.recipes()
		const pasta = recipes.find(r => r.title === 'Weeknight Pasta')
		expect(pasta?.ingredients.map(i => i.group)).toEqual(['', 'Sauce'])
		const image = await api.recipeImage(pasta!.id, pasta!.image_ids[0])
		const bytes = new Uint8Array(image)
		expect([bytes[0], bytes[1]]).toEqual([0xFF, 0xD8]) // a JPEG
	})

	it('explains a bad token', async () => {
		const bad = new MandarinApi(() => ({...DEFAULT_SETTINGS, url, token: 'tk_nope'}), transport)
		await expect(bad.projects()).rejects.toThrow(/401|token/i)
	})
})
