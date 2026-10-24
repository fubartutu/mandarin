import {describe, expect, it, vi} from 'vitest'

import {leaveCooking} from './leave'

function fakeRouter() {
	return {
		back: vi.fn(),
		replace: vi.fn(),
	}
}

describe('leaveCooking', () => {
	it('goes back to the in-app page cooking mode was opened from', () => {
		for (const back of ['/menu', '/', '/recipes/7']) {
			const router = fakeRouter()

			leaveCooking(router, 7, {back})

			expect(router.back).toHaveBeenCalledOnce()
			expect(router.replace).not.toHaveBeenCalled()
		}
	})

	it('opens the recipe page in its place when there is no page to go back to', () => {
		for (const state of [{back: null}, {}, null]) {
			const router = fakeRouter()

			leaveCooking(router, 7, state)

			expect(router.back).not.toHaveBeenCalled()
			expect(router.replace).toHaveBeenCalledWith({
				name: 'recipe',
				params: {recipeId: 7},
			})
		}
	})
})
