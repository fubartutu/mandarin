// Which page the app opens on: Home (the default), Shopping, Recipes or Menu.
// Each value is also the name of the page's route.
// Kept in the user's frontend_settings (free-form JSON on the server), so it
// follows them to every device. Chosen in Settings → General.

import {START_LOCATION, type Router, type RouteLocationNormalized} from 'vue-router'

import {useAuthStore} from '@/stores/auth'

export const START_PAGES = ['home', 'shopping', 'recipes', 'menu'] as const
export type StartPage = typeof START_PAGES[number]

/** The pages that hand over to the start page once someone has signed in. */
const SIGN_IN_ROUTES = new Set(['user.login', 'user.register', 'openid.auth'])

/** The start page saved in the settings; anything unknown counts as Home. */
export function startPageFrom(value: unknown): StartPage {
	return START_PAGES.includes(value as StartPage) ? value as StartPage : 'home'
}

/** Opening the app (or signing in), not clicking Home in the menu. */
export function isAppOpening(from: RouteLocationNormalized): boolean {
	return from === START_LOCATION || SIGN_IN_ROUTES.has(String(from.name))
}

/**
 * Sends "/" to the chosen start page when the app opens. Clicking Home
 * afterwards still shows Home. Registered after Vikunja's own guard, so
 * the login has been checked and the settings are loaded by now.
 */
export function registerStartPage(router: Router) {
	router.beforeEach((to, from) => {
		if (to.name !== 'home' || !isAppOpening(from)) return
		const page = startPageFrom(useAuthStore().settings.frontend_settings.household_start_page)
		if (page !== 'home') return {name: page}
	})
}
