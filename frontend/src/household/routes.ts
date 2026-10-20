import type {RouteRecordRaw} from 'vue-router'

// Spread into Vikunja's route table in src/router/index.ts (marked FORK).
export const householdRoutes: RouteRecordRaw[] = [
	{
		// `:listId?` is optional: /shopping opens the last-used list.
		path: '/shopping/:listId?',
		name: 'shopping',
		component: () => import('./shopping/ShoppingView.vue'),
		// Hand the param to the component as a number prop instead of a string.
		props: route => ({listId: Number(route.params.listId) || 0}),
	},
]
