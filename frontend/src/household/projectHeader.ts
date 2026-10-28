// The top bar shows the project last opened (title, link, "⋯" menu) until a
// page clears it. Vikunja's ContentAuth.vue clears it for its own pages (Home,
// Upcoming, Labels, …) by route name; this does the same for Shopping,
// Recipes and Menu, without editing that list.

import type {Router} from 'vue-router'

import {useBaseStore} from '@/stores/base'
import {householdRoutes} from './routes'

const NO_PROJECT = new Set(householdRoutes.map(route => route.name))

export function registerProjectHeaderReset(router: Router) {
	router.afterEach(to => {
		if (to.name && NO_PROJECT.has(to.name)) useBaseStore().setCurrentProject(null)
	})
}
