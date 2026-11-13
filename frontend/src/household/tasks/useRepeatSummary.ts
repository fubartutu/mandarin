// The task page's collapsed Repeat row (FORK line in TaskDetailView.vue).
// Vikunja labels it from repeat_after, which for a task with a household
// calendar rule is only the stand-in (7/30/365 days × interval), so it would
// say "Every 7 days" for "Every Mon, Thu". This wraps Vikunja's label: the
// rule in plain words when there is one, plus the end ("until 1 Mar").

import {computed, watch, type Ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'

import type {Task} from '@/client/generated'
import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'
import {taskRepeatQuery} from './queries'
import {fromRRule} from './repeatRule'
import {repeatLabel} from './repeatLabel'

export function useRepeatSummary(task: Ref<Task>, vikunjaLabel: Ref<string>) {
	const {t, locale} = useI18n({useScope: 'global'})

	const taskId = computed(() => task.value.id ?? 0)
	// The same test Vikunja uses for "is set": only repeating tasks can have
	// household settings, so others make no request.
	const repeats = computed(() => (task.value.repeat_after ?? 0) > 0
		|| task.value.repeat_mode !== TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT)

	// Shares its cache entry with RepeatSettingsField, so a change made in
	// the editor shows here straight away (its save invalidates the key).
	const settings = useQuery(computed(() => ({
		...taskRepeatQuery(taskId.value),
		enabled: taskId.value > 0 && repeats.value,
	})))

	// Completing or skipping counts the end down on the server.
	watch(() => task.value.updated, () => {
		if (taskId.value > 0 && repeats.value) settings.refetch()
	})

	return computed(() => {
		const value = settings.data.value
		if (!repeats.value || !value) return vikunjaLabel.value
		return repeatLabel(vikunjaLabel.value, fromRRule(value.rule), value, t, locale.value)
	})
}
