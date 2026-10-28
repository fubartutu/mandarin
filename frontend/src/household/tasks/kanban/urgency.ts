// Mandarin: how urgent a Kanban card looks (a coloured stripe on its left,
// household/kanban.scss). Priority and due date both count; the more urgent
// one wins. Done tasks are never urgent.

import {PRIORITIES} from '@/constants/priorities'

export type Urgency = 'red' | 'orange' | 'yellow' | null

const RANK: Record<NonNullable<Urgency>, number> = {yellow: 1, orange: 2, red: 3}
const SOON_DAYS = 3

export function urgencyFromPriority(priority: number | null | undefined): Urgency {
	if ((priority ?? 0) >= PRIORITIES.URGENT) return 'red' // Urgent, Do now
	if (priority === PRIORITIES.HIGH) return 'orange'
	if (priority === PRIORITIES.MEDIUM) return 'yellow'
	return null
}

/** Overdue: red. Due today: orange. Due in the next 3 days: yellow. Days are local days. */
export function urgencyFromDue(due: string | Date | null | undefined, now: Date): Urgency {
	const date = due ? new Date(due) : null
	if (!date || isNaN(+date) || date.getFullYear() < 1900) return null // Vikunja's "no date" is year 1
	if (date < now) return 'red'
	const day = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
	const daysAway = Math.round((day(date) - day(now)) / 864e5)
	if (daysAway === 0) return 'orange'
	if (daysAway <= SOON_DAYS) return 'yellow'
	return null
}

export function urgencyOf(task: {done?: boolean, priority?: number, due_date?: string | null}, now: Date): Urgency {
	if (task.done) return null
	const fromPriority = urgencyFromPriority(task.priority)
	const fromDue = urgencyFromDue(task.due_date, now)
	const rank = (u: Urgency) => (u ? RANK[u] : 0)
	return rank(fromDue) > rank(fromPriority) ? fromDue : fromPriority
}
