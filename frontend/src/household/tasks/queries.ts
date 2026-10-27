// A repeating task's household settings (calendar rule and end), following
// the conventions in .agents/docs/api.md. Online-only, like the menu: the
// server applies them and counts the end down when the task is done.

import {queryOptions, useMutation} from '@tanstack/vue-query'

import {tasksRepeatRead, tasksRepeatUpdate} from '@/client/generated'
import type {TaskRepeat} from '@/client/generated'
import {contextMutationOptions} from '@/client/queries/contextMutation'
import {endOfDay} from './repeatEnd'

export interface RepeatSettings {
	/** An RRULE, or '' when Vikunja's interval decides (see repeatRule.ts). */
	rule: string
	/** The last day an occurrence may fall on. */
	until: Date | null
	/** Occurrences left, the current one included. 0 = no limit. */
	repeatsLeft: number
}

export const repeatKeys = {
	all: ['taskRepeat'] as const,
	task: (taskId: number) => [...repeatKeys.all, taskId] as const,
}

export function normalizeRepeat(e: TaskRepeat): RepeatSettings {
	return {
		rule: e.rule ?? '',
		until: e.until ? new Date(e.until) : null,
		repeatsLeft: e.repeats_left ?? 0,
	}
}

export function taskRepeatQuery(taskId: number) {
	return queryOptions({
		queryKey: repeatKeys.task(taskId),
		queryFn: async () => (await tasksRepeatRead({path: {task: taskId}})).data,
		select: normalizeRepeat,
	})
}

/** Replaces the task's settings; empty ones remove them. */
export async function saveRepeat(taskId: number, settings: RepeatSettings): Promise<RepeatSettings> {
	const body = {
		rule: settings.rule,
		until: settings.until ? endOfDay(settings.until).toISOString() : null,
		repeats_left: settings.repeatsLeft,
	}
	return normalizeRepeat((await tasksRepeatUpdate({path: {task: taskId}, body})).data)
}

export function useSaveRepeatMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: ({taskId, settings}: {taskId: number, settings: RepeatSettings}) => saveRepeat(taskId, settings),
		onSettled: (_input, client) => client.invalidateQueries({queryKey: repeatKeys.all}),
	}))
}
