<template>
	<!-- Mandarin: replaces Vikunja's DeferTask in task rows (FORK import in SingleTaskInProject.vue). -->
	<div
		:class="{'is-loading': busy}"
		class="reschedule loading-container"
		@click.stop
		@mousedown.stop
		@pointerdown.stop
		@touchstart.stop
	>
		<label class="label">{{ $t('household.tasks.reschedule.title') }}</label>

		<XButton
			v-if="repeats && next"
			class="skip"
			variant="secondary"
			icon="forward"
			:shadow="false"
			@click.prevent.stop="skip"
		>
			{{ $t('household.tasks.reschedule.skipTo', {date: dayLabel(next)}) }}
		</XButton>

		<div class="quick-days">
			<XButton
				v-for="{key, day} in days"
				:key="key"
				:shadow="false"
				variant="secondary"
				@click.prevent.stop="moveTo(day)"
			>
				<span class="quick-day">
					<span class="day-name">{{ $t(`household.tasks.reschedule.${key}`) }}</span>
					<span class="day-date">{{ weekdayLabel(day) }}</span>
				</span>
			</XButton>
		</div>

		<DatepickerInline
			v-model="dueDate"
			:show-shortcuts="false"
			@update:modelValue="onPickerUpdate"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed, onBeforeUnmount, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useDebounceFn} from '@vueuse/core'
import {useQuery} from '@tanstack/vue-query'

import DatepickerInline from '@/components/input/DatepickerInline.vue'
import {useUpdateTaskMutation} from '@/client/queries/taskMutations'
import {useDelayedLoading} from '@/composables/useDelayedLoading'
import {parseDateOrNull} from '@/helpers/parseDateOrNull'
import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'
import type {Task as ITask} from '@/client/generated'
import {atTaskTime, quickDays} from './reschedule'
import {taskRepeatQuery, useSkipOccurrenceMutation} from './queries'

const props = defineProps<{
	modelValue: ITask,
}>()

const emit = defineEmits<{
	'update:modelValue': [value: ITask]
}>()

const {locale} = useI18n({useScope: 'global'})

const update = useUpdateTaskMutation()
const skipMutation = useSkipOccurrenceMutation()
const busy = useDelayedLoading(() => update.isPending.value || skipMutation.isPending.value)

const task = ref<ITask>({}) // filled from modelValue by the watch below
// Kept apart from the task so the calendar never sees a null mid-update.
const dueDate = ref<Date | null>(null)
const lastValue = ref<Date | null>(null)
watch(() => props.modelValue, value => {
	task.value = {...value}
	dueDate.value = parseDateOrNull(value.due_date)
	lastValue.value = dueDate.value
}, {immediate: true})

const days = computed(() => quickDays(new Date()))

const repeats = computed(() => (task.value.repeat_after ?? 0) > 0
	|| task.value.repeat_mode === TASK_REPEAT_MODES.REPEAT_MODE_MONTH)
// The server says where a skip lands (it knows the rule and the end).
const settings = useQuery(computed(() => ({
	...taskRepeatQuery(task.value.id ?? 0),
	enabled: repeats.value && (task.value.id ?? 0) > 0,
})))
const next = computed(() => settings.data.value?.next ?? null)

function weekdayLabel(day: Date) {
	return day.toLocaleDateString(locale.value, {weekday: 'short'})
}

function dayLabel(day: Date) {
	return day.toLocaleDateString(locale.value, {weekday: 'short', day: 'numeric', month: 'short'})
}

function moveTo(day: Date) {
	debouncedSave.cancel()
	dueDate.value = atTaskTime(day, parseDateOrNull(task.value.due_date))
	save()
}

async function skip() {
	debouncedSave.cancel()
	const skipped = await skipMutation.mutateAsync(task.value.id!)
	task.value = skipped
	lastValue.value = parseDateOrNull(skipped.due_date)
	emit('update:modelValue', skipped)
}

const debouncedSave = useDebounceFn(save, 500)

function onPickerUpdate() {
	debouncedSave()
}

onBeforeUnmount(() => debouncedSave.flush())

async function save() {
	if (!dueDate.value) return
	if (lastValue.value && +dueDate.value === +lastValue.value) return
	const saved = await update.mutateAsync({
		...task.value,
		id: task.value.id!,
		due_date: dueDate.value.toISOString(),
	})
	lastValue.value = parseDateOrNull(saved.due_date)
	task.value = saved
	emit('update:modelValue', saved)
}
</script>

<style lang="scss" scoped>
// Same width as Vikunja's DeferTask: 350px plus the room the pane is pulled right.
$reschedule-max-width: 350px + 100px;

.reschedule {
	inline-size: 100%;
	max-inline-size: $reschedule-max-width;

	.bottom-sheet & {
		max-inline-size: none;
		padding: 0 1rem 1rem;

		> .label {
			display: none;
		}
	}
}

.skip {
	inline-size: 100%;
	margin-block: .5rem;
}

.quick-days {
	display: flex;
	gap: .25rem;
	margin: .5rem 0;

	> * {
		flex: 1;
		min-inline-size: 0;
		padding-inline: .25rem;
	}
}

// Name over weekday, in one compact button each.
.quick-day {
	display: flex;
	flex-direction: column;
	align-items: center;
	line-height: 1.15;
	white-space: nowrap;
}

.day-name {
	font-size: .75rem;
}

.day-date {
	font-size: .7rem;
	font-weight: normal;
	color: var(--grey-500);
	text-transform: none;
}
</style>
