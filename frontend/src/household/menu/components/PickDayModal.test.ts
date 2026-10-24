import {afterEach, beforeEach, describe, expect, it, vi, type MockInstance} from 'vitest'
import {flushPromises, mount} from '@vue/test-utils'

import Card from '@/components/misc/Card.vue'
import Modal from '@/components/misc/Modal.vue'
import PickDayModal from './PickDayModal.vue'

let showModalSpy: MockInstance<HTMLDialogElement['showModal']>

beforeEach(() => {
	// jsdom has <dialog> but not showModal().
	showModalSpy = vi.spyOn(HTMLDialogElement.prototype, 'showModal').mockImplementation(() => {})
})

afterEach(() => {
	showModalSpy.mockRestore()
	document.body.innerHTML = ''
})

async function mountModal(props: {day: string, busy?: boolean}) {
	const wrapper = mount(PickDayModal, {
		attachTo: document.body,
		props: {title: 'Plan Soup', confirmLabel: 'Plan for dinner', ...props},
		global: {
			mocks: {$t: (key: string) => key},
			components: {Modal, Card},
			stubs: {
				Icon: true,
				// Like the real XButton, the click listener falls through to <button>.
				XButton: {
					props: ['disabled'],
					template: '<button type="button" :disabled="disabled"><slot /></button>',
				},
			},
		},
	})
	await flushPromises()
	return wrapper
}

// The modal is teleported to <body>, so it's looked up there.
const dateInput = () => document.querySelector<HTMLInputElement>('input[type="date"]')!
const button = (text: string) => [...document.querySelectorAll('button')].find(b => b.textContent?.trim() === text)!

describe('PickDayModal', () => {
	it('starts on the given day and emits the day picked', async () => {
		const wrapper = await mountModal({day: '2026-10-24'})
		expect(dateInput().value).toBe('2026-10-24')

		dateInput().value = '2026-10-29'
		dateInput().dispatchEvent(new Event('input'))
		button('Plan for dinner').click()

		expect(wrapper.emitted('pick')).toEqual([['2026-10-29']])
	})

	it('does not confirm without a day, or while busy', async () => {
		const wrapper = await mountModal({day: '2026-10-24', busy: true})
		expect(button('Plan for dinner').disabled).toBe(true)

		await wrapper.setProps({busy: false})
		dateInput().value = ''
		dateInput().dispatchEvent(new Event('input'))
		await flushPromises()
		expect(button('Plan for dinner').disabled).toBe(true)
		expect(wrapper.emitted('pick')).toBeUndefined()
	})

	it('emits close on Cancel', async () => {
		const wrapper = await mountModal({day: '2026-10-24'})
		button('household.menu.cancel').click()
		expect(wrapper.emitted('close')).toHaveLength(1)
	})
})
