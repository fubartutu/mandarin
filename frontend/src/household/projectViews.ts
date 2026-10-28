// Each project opens on the view it was last shown in (List, Kanban, …).
//
// Vikunja already remembers this, but only in the browser's localStorage
// (helpers/projectView.ts), which logging out clears and which other devices
// don't share. Here the last view per project is also kept in the user's
// frontend_settings on the server, like the start page (startPage.ts):
// - opening a project with no view in localStorage (view id 0) uses the
//   saved one; a view that no longer exists falls back to Vikunja's default
// - showing a view saves it, once it has stayed for a moment and only when
//   it changed, so flicking through views makes one write (SQLite)

import type {Router} from 'vue-router'

import {queryClient} from '@/client/queryClient'
import {updateFrontendSettingsMutationOptions} from '@/client/queries/account'
import {useAuthStore} from '@/stores/auth'

export type ProjectViews = Record<string, number>

const SAVE_DELAY = 1500 // ms

/** The saved views, tolerating anything odd in the settings JSON. */
export function savedViews(settings: {household_project_views?: unknown} | undefined): ProjectViews {
	const value = settings?.household_project_views
	if (!value || typeof value !== 'object') return {}
	return Object.fromEntries(Object.entries(value as Record<string, unknown>)
		.filter((entry): entry is [string, number] => Number.isInteger(entry[1]) && (entry[1] as number) > 0))
}

/** The saved view for a project, or 0 for none. */
export function savedViewFor(views: ProjectViews, projectId: number): number {
	return views[String(projectId)] ?? 0
}

/** The views with this one set, or null when it's already saved (nothing to write). */
export function withView(views: ProjectViews, projectId: number, viewId: number): ProjectViews | null {
	if (projectId <= 0 || viewId <= 0 || savedViewFor(views, projectId) === viewId) return null
	return {...views, [String(projectId)]: viewId}
}

export function registerProjectViewMemory(router: Router) {
	let timer: ReturnType<typeof setTimeout> | null = null

	// Vikunja's redirect gives view id 0 when localStorage has nothing for the project.
	router.beforeEach(to => {
		if (to.name !== 'project.view' || Number(to.params.viewId) !== 0) return
		const auth = useAuthStore()
		if (!auth.authenticated || auth.isLinkShareAuth) return
		const viewId = savedViewFor(savedViews(auth.settings.frontend_settings), Number(to.params.projectId))
		if (viewId) return {name: 'project.view', params: {...to.params, viewId: String(viewId)}, query: to.query, hash: to.hash, replace: true}
	})

	router.afterEach(to => {
		if (to.name !== 'project.view') return
		const projectId = Number(to.params.projectId)
		const viewId = Number(to.params.viewId)
		if (timer) clearTimeout(timer)
		timer = setTimeout(() => {
			timer = null
			save(projectId, viewId)
		}, SAVE_DELAY)
	})
}

async function save(projectId: number, viewId: number) {
	const auth = useAuthStore()
	const session = auth.session
	if (!auth.authenticated || auth.isLinkShareAuth || !session) return
	const views = withView(savedViews(auth.settings.frontend_settings), projectId, viewId)
	if (!views) return
	try {
		// Vikunja's own settings mutation (merge patch, cache update), run outside a component.
		await queryClient.getMutationCache()
			.build(queryClient, updateFrontendSettingsMutationOptions())
			.execute({id: session.id, type: session.type, frontendSettings: {household_project_views: views}})
	} catch {
		// Only a convenience: Vikunja's localStorage copy still works on this device.
	}
}
