// The plugin's text work, kept free of Obsidian so it can be tested.

import type {Recipe} from './api'

// ---- Note line → task ----

export interface LineTask {
	title: string
	due: string | null // YYYY-MM-DD
}

const LIST_MARKER = /^(\s*)([-*+]|\d+[.)])\s+(\[[ xX]\]\s+)?/
// "📅 2026-10-25" (the Tasks plugin), "due:: 2026-10-25" (Dataview) or "due: 2026-10-25".
const DUE = /\s*(?:📅\s*|\bdue::?\s*)(\d{4}-\d{2}-\d{2})\b/u
const MANDARIN_LINK = /\[[^\]]*\]\((https?:\/\/[^)\s]+\/tasks\/\d+)\)/

/** The task a note line describes: its text without list markers, and a due date if written. */
export function lineToTask(line: string): LineTask | null {
	let text = line.replace(LIST_MARKER, '')
	let due: string | null = null
	const m = DUE.exec(text)
	if (m) {
		due = m[1]
		text = text.replace(m[0], ' ')
	}
	const title = text.replace(/\s+/g, ' ').trim()
	return title ? {title, due} : null
}

/** Whether the line already links to a Mandarin task. */
export function linkedTask(line: string): string | null {
	return MANDARIN_LINK.exec(line)?.[1] ?? null
}

/** The line with a link to its new task added at the end. */
export function withTaskLink(line: string, taskUrl: string): string {
	return `${line.replace(/\s+$/, '')} [↗ Mandarin](${taskUrl})`
}

/** A date (and default time) as the ISO timestamp Mandarin wants, in local time. */
export function dueTimestamp(date: string, time: string): string {
	const [h, m] = (time.match(/^\d{1,2}:\d{2}$/) ? time : '17:00').split(':').map(Number)
	const [y, mo, d] = date.split('-').map(Number)
	return new Date(y, mo - 1, d, h, m).toISOString()
}

/** An obsidian:// link that opens the note. */
export function noteLink(vault: string, path: string): string {
	return `obsidian://open?vault=${encodeURIComponent(vault)}&file=${encodeURIComponent(path)}`
}

// ---- mandarin-tasks code block ----

export interface BlockOptions {
	project?: string // a title or id
	filter?: string // raw Vikunja filter
	limit: number
	showDone: boolean
	title?: string
}

/**
 * Reads a code block's options, one "key: value" per line:
 *
 *   ```mandarin-tasks
 *   project: Inbox
 *   filter: due_date < now+7d
 *   limit: 10
 *   ```
 */
export function parseBlock(source: string): BlockOptions {
	const options: BlockOptions = {limit: 50, showDone: false}
	for (const raw of source.split('\n')) {
		const line = raw.replace(/#.*$/, '').trim()
		const i = line.indexOf(':')
		if (i < 0) continue
		const key = line.slice(0, i).trim().toLowerCase()
		const value = line.slice(i + 1).trim()
		if (key === 'project') options.project = value
		else if (key === 'filter') options.filter = value
		else if (key === 'title') options.title = value
		else if (key === 'limit') options.limit = Math.max(1, Number(value) || 50)
		else if (key === 'show-done' || key === 'done') options.showDone = /^(true|yes|1)$/i.test(value)
	}
	return options
}

/** The Vikunja filter for a block: its own filter, the project, and open tasks only unless asked. */
export function blockFilter(options: BlockOptions, projectId: number | null): string {
	const parts: string[] = []
	if (!options.showDone) parts.push('done = false')
	if (projectId) parts.push(`project = ${projectId}`)
	if (options.filter) parts.push(`(${options.filter})`)
	return parts.join(' && ') // '' = every task
}

// ---- Recipe → note ----

/** A file name Obsidian accepts on every platform. */
export function safeFileName(title: string): string {
	return title.replace(/[\\/:*?"<>|#^[\]]/g, ' ').replace(/\s+/g, ' ').trim().slice(0, 120) || 'Recipe'
}

function minutes(m: number): string | null {
	if (!m) return null
	const h = Math.floor(m / 60)
	return [h ? `${h} h` : '', m % 60 ? `${m % 60} min` : ''].filter(Boolean).join(' ')
}

/** Obsidian tags can't have spaces: "American Chinese" → "american-chinese". */
function tag(text: string): string {
	return text.trim().toLowerCase().replace(/[^\p{L}\p{N}/_-]+/gu, '-').replace(/^-+|-+$/g, '')
}

function yamlString(value: string): string {
	return JSON.stringify(value) // a valid YAML double-quoted string
}

/** The recipe as a Markdown note. imageLink is an embed like ![[Recipes/attachments/x.jpg]], or ''. */
export function recipeMarkdown(recipe: Recipe, mandarinUrl: string, imageLink: string): string {
	const front: string[] = ['---', `mandarin: ${yamlString(mandarinUrl)}`]
	if (recipe.source_url) front.push(`source: ${yamlString(recipe.source_url)}`)
	if (recipe.servings) front.push(`servings: ${recipe.servings}`)
	if (recipe.yields) front.push(`yields: ${yamlString(recipe.yields)}`)
	for (const [key, value] of [['prep', recipe.prep_minutes], ['cook', recipe.cook_minutes], ['total', recipe.total_minutes]] as const) {
		const text = minutes(value)
		if (text) front.push(`${key}: ${yamlString(text)}`)
	}
	const tags = ['recipe', ...recipe.tags.map(tag).filter(Boolean)]
	front.push(`tags: [${[...new Set(tags)].join(', ')}]`, '---', '')

	const body: string[] = [`# ${recipe.title}`, '']
	if (imageLink) body.push(imageLink, '')
	if (recipe.description) body.push(recipe.description, '')

	body.push('## Ingredients', '')
	let group = ''
	for (const ingredient of recipe.ingredients) {
		if (ingredient.group !== group) {
			group = ingredient.group
			if (group) body.push('', `### ${group}`, '')
		}
		body.push(`- ${ingredient.text}`)
	}
	body.push('')

	if (recipe.steps.length) {
		body.push('## Method', '')
		let n = 0
		for (const step of recipe.steps) {
			// Section labels ("Sauce:") stay headings, like in Mandarin.
			if (step.endsWith(':') && !/^\d/.test(step)) body.push('', `### ${step.slice(0, -1)}`, '')
			else body.push(`${++n}. ${step}`)
		}
		body.push('')
	}
	if (recipe.notes) body.push('## Notes', '', recipe.notes, '')
	return [...front, ...body].join('\n').replace(/\n{3,}/g, '\n\n')
}
