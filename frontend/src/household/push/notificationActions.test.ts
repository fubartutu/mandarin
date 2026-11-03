import {describe, expect, it, vi} from 'vitest'

import {isNotificationAction, runNotificationAction} from './notificationActions'

const scope = 'https://mandarin.example.com/'

function answer(status: number, type: ResponseType = 'basic', body: unknown = {result: 'done'}) {
	return vi.fn().mockResolvedValue({ok: status >= 200 && status < 300, status, type, json: async () => body})
}

describe('runNotificationAction', () => {
	it('posts the token and button to the server', async () => {
		const fetchFn = answer(200)
		expect(await runNotificationAction(scope, 'done', 'tok.sig', 'Bins', fetchFn)).toBeNull()
		expect(fetchFn).toHaveBeenCalledOnce()
		const [url, init] = fetchFn.mock.calls[0]
		expect(url).toBe('https://mandarin.example.com/api/v2/push/actions')
		expect(init.method).toBe('POST')
		expect(JSON.parse(init.body)).toEqual({token: 'tok.sig', action: 'done'})
		expect(init.redirect).toBe('manual')
	})

	it('confirms a snooze quietly', async () => {
		const now = new Date(2026, 10, 3, 9, 40)
		const until = new Date(2026, 10, 3, 10, 40)
		const fetchFn = answer(200, 'basic', {result: 'snoozed', snoozed_until: until.toISOString()})
		const next = await runNotificationAction(scope, 'snooze', 't', 'Bins', fetchFn, now)
		expect(next).toEqual({
			title: 'Bins',
			body: `Snoozed until ${until.toLocaleTimeString(undefined, {hour: 'numeric', minute: '2-digit'})}`,
			quiet: true,
		})
		// Already done elsewhere: nothing to confirm.
		expect(await runNotificationAction(scope, 'snooze', 't', 'Bins', answer(200, 'basic', {result: 'unchanged'}))).toBeNull()
	})

	it('says so when it fails', async () => {
		expect(await runNotificationAction(scope, 'done', 't', 'Bins', answer(500))).toEqual({
			title: 'Couldn\'t mark “Bins” done',
			body: 'Tap to open it.',
			quiet: false,
		})
		expect((await runNotificationAction(scope, 'snooze', 't', 'Bins', answer(401)))?.body)
			.toContain('too old')
		// A redirect to the Cloudflare login page comes back opaque, status 0.
		expect((await runNotificationAction(scope, 'snooze', 't', 'Bins', answer(0, 'opaqueredirect')))?.title)
			.toBe('Couldn\'t snooze “Bins”')
		expect((await runNotificationAction(scope, 'skip', 't', 'Stretch', answer(500)))?.title)
			.toBe('Couldn\'t skip “Stretch”')
		// A habit skipped (or already done) elsewhere: the notification just goes.
		expect(await runNotificationAction(scope, 'skip', 't', 'Stretch', answer(200, 'basic', {result: 'skipped'}))).toBeNull()
		const offline = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))
		expect(await runNotificationAction(scope, 'done', 't', 'Bins', offline)).not.toBeNull()
	})
})

describe('isNotificationAction', () => {
	it('knows our buttons only', () => {
		expect(isNotificationAction('done')).toBe(true)
		expect(isNotificationAction('snooze')).toBe(true)
		expect(isNotificationAction('skip')).toBe(true)
		expect(isNotificationAction('show-task')).toBe(false) // Vikunja's own
		expect(isNotificationAction('')).toBe(false) // a tap on the notification itself
	})
})
