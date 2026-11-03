// The caller's snooze on a task (Snooze 1h on a reminder notification), for
// the task page. Online-only; the server keeps it (pkg/household/push).

import {queryOptions, useMutation} from '@tanstack/vue-query'

import {tasksSnoozeDelete, tasksSnoozeRead} from '@/client/generated'
import {contextMutationOptions} from '@/client/queries/contextMutation'

export const snoozeKeys = {
	all: ['taskSnooze'] as const,
	task: (taskId: number) => [...snoozeKeys.all, taskId] as const,
}

/** When the snoozed reminder comes back, or null. */
export function taskSnoozeQuery(taskId: number) {
	return queryOptions({
		queryKey: snoozeKeys.task(taskId),
		queryFn: async () => (await tasksSnoozeRead({path: {task: taskId}})).data,
		select: data => data.snoozed_until ? new Date(data.snoozed_until) : null,
	})
}

export function useCancelSnoozeMutation() {
	return useMutation(contextMutationOptions({
		mutationFn: async (taskId: number) => {
			await tasksSnoozeDelete({path: {task: taskId}})
		},
		onSettled: (taskId, client) => client.invalidateQueries({queryKey: snoozeKeys.task(taskId)}),
	}))
}
