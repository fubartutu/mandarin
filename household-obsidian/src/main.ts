// Mandarin for Obsidian: your tasks in a sidebar and in notes, note lines
// turned into tasks, and recipes saved as notes. See README.md.

import {
	App,
	Editor,
	FuzzySuggestModal,
	ItemView,
	Notice,
	Plugin,
	PluginSettingTab,
	Setting,
	TFile,
	WorkspaceLeaf,
	normalizePath,
	requestUrl,
} from 'obsidian'

import {ApiError, DEFAULT_SETTINGS, MandarinApi, hasDate, type Project, type Recipe, type Settings, type Task, type Transport} from './api'
import {blockFilter, dueTimestamp, lineToTask, linkedTask, noteLink, parseBlock, recipeMarkdown, safeFileName, withTaskLink} from './markdown'

const VIEW_TYPE = 'mandarin-tasks'

// Obsidian's requestUrl: not limited by CORS, works on mobile.
const transport: Transport = async request => {
	const response = await requestUrl({...request, throw: false})
	return {status: response.status, text: response.text, arrayBuffer: response.arrayBuffer}
}

function errorText(e: unknown): string {
	return e instanceof ApiError || e instanceof Error ? e.message : String(e)
}

export default class MandarinPlugin extends Plugin {
	settings: Settings = {...DEFAULT_SETTINGS}
	api = new MandarinApi(() => this.settings, transport)

	async onload() {
		this.settings = {...DEFAULT_SETTINGS, ...(await this.loadData())}
		this.addSettingTab(new MandarinSettingTab(this.app, this))

		this.registerView(VIEW_TYPE, leaf => new TasksView(leaf, this))
		this.addRibbonIcon('check-square', 'Mandarin tasks', () => this.openTasks())
		this.addCommand({id: 'open-tasks', name: 'Open tasks', callback: () => this.openTasks()})

		this.addCommand({
			id: 'line-to-task',
			name: 'Create task from this line',
			editorCallback: (editor, view) => this.lineToTask(editor, view),
		})
		this.registerEvent(this.app.workspace.on('editor-menu', (menu, editor, view) => {
			menu.addItem(item => item
				.setTitle('Create Mandarin task')
				.setIcon('check-square')
				.onClick(() => this.lineToTask(editor, view)))
		}))

		this.addCommand({id: 'import-recipe', name: 'Save a recipe as a note', callback: () => new RecipeModal(this).open()})

		this.registerMarkdownCodeBlockProcessor('mandarin-tasks', async (source, el) => {
			await this.renderBlock(source, el)
		})
	}

	async saveSettings() {
		await this.saveData(this.settings)
	}

	async openTasks() {
		const existing = this.app.workspace.getLeavesOfType(VIEW_TYPE)[0]
		const leaf = existing ?? this.app.workspace.getRightLeaf(false)
		if (!leaf) return
		await leaf.setViewState({type: VIEW_TYPE, active: true})
		this.app.workspace.revealLeaf(leaf)
	}

	// ---- Note line → task ----

	async lineToTask(editor: Editor, view: {file: TFile | null}) {
		const lineNo = editor.getCursor().line
		const line = editor.getLine(lineNo)
		const existing = linkedTask(line)
		if (existing) {
			new Notice('This line already has a Mandarin task.')
			return
		}
		const parsed = lineToTask(line)
		if (!parsed) {
			new Notice('Put the cursor on a line with some text.')
			return
		}
		const projectId = this.settings.defaultProjectId
		if (!projectId) {
			new Notice('Pick a project for new tasks in the Mandarin plugin settings.')
			return
		}
		const file = view.file
		try {
			const task = await this.api.createTask(projectId, {
				title: parsed.title,
				description: file ? `<p>From <a href="${noteLink(this.app.vault.getName(), file.path)}">${file.basename}</a> in Obsidian.</p>` : undefined,
				due_date: parsed.due ? dueTimestamp(parsed.due, this.settings.defaultDueTime) : undefined,
			})
			// The cursor may have moved while waiting; edit the same line.
			if (editor.getLine(lineNo) === line) {
				editor.setLine(lineNo, withTaskLink(line, this.api.pageUrl(`/tasks/${task.id}`)))
			}
			new Notice(`Task created: ${task.title}`)
		} catch (e) {
			new Notice(`Couldn't create the task. ${errorText(e)}`)
		}
	}

	// ---- mandarin-tasks code block ----

	async renderBlock(source: string, el: HTMLElement) {
		const options = parseBlock(source)
		el.addClass('mandarin-block')
		const status = el.createDiv({cls: 'mandarin-muted', text: 'Loading tasks…'})
		try {
			let projectId: number | null = null
			if (options.project) {
				const projects = await this.api.projects()
				const wanted = options.project.toLowerCase()
				const found = projects.find(p => String(p.id) === options.project || p.title.toLowerCase() === wanted)
				if (!found) throw new Error(`No project called "${options.project}".`)
				projectId = found.id
			}
			const tasks = (await this.api.tasks(blockFilter(options, projectId))).slice(0, options.limit)
			status.remove()
			if (options.title) el.createEl('h4', {text: options.title})
			renderTasks(el, tasks, this, () => this.renderBlockAgain(source, el))
		} catch (e) {
			status.setText(`Mandarin: ${errorText(e)}`)
		}
	}

	private async renderBlockAgain(source: string, el: HTMLElement) {
		el.empty()
		await this.renderBlock(source, el)
	}

	// ---- Recipe → note ----

	async saveRecipe(recipe: Recipe) {
		const folder = normalizePath(this.settings.recipeFolder || 'Recipes')
		const path = normalizePath(`${folder}/${safeFileName(recipe.title)}.md`)
		const existing = this.app.vault.getAbstractFileByPath(path)
		if (existing instanceof TFile) {
			new Notice('Already in your vault. Delete the note to save it again.')
			await this.app.workspace.getLeaf(false).openFile(existing)
			return
		}
		try {
			await this.ensureFolder(folder)
			let imageLink = ''
			if (recipe.image_ids.length > 0) {
				const attachments = normalizePath(`${folder}/attachments`)
				await this.ensureFolder(attachments)
				const imagePath = normalizePath(`${attachments}/${safeFileName(recipe.title)}.jpg`)
				if (!this.app.vault.getAbstractFileByPath(imagePath)) {
					const data = await this.api.recipeImage(recipe.id, recipe.image_ids[0])
					await this.app.vault.createBinary(imagePath, data)
				}
				imageLink = `![[${imagePath}]]`
			}
			const note = await this.app.vault.create(path, recipeMarkdown(recipe, this.api.pageUrl(`/recipes/${recipe.id}`), imageLink))
			await this.app.workspace.getLeaf(false).openFile(note)
			new Notice(`Saved ${recipe.title}`)
		} catch (e) {
			new Notice(`Couldn't save the recipe. ${errorText(e)}`)
		}
	}

	private async ensureFolder(path: string) {
		if (!this.app.vault.getAbstractFileByPath(path)) await this.app.vault.createFolder(path)
	}
}

// ---- Task list (sidebar and code blocks) ----

function dueText(task: Task): string {
	if (!hasDate(task.due_date)) return ''
	const due = new Date(task.due_date)
	const today = new Date()
	const day = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
	const diff = Math.round((day(due) - day(today)) / 86_400_000)
	const time = due.toLocaleTimeString([], {hour: 'numeric', minute: '2-digit'})
	if (diff === 0) return `today ${time}`
	if (diff === 1) return `tomorrow ${time}`
	if (diff === -1) return 'yesterday'
	if (diff < 0) return `${-diff} days ago`
	return due.toLocaleDateString([], {weekday: 'short', day: 'numeric', month: 'short'})
}

function renderTasks(el: HTMLElement, tasks: Task[], plugin: MandarinPlugin, refresh: () => void) {
	if (tasks.length === 0) {
		el.createDiv({cls: 'mandarin-muted', text: 'Nothing here.'})
		return
	}
	const list = el.createEl('ul', {cls: 'mandarin-tasks'})
	const now = Date.now()
	for (const task of tasks) {
		const li = list.createEl('li', {cls: 'mandarin-task'})
		const box = li.createEl('input', {type: 'checkbox'})
		box.checked = task.done
		box.addEventListener('change', async () => {
			box.disabled = true
			try {
				await plugin.api.setDone(task.id, box.checked)
				li.toggleClass('is-done', box.checked)
				window.setTimeout(refresh, 800)
			} catch (e) {
				box.checked = !box.checked
				new Notice(`Couldn't update the task. ${errorText(e)}`)
			} finally {
				box.disabled = false
			}
		})
		const link = li.createEl('a', {text: task.title, href: plugin.api.pageUrl(`/tasks/${task.id}`)})
		link.addClass('mandarin-task-title')
		const due = dueText(task)
		if (due) {
			const overdue = hasDate(task.due_date) && new Date(task.due_date).getTime() < now && !task.done
			li.createSpan({cls: overdue ? 'mandarin-due is-overdue' : 'mandarin-due', text: due})
		}
	}
}

class TasksView extends ItemView {
	private timer = 0

	constructor(leaf: WorkspaceLeaf, private plugin: MandarinPlugin) {
		super(leaf)
	}

	getViewType() {
		return VIEW_TYPE
	}

	getDisplayText() {
		return 'Mandarin tasks'
	}

	getIcon() {
		return 'check-square'
	}

	async onOpen() {
		await this.refresh()
		// Every five minutes while open.
		this.timer = window.setInterval(() => this.refresh(), 5 * 60_000)
		this.registerInterval(this.timer)
	}

	async refresh() {
		const root = this.contentEl
		root.empty()
		root.addClass('mandarin-view')
		const header = root.createDiv({cls: 'mandarin-view-header'})
		header.createEl('h4', {text: 'Mandarin'})
		const reload = header.createEl('button', {text: 'Refresh'})
		reload.addEventListener('click', () => this.refresh())

		const sections: [string, string][] = [
			['Overdue', 'done = false && due_date < now/d'],
			['Today', 'done = false && due_date >= now/d && due_date < now/d+1d'],
			['Next 7 days', 'done = false && due_date >= now/d+1d && due_date < now/d+8d'],
		]
		for (const [title, filter] of sections) {
			const section = root.createDiv({cls: 'mandarin-section'})
			section.createEl('h5', {text: title})
			try {
				renderTasks(section, await this.plugin.api.tasks(filter), this.plugin, () => this.refresh())
			} catch (e) {
				section.createDiv({cls: 'mandarin-muted', text: errorText(e)})
				break // the same problem would repeat for every section
			}
		}
	}
}

// ---- Recipe picker ----

class RecipeModal extends FuzzySuggestModal<Recipe> {
	private recipes: Recipe[] = []

	constructor(private plugin: MandarinPlugin) {
		super(plugin.app)
		this.setPlaceholder('Loading recipes…')
		plugin.api.recipes().then(recipes => {
			this.recipes = recipes
			this.setPlaceholder('Pick a recipe to save as a note')
			// Re-run the search so the list appears.
			this.inputEl.dispatchEvent(new Event('input'))
		}).catch(e => {
			new Notice(`Couldn't load recipes. ${errorText(e)}`)
			this.close()
		})
	}

	getItems() {
		return this.recipes
	}

	getItemText(recipe: Recipe) {
		return [recipe.title, ...recipe.tags].join(' · ')
	}

	onChooseItem(recipe: Recipe) {
		this.plugin.saveRecipe(recipe)
	}
}

// ---- Settings ----

class MandarinSettingTab extends PluginSettingTab {
	private projects: Project[] = []

	constructor(app: App, private plugin: MandarinPlugin) {
		super(app, plugin)
	}

	display() {
		const {containerEl} = this
		const s = this.plugin.settings
		containerEl.empty()

		const text = (name: string, desc: string, key: keyof Settings, secret = false) =>
			new Setting(containerEl).setName(name).setDesc(desc).addText(input => {
				input.setValue(String(s[key])).onChange(async value => {
					(s as unknown as Record<string, string>)[key] = value.trim()
					await this.plugin.saveSettings()
				})
				if (secret) input.inputEl.type = 'password'
			})

		text('Mandarin address', 'The address you open Mandarin at, e.g. https://mandarin.example.com', 'url')
		text('API token', 'Mandarin → Settings → API tokens. Give it: projects (read all), tasks (read all, create, update), recipes (read all, read one, images).', 'token', true)
		new Setting(containerEl).setHeading().setName('Cloudflare Zero Trust')
		text('Service token client ID', 'Cloudflare Zero Trust → Access → Service credentials. Leave empty if Mandarin isn\'t behind Access.', 'cfClientId')
		text('Service token client secret', '', 'cfClientSecret', true)

		new Setting(containerEl).setHeading().setName('Tasks and recipes')
		new Setting(containerEl)
			.setName('Project for new tasks')
			.setDesc('Where "Create task from this line" puts tasks.')
			.addDropdown(dropdown => {
				dropdown.addOption('0', this.projects.length ? 'Choose…' : 'Test the connection to load projects')
				for (const p of this.projects.filter(p => !p.is_archived)) dropdown.addOption(String(p.id), p.title)
				dropdown.setValue(String(s.defaultProjectId)).onChange(async value => {
					s.defaultProjectId = Number(value)
					await this.plugin.saveSettings()
				})
			})
		text('Due time', 'For due dates written without a time (📅 2026-11-03), as HH:MM.', 'defaultDueTime')
		text('Recipe folder', 'Where saved recipes go; pictures go in its attachments folder.', 'recipeFolder')

		new Setting(containerEl)
			.setName('Test the connection')
			.addButton(button => button.setButtonText('Test').onClick(async () => {
				try {
					this.projects = await this.plugin.api.projects()
					new Notice(`Connected: ${this.projects.length} projects.`)
					this.display()
				} catch (e) {
					new Notice(`Mandarin: ${errorText(e)}`)
				}
			}))
	}
}
