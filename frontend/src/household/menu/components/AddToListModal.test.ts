import {afterEach, beforeEach, describe, expect, it, vi, type MockInstance} from 'vitest'
import {flushPromises, mount} from '@vue/test-utils'
import {QueryClient, VueQueryPlugin} from '@tanstack/vue-query'

import Card from '@/components/misc/Card.vue'
import Modal from '@/components/misc/Modal.vue'

// The recipe comes from a fake fetch; `fetchRecipe` decides what it answers.
const fetchRecipe = vi.fn()
vi.mock('../../recipes/queries', () => ({
	recipeQuery: (id: number) => ({
		queryKey: ['recipes', 'detail', id],
		queryFn: () => fetchRecipe(id),
	}),
}))

// t and $t write the key and its values, so the tests can see what was asked for.
// vi.hoisted runs before the (hoisted) vi.mock calls, so they can use it.
const {$t} = vi.hoisted(() => ({
	$t: (key: string, values?: Record<string, unknown>) => values ? `${key} ${JSON.stringify(values)}` : key,
}))
vi.mock('vue-i18n', async importOriginal => ({
	...await importOriginal<typeof import('vue-i18n')>(),
	useI18n: () => ({t: $t}),
}))

import AddToListModal from './AddToListModal.vue'

let showModalSpy: MockInstance<HTMLDialogElement['showModal']>

beforeEach(() => {
	// jsdom has <dialog> but not showModal().
	showModalSpy = vi.spyOn(HTMLDialogElement.prototype, 'showModal').mockImplementation(() => {})
	fetchRecipe.mockReset()
})

afterEach(() => {
	showModalSpy.mockRestore()
	document.body.innerHTML = ''
})

const soup = {
	id: 7,
	title: 'Soup',
	servings: 4,
	ingredients: [{group: '', text: '2 onions', quantity: 2, unit: '', name: 'onions', note: ''}],
}

async function mountModal(props: {recipeId: number, recipeTitle: string, servings: number}) {
	const queryClient = new QueryClient({defaultOptions: {queries: {retry: false}}})
	const wrapper = mount(AddToListModal, {
		attachTo: document.body,
		props,
		global: {
			plugins: [[VueQueryPlugin, {queryClient}]],
			mocks: {$t},
			components: {Modal, Card},
			stubs: {
				Icon: true,
				XButton: {template: '<button type="button"><slot /></button>'},
				// The picker has its own tests; here it only shows what it was given.
				IngredientPicker: {
					name: 'IngredientPicker',
					props: ['lines', 'factor'],
					emits: ['added'],
					template: '<div class="picker-stub" :data-lines="lines.length" :data-factor="factor" />',
				},
			},
		},
	})
	await flushPromises()
	return wrapper
}

const picker = () => document.querySelector<HTMLElement>('.picker-stub')
const button = (text: string) => [...document.querySelectorAll('button')].find(b => b.textContent?.trim() === text)!

describe('AddToListModal', () => {
	it('shows the recipe\'s ingredients scaled to the servings planned', async () => {
		fetchRecipe.mockResolvedValue(soup)
		await mountModal({recipeId: 7, recipeTitle: 'Soup', servings: 6})

		expect(document.body.textContent).toContain('household.menu.addToListTitle {"recipe":"Soup"}')
		expect(document.body.textContent).toContain('household.menu.servings')
		expect(picker()?.dataset.lines).toBe('1')
		expect(picker()?.dataset.factor).toBe('1.5')
	})

	it('leaves out the servings line when none are planned', async () => {
		fetchRecipe.mockResolvedValue(soup)
		await mountModal({recipeId: 7, recipeTitle: 'Soup', servings: 0})

		expect(document.body.textContent).not.toContain('household.menu.servings')
		expect(picker()?.dataset.factor).toBe('1')
	})

	it('says so while loading, and when the recipe could not be loaded', async () => {
		let fail: (e: Error) => void = () => {}
		fetchRecipe.mockReturnValue(new Promise((_, reject) => {
			fail = reject
		}))
		await mountModal({recipeId: 7, recipeTitle: 'Soup', servings: 6})
		expect(document.body.textContent).toContain('household.menu.loading')
		expect(picker()).toBeNull()

		fail(new Error('offline'))
		await flushPromises()
		expect(document.body.textContent).toContain('household.recipes.notLoaded')
		expect(picker()).toBeNull()
	})

	it('closes once the ingredients are added, or on Close', async () => {
		fetchRecipe.mockResolvedValue(soup)
		const wrapper = await mountModal({recipeId: 7, recipeTitle: 'Soup', servings: 6})

		wrapper.findComponent({name: 'IngredientPicker'}).vm.$emit('added')
		expect(wrapper.emitted('close')).toHaveLength(1)

		button('household.menu.close').click()
		expect(wrapper.emitted('close')).toHaveLength(2)
	})
})
