// Task breakdown through the generated client. Online only: the AI is on the
// server, and saving creates tasks.

import {useMutation} from '@tanstack/vue-query'

import {tasksBreakdown, tasksBreakdownSuggest} from '@/client/generated'
import {contextMutationOptions} from '@/client/queries/contextMutation'
import {invalidateTaskMembership, replaceTaskEverywhere} from '@/client/queries/taskCache'
import {newItem, type BreakdownItem, type BreakdownMode} from './breakdown'

/** Asks the AI. Errors aren't toasted: the panel shows them next to the typing box. */
export function useSuggestBreakdownMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async ({taskId, mode}: {taskId: number, mode: BreakdownMode}): Promise<BreakdownItem[]> => {
			const {data} = await tasksBreakdownSuggest({path: {task: taskId}, body: {mode}})
			return (data.steps ?? []).map(s => newItem(s.title ?? '', s.minutes ?? 0)).filter(i => i.title !== '')
		},
		toastError: () => false,
	}))
}

/** Saves the kept items: subtasks or a checklist. Resolves to the updated task. */
export function useApplyBreakdownMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async ({taskId, mode, items}: {taskId: number, mode: BreakdownMode, items: string[]}) =>
			(await tasksBreakdown({path: {task: taskId}, body: {mode, items}})).data,
		onSuccess: (task, _input, client) => replaceTaskEverywhere(client, task),
		// New subtasks join the project's lists and boards.
		onSettled: ({taskId}, client) => invalidateTaskMembership(client, taskId),
	}))
}
