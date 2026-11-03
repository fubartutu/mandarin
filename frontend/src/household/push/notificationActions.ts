// The Done and Snooze buttons on task notifications (pkg/household/push,
// actions.go). Runs in the service worker (household/sw.ts), which can't read
// the app's login token: the notification carries a signed action token, and
// that's all POST /api/v2/push/actions needs.
//
// A plain fetch, not the generated client: the client reads the login token
// and API address from the page, and a service worker has no page.

/** The buttons the server sends, as the Web Notification API takes them. */
export interface NotificationButton {
	action: string
	title: string
}

export const NOTIFICATION_ACTIONS = ['done', 'snooze'] as const
export type NotificationAction = typeof NOTIFICATION_ACTIONS[number]

export function isNotificationAction(action: string): action is NotificationAction {
	return (NOTIFICATION_ACTIONS as readonly string[]).includes(action)
}

/** A notification to show instead, when the tap didn't work. */
export interface FallbackNotification {
	title: string
	body: string
}

/**
 * Sends the tapped button to the server. Returns null when it worked (or the
 * task had already moved on), else what to tell the person.
 */
export async function runNotificationAction(
	scope: string,
	action: NotificationAction,
	token: string,
	taskTitle: string,
	fetchFn: typeof fetch = fetch,
): Promise<FallbackNotification | null> {
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
			return null
		}
		status = res.status
	} catch {
		// No signal, or the server is down.
	}
	return {
		title: action === 'done' ? `Couldn't mark “${taskTitle}” done` : `Couldn't snooze “${taskTitle}”`,
		body: status === 401
			? 'This notification is too old for its buttons. Tap to open the task.'
			: 'Tap to open the task.',
	}
}
