// "10:40", or "Wed 10:40" when it isn't today: when a snoozed reminder comes
// back. Used by the notification that confirms a snooze (sw.ts) and the task
// page (SnoozeNotice.vue). No Vue in here, so the service worker can import it.

export function snoozeLabel(until: Date, now: Date, locale?: string): string {
	const time = until.toLocaleTimeString(locale, {hour: 'numeric', minute: '2-digit'})
	if (until.toDateString() === now.toDateString()) {
		return time
	}
	return `${until.toLocaleDateString(locale, {weekday: 'short'})} ${time}`
}
