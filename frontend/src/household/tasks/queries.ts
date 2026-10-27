// A repeating task's end, following the conventions in .agents/docs/api.md.
// Online-only, like the menu: the server counts it down when the task is done.

import {queryOptions, useMutation} from '@tanstack/vue-query'

import {tasksRepeatRead, tasksRepeatUpdate} from '@/client/generated'
import type {TaskRepeat} from '@/client/generated'
import {contextMutationOptions} from '@/client/queries/contextMutation'
import {endOfDay, type RepeatEnd} from './repeatEnd'

export const repeatEndKeys = {
	all: ['taskRepeatEnd'] as const,
	task: (taskId: number) => [...repeatEndKeys.all, taskId] as const,
}

export function normalizeRepeatEnd(e: TaskRepeat): RepeatEnd {
	return {
		until: e.until ? new Date(e.until) : null,
		repeatsLeft: e.repeats_left ?? 0,
	}
}

export function taskRepeatEndQuery(taskId: number) {
	return queryOptions({
		queryKey: repeatEndKeys.task(taskId),
		queryFn: async () => (await tasksRepeatRead({path: {task: taskId}})).data,
		select: normalizeRepeatEnd,
	})
}

/** Replaces the task's end; an empty one ({until: null, repeatsLeft: 0}) removes it. */
export async function saveRepeatEnd(taskId: number, end: RepeatEnd): Promise<RepeatEnd> {
	const body = {
		until: end.until ? endOfDay(end.until).toISOString() : null,
		repeats_left: end.repeatsLeft,
	}
	return normalizeRepeatEnd((await tasksRepeatUpdate({path: {task: taskId}, body})).data)
}

export function useSaveRepeatEndMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: ({taskId, end}: {taskId: number, end: RepeatEnd}) => saveRepeatEnd(taskId, end),
		onSettled: (_input, client) => client.invalidateQueries({queryKey: repeatEndKeys.all}),
	}))
}
