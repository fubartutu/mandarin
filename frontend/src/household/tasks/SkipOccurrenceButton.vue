<template>
	<!-- Mandarin: "Skip to Thu 30 Oct" next to Mark done on the task page (FORK in TaskDetailView.vue). -->
	<XButton
		v-if="repeats && next"
		variant="secondary"
		icon="forward"
		:loading="skipMutation.isPending.value"
		@click="skip"
	>
		{{ $t('household.tasks.reschedule.skipTo', {date: label}) }}
	</XButton>
</template>

<script setup lang="ts">
import {computed, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'

import type {Task} from '@/client/generated'
import {success} from '@/message'
import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'
import {taskRepeatQuery, useSkipOccurrenceMutation} from './queries'

const props = defineProps<{
	task: Task
}>()

const emit = defineEmits<{
	/** The task as the server left it: TaskDetailView takes its new dates (its own copy keeps edited fields). */
	skipped: [task: Task]
}>()

const {t, locale} = useI18n({useScope: 'global'})

const repeats = computed(() => (props.task.repeat_after ?? 0) > 0
	|| props.task.repeat_mode === TASK_REPEAT_MODES.REPEAT_MODE_MONTH)
const settings = useQuery(computed(() => ({
	...taskRepeatQuery(props.task.id ?? 0),
	enabled: repeats.value && (props.task.id ?? 0) > 0,
})))
// Ask again when the task's date changes (done, moved): the next date moves with it.
watch(() => props.task.due_date, () => {
	if (repeats.value) settings.refetch()
})
const next = computed(() => settings.data.value?.next ?? null)
const label = computed(() => next.value?.toLocaleDateString(locale.value, {weekday: 'short', day: 'numeric', month: 'short'}) ?? '')

const skipMutation = useSkipOccurrenceMutation()

async function skip() {
	const skipped = await skipMutation.mutateAsync(props.task.id!)
	emit('skipped', skipped)
	success({message: t('household.tasks.reschedule.skipped')})
}
</script>
