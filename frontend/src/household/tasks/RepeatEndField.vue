<template>
	<!-- Mandarin: when a repeating task stops. Sits under Vikunja's RepeatAfter (FORK in TaskDetailView.vue). -->
	<div class="repeat-end">
		<label class="repeat-end-row">
			<span>{{ $t('household.tasks.repeatEnd.until') }}</span>
			<input
				v-model="untilInput"
				class="input is-small"
				type="date"
				:disabled="disabled || undefined"
				@change="save"
			>
		</label>
		<label class="repeat-end-row">
			<span>{{ $t('household.tasks.repeatEnd.timesLeft') }}</span>
			<input
				v-model.number="countInput"
				class="input is-small"
				type="number"
				min="0"
				:max="MAX_REPEATS_LEFT"
				:placeholder="$t('household.tasks.repeatEnd.noLimit')"
				:disabled="disabled || undefined"
				@change="save"
			>
		</label>
		<p class="help">
			{{ summary }}
		</p>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'

import type {Task} from '@/client/generated'
import {MAX_REPEATS_LEFT, type RepeatEnd} from './repeatEnd'
import {taskRepeatEndQuery, useSaveRepeatEndMutation} from './queries'

const props = withDefaults(defineProps<{
	task: Task
	disabled?: boolean
}>(), {
	disabled: false,
})

const {t, locale} = useI18n({useScope: 'global'})

const taskId = computed(() => props.task.id ?? 0)
const end = useQuery(computed(() => ({
	...taskRepeatEndQuery(taskId.value),
	enabled: taskId.value > 0,
})))
const saveEnd = useSaveRepeatEndMutation()

// Completing the task counts it down (or ends it) on the server, so look
// again whenever the task itself was saved.
watch(() => props.task.updated, () => end.refetch())

// <input type="date"> speaks YYYY-MM-DD in local time.
const untilInput = ref('')
const countInput = ref<number | ''>('')
watch(() => end.data.value, (value?: RepeatEnd) => {
	untilInput.value = value?.until ? toDateInput(value.until) : ''
	countInput.value = value?.repeatsLeft ? value.repeatsLeft : ''
}, {immediate: true})

function toDateInput(d: Date): string {
	const pad = (n: number) => String(n).padStart(2, '0')
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function fromDateInput(value: string): Date | null {
	const [y, m, d] = value.split('-').map(Number)
	return y && m && d ? new Date(y, m - 1, d) : null
}

function save() {
	const count = Number(countInput.value) || 0
	if (count < 0 || count > MAX_REPEATS_LEFT) return
	saveEnd.mutate({
		taskId: taskId.value,
		end: {until: fromDateInput(untilInput.value), repeatsLeft: count},
	})
}

const summary = computed(() => {
	const value = end.data.value
	if (!value || (!value.until && !value.repeatsLeft)) return t('household.tasks.repeatEnd.forever')
	const parts: string[] = []
	if (value.until) parts.push(t('household.tasks.repeatEnd.summaryUntil', {date: value.until.toLocaleDateString(locale.value, {weekday: 'short', day: 'numeric', month: 'short', year: 'numeric'})}))
	if (value.repeatsLeft) parts.push(t('household.tasks.repeatEnd.summaryCount', {count: value.repeatsLeft}, value.repeatsLeft))
	return parts.join(t('household.tasks.repeatEnd.or'))
})
</script>

<style lang="scss" scoped>
.repeat-end {
	margin-block-start: .5rem;
}

.repeat-end-row {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: .5rem;
	margin-block-end: .25rem;

	.input {
		max-inline-size: 10rem;
	}
}
</style>
