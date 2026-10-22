// Mandarin's API as the plugin uses it. The transport is passed in: in
// Obsidian it's requestUrl (no CORS limits, works on mobile), in tests fetch.

export interface Settings {
	url: string // e.g. https://mandarin.example.com
	token: string // a Mandarin API token
	cfClientId: string // Cloudflare Access service token, if Mandarin is behind Zero Trust
	cfClientSecret: string
	recipeFolder: string
	defaultProjectId: number
	defaultDueTime: string // HH:MM for due dates written without a time
}

export const DEFAULT_SETTINGS: Settings = {
	url: '',
	token: '',
	cfClientId: '',
	cfClientSecret: '',
	recipeFolder: 'Recipes',
	defaultProjectId: 0,
	defaultDueTime: '17:00',
}

export interface Request {
	url: string
	method: string
	headers: Record<string, string>
	body?: string
}

export interface Response {
	status: number
	text: string
	arrayBuffer: ArrayBuffer
}

export type Transport = (request: Request) => Promise<Response>

export interface Task {
	id: number
	title: string
	done: boolean
	due_date: string | null // ISO; Vikunja's "no date" is 0001-01-01
	project_id: number
	description?: string
}

export interface Project {
	id: number
	title: string
	is_archived?: boolean
}

export interface Ingredient {
	group: string
	text: string
}

export interface Recipe {
	id: number
	title: string
	source_url: string
	description: string
	servings: number
	yields: string
	prep_minutes: number
	cook_minutes: number
	total_minutes: number
	ingredients: Ingredient[]
	steps: string[]
	tags: string[]
	notes: string
	image_ids: number[]
}

export class ApiError extends Error {
	constructor(readonly status: number, message: string) {
		super(message)
	}
}

/** Vikunja's "no due date". */
export function hasDate(iso: string | null | undefined): iso is string {
	return Boolean(iso) && !iso!.startsWith('0001-')
}

export class MandarinApi {
	constructor(private settings: () => Settings, private transport: Transport) {}

	/** The web address of a page in Mandarin, e.g. a task. */
	pageUrl(path: string): string {
		return this.settings().url.replace(/\/+$/, '') + path
	}

	private async request(method: string, path: string, body?: unknown, contentType = 'application/json'): Promise<Response> {
		const s = this.settings()
		if (!s.url || !s.token) throw new ApiError(0, 'Set the Mandarin address and API token in the plugin settings.')
		const headers: Record<string, string> = {
			'Authorization': `Bearer ${s.token}`,
			'Accept': 'application/json',
		}
		// Cloudflare Zero Trust: a service token gets apps through Access.
		if (s.cfClientId && s.cfClientSecret) {
			headers['CF-Access-Client-Id'] = s.cfClientId
			headers['CF-Access-Client-Secret'] = s.cfClientSecret
		}
		if (body !== undefined) headers['Content-Type'] = contentType
		const response = await this.transport({
			url: this.pageUrl('/api/v2' + path),
			method,
			headers,
			body: body === undefined ? undefined : JSON.stringify(body),
		})
		if (response.status >= 400) throw new ApiError(response.status, describe(response))
		// Cloudflare's login page instead of Mandarin: the service token is missing or not allowed.
		if (response.text.trimStart().startsWith('<')) {
			throw new ApiError(response.status, 'Got a web page instead of Mandarin: check the Cloudflare service token.')
		}
		return response
	}

	private async json<T>(method: string, path: string, body?: unknown, contentType?: string): Promise<T> {
		const response = await this.request(method, path, body, contentType)
		return (response.text ? JSON.parse(response.text) : undefined) as T
	}

	/** Every page of a list endpoint. */
	private async all<T>(path: string): Promise<T[]> {
		const out: T[] = []
		const join = path.includes('?') ? '&' : '?'
		for (let page = 1; ; page++) {
			const data = await this.json<{items?: T[] | null, total_pages?: number}>('GET', `${path}${join}page=${page}&per_page=100`)
			out.push(...(data.items ?? []))
			if (page >= (data.total_pages ?? 1)) return out
		}
	}

	projects(): Promise<Project[]> {
		return this.all<Project>('/projects')
	}

	/** Tasks matching a Vikunja filter (https://vikunja.io/docs/filters), by due date. */
	tasks(filter: string, timezone = Intl.DateTimeFormat().resolvedOptions().timeZone): Promise<Task[]> {
		const query = new URLSearchParams({filter_timezone: timezone})
		if (filter) query.set('filter', filter)
		query.append('sort_by', 'due_date')
		query.append('order_by', 'asc')
		return this.all<Task>(`/tasks?${query}`)
	}

	setDone(taskId: number, done: boolean): Promise<Task> {
		return this.json<Task>('PATCH', `/tasks/${taskId}`, {done}, 'application/merge-patch+json')
	}

	createTask(projectId: number, task: {title: string, description?: string, due_date?: string}): Promise<Task> {
		return this.json<Task>('POST', `/projects/${projectId}/tasks`, task)
	}

	recipes(): Promise<Recipe[]> {
		return this.all<Recipe>('/recipes')
	}

	async recipeImage(recipeId: number, imageId: number): Promise<ArrayBuffer> {
		const response = await this.request('GET', `/recipes/${recipeId}/images/${imageId}`)
		return response.arrayBuffer
	}
}

function describe(response: Response): string {
	try {
		const problem = JSON.parse(response.text) as {detail?: string, message?: string, title?: string}
		const reason = problem.detail || problem.message || problem.title
		if (reason) return `Mandarin answered ${response.status}: ${reason}`
	} catch {
		// not JSON
	}
	if (response.status === 401) return 'Mandarin refused the API token.'
	if (response.status === 403) return 'Not allowed: check the API token\'s permissions (or the Cloudflare service token).'
	return `Mandarin answered ${response.status}.`
}
