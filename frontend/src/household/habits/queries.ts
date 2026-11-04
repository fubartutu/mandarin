// Habit API calls, following the conventions in .agents/docs/api.md.
// Today's habits and ticks work offline through the shared sync engine
// (sync/); this file has the shapes, the history grid and archived habits
// (fetched with TanStack Query), and the online-only edits.

import {queryOptions, useMutation} from '@tanstack/vue-query'
import type {QueryClient} from '@tanstack/vue-query'

import {
	habitGroupsCreate,
	habitGroupsDelete,
	habitGroupsList,
	habitGroupsUpdate,
	habitsCreate,
	habitsDelete,
	habitsHistory,
	habitsList,
	habitsUpdate,
} from '@/client/generated'
import type {Habit, HabitGroup, HabitWritable, History, Period, Progress} from '@/client/generated'
import {contextMutationOptions} from '@/client/queries/contextMutation'
import {fetchAllPages} from '@/client/queries/fetchAllPages'

// ---- Normalised shapes ----

export type HabitKind = NonNullable<Habit['kind']>
export type PeriodStatus = NonNullable<Period['status']>
export type HabitProgress = Required<Omit<Progress, 'today_entries'>> & {today_entries: string[]}

export interface HabitResponse {
	id: number
	title: string
	kind: HabitKind
	weekdays: string // seven 0/1 characters, Monday first
	goal: number
	color: string // hex without #, '' for none
	groupId: number
	position: number
	shared: boolean
	reminderTimes: string[]
	archived: boolean
	paused: boolean
	ownerId: number
	progress: HabitProgress
}

export interface HabitGroupResponse {
	id: number
	title: string
	position: number
	shared: boolean
	ownerId: number
}

export interface PeriodResponse {
	start: string // YYYY-MM-DD; a week's Monday for weekly habits
	status: PeriodStatus
	count: number
}

export interface HistoryResponse {
	from: string
	to: string
	periods: PeriodResponse[]
	/** Tick uids by day, oldest first, for taking a tick off a past day. */
	entries: Map<string, string[]>
	skips: Set<string>
}

export function normalizeProgress(p: Progress | undefined): HabitProgress {
	return {
		today: p?.today ?? '',
		today_count: p?.today_count ?? 0,
		goal: p?.goal ?? 1,
		due_today: p?.due_today ?? false,
		done_today: p?.done_today ?? false,
		skipped: p?.skipped ?? false,
		today_entries: p?.today_entries ?? [],
		done: p?.done ?? 0,
		due: p?.due ?? 0,
		streak: p?.streak ?? 0,
		best_streak: p?.best_streak ?? 0,
		grace_used: p?.grace_used ?? false,
		this_week: p?.this_week ?? 0,
		last_week: p?.last_week ?? 0,
	}
}

export function normalizeHabit(h: Habit): HabitResponse {
	return {
		id: h.id ?? 0,
		title: h.title ?? '',
		kind: h.kind ?? 'daily',
		weekdays: h.weekdays ?? '1111111',
		goal: h.goal ?? 1,
		color: h.color ?? '',
		groupId: h.group_id ?? 0,
		position: h.position ?? 0,
		shared: h.shared ?? false,
		reminderTimes: h.reminder_times ?? [],
		archived: h.archived ?? false,
		paused: h.paused ?? false,
		ownerId: h.owner_id ?? 0,
		progress: normalizeProgress(h.progress),
	}
}

export function normalizeGroup(g: HabitGroup): HabitGroupResponse {
	return {
		id: g.id ?? 0,
		title: g.title ?? '',
		position: g.position ?? 0,
		shared: g.shared ?? false,
		ownerId: g.owner_id ?? 0,
	}
}

export function normalizeHistory(h: History): HistoryResponse {
	const entries = new Map<string, string[]>()
	for (const e of h.entries ?? []) {
		const day = e.day ?? ''
		entries.set(day, [...(entries.get(day) ?? []), e.uid ?? ''])
	}
	return {
		from: h.from ?? '',
		to: h.to ?? '',
		periods: (h.periods ?? []).map(p => ({start: p.start ?? '', status: p.status ?? 'missed', count: p.count ?? 0})),
		entries,
		skips: new Set((h.skips ?? []).map(s => s.day ?? '')),
	}
}

/** What the editor sends. The kind can't change once a habit exists. */
export interface HabitDraft {
	title: string
	kind: HabitKind
	weekdays: string
	goal: number
	color: string
	groupId: number
	shared: boolean
	reminderTimes: string[] // HH:MM, sorted
}

// ---- Keys and queries ----

export const habitKeys = {
	all: ['habits'] as const,
	list: (archived: boolean) => [...habitKeys.all, 'list', archived] as const,
	groups: () => [...habitKeys.all, 'groups'] as const,
	history: (id: number, from: string, to: string) => [...habitKeys.all, 'history', id, from, to] as const,
}

/** The caller's habits and the household routines, each with today's progress. */
export function habitsQuery(archived = false) {
	return queryOptions({
		queryKey: habitKeys.list(archived),
		queryFn: () => fetchAllPages(async page => (await habitsList({query: {page, archived}})).data),
		select: habits => habits.map(normalizeHabit),
		staleTime: 30 * 1000,
	})
}

export function habitGroupsQuery() {
	return queryOptions({
		queryKey: habitKeys.groups(),
		queryFn: () => fetchAllPages(async page => (await habitGroupsList({query: {page}})).data),
		select: groups => groups.map(normalizeGroup),
		staleTime: 60 * 1000,
	})
}

/** The habit's grid from one day to another, both included. */
export function habitHistoryQuery(id: number, from: string, to: string) {
	return queryOptions({
		queryKey: habitKeys.history(id, from, to),
		queryFn: async ({signal}) => normalizeHistory((await habitsHistory({path: {habit: id}, query: {from, to}, signal})).data),
		staleTime: 30 * 1000,
	})
}

// ---- Mutations ----
// Editing habits and routines is online-only; ticks and skips go through the
// offline queue instead (sync/useHabitsSync.ts). Each edit refreshes every
// habit query afterwards (onSettled), and tells the queue to fetch again.

// Edits go one at a time: SQLite takes one writer at once, so two quick
// saves mustn't race each other. Mutations sharing a scope queue.
const HABIT_WRITES = {id: 'habit-writes'}

/** Fired after an edit; the offline queue (sync/useHabitsSync.ts) fetches again. */
export const HABITS_CHANGED_EVENT = 'household.habits.changed'

const refresh = (client: QueryClient) => {
	window.dispatchEvent(new Event(HABITS_CHANGED_EVENT))
	return client.invalidateQueries({queryKey: habitKeys.all})
}

/**
 * The whole habit as PUT wants it (PUT replaces every field, so fields the
 * caller doesn't change are sent as they are), with the draft's fields on top.
 */
function habitBody(habit: HabitResponse | undefined, draft: Partial<HabitDraft>, extra: Partial<HabitWritable> = {}): HabitWritable {
	const d = {...(habit ?? {}), ...draft}
	return {
		title: d.title,
		kind: d.kind,
		weekdays: d.weekdays,
		goal: d.goal,
		color: d.color,
		group_id: d.groupId,
		shared: d.shared,
		position: habit?.position,
		archived: habit?.archived,
		paused: habit?.paused,
		reminder_times: d.reminderTimes,
		...extra,
	}
}

/** Creates a habit (no `habit`) or saves the editor's changes to one. */
export function useSaveHabitMutation() {
	return useMutation({...contextMutationOptions({
		mutationFn: async ({habit, draft}: {habit?: HabitResponse, draft: HabitDraft}) => normalizeHabit(habit
			? (await habitsUpdate({path: {habit: habit.id}, body: habitBody(habit, draft)})).data
			: (await habitsCreate({body: habitBody(undefined, draft)})).data),
		onSettled: (_input, client) => refresh(client),
	}), scope: HABIT_WRITES})
}

/** Pauses, archives or moves a habit, leaving the rest as it is. */
export function useChangeHabitMutation() {
	return useMutation({...contextMutationOptions({
		mutationFn: async ({habit, changes}: {habit: HabitResponse, changes: Partial<Pick<HabitWritable, 'paused' | 'archived' | 'position' | 'group_id'>>}) =>
			normalizeHabit((await habitsUpdate({path: {habit: habit.id}, body: habitBody(habit, {}, changes)})).data),
		onSettled: (_input, client) => refresh(client),
	}), scope: HABIT_WRITES})
}

export function useDeleteHabitMutation() {
	return useMutation({...contextMutationOptions({
		mutationFn: async (id: number) => {
			await habitsDelete({path: {habit: id}})
		},
		onSettled: (_input, client) => refresh(client),
	}), scope: HABIT_WRITES})
}

export function useSaveGroupMutation() {
	return useMutation({...contextMutationOptions({
		mutationFn: async ({id, title, shared, position}: {id: number, title: string, shared: boolean, position: number}) => {
			const body = {title, shared, position}
			return normalizeGroup(id
				? (await habitGroupsUpdate({path: {group: id}, body})).data
				: (await habitGroupsCreate({body})).data)
		},
		onSettled: (_input, client) => refresh(client),
	}), scope: HABIT_WRITES})
}

export function useDeleteGroupMutation() {
	return useMutation({...contextMutationOptions({
		mutationFn: async (id: number) => {
			await habitGroupsDelete({path: {group: id}})
		},
		onSettled: (_input, client) => refresh(client),
	}), scope: HABIT_WRITES})
}
