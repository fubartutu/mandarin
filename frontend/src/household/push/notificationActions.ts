// The Done and Snooze buttons on task notifications, and Done and Skip on
// habit reminders (pkg/household/push, actions.go and habits.go). Runs in the service worker (household/sw.ts), which can't read
// the app's login token: the notification carries a signed action token, and
// that's all POST /api/v2/push/actions needs.
//
// A plain fetch, not the generated client: the client reads the login token
// and API address from the page, and a service worker has no page.

import {snoozeLabel} from './snoozeLabel'

/** The buttons the server sends, as the Web Notification API takes them. */
export interface NotificationButton {
	action: string
	title: string
}

export const NOTIFICATION_ACTIONS = ['done', 'snooze', 'skip'] as const
export type NotificationAction = typeof NOTIFICATION_ACTIONS[number]

export function isNotificationAction(action: string): action is NotificationAction {
	return (NOTIFICATION_ACTIONS as readonly string[]).includes(action)
}

/** A notification to show after the tap: a confirmation or what went wrong. */
export interface FollowUpNotification {
	title: string
	body: string
	/** True for a confirmation: shown without a sound or buzz. */
	quiet: boolean
}

/**
 * Sends the tapped button to the server. Returns what to show next: "Snoozed
 * until 10:40" after a snooze, a "couldn't" when it failed, or null (Done
 * or Skip worked, or the task or habit had already moved on: the notification just goes).
 */
export async function runNotificationAction(
	scope: string,
	action: NotificationAction,
	token: string,
	taskTitle: string,
	fetchFn: typeof fetch = fetch,
	now: Date = new Date(),
): Promise<FollowUpNotification | null> {
	let status = 0
	try {
		const res = await fetchFn(new URL('api/v2/push/actions', scope).href, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({token, action}),
			credentials: 'same-origin', // the Cloudflare Access cookie, when there is one
			// An expired Cloudflare login answers with a redirect to its login
			// page; don't follow it, count it as a failure.
			redirect: 'manual',
		})
		if (res.ok) {
			const result: {result?: string, snoozed_until?: string} = await res.json().catch(() => ({}))
			if (result.result === 'snoozed' && result.snoozed_until) {
				return {
					title: taskTitle,
					body: `Snoozed until ${snoozeLabel(new Date(result.snoozed_until), now)}`,
					quiet: true,
				}
			}
			return null
		}
		status = res.status
	} catch {
		// No signal, or the server is down.
	}
	const failed: Record<NotificationAction, string> = {
		done: `Couldn't mark “${taskTitle}” done`,
		snooze: `Couldn't snooze “${taskTitle}”`,
		skip: `Couldn't skip “${taskTitle}”`,
	}
	return {
		title: failed[action],
		// "it": the task, or the Habits page for a habit reminder.
		body: status === 401
			? 'This notification is too old for its buttons. Tap to open it.'
			: 'Tap to open it.',
		quiet: false,
	}
}
