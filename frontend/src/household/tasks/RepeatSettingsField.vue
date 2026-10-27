<template>
	<!-- Mandarin: a calendar rule and an end for a repeating task. Sits under Vikunja's RepeatAfter (FORK in TaskDetailView.vue). -->
	<div class="repeat-settings">
		<label class="repeat-row">
			<span>{{ $t('household.tasks.repeat.on') }}</span>
			<div class="select is-small">
				<select
					v-model="kind"
					:disabled="disabled || undefined"
					@change="changeKind"
				>
					<option value="">{{ $t('household.tasks.repeat.kinds.interval') }}</option>
					<option value="weekly">{{ $t('household.tasks.repeat.kinds.weekly') }}</option>
					<option value="monthDay">{{ $t('household.tasks.repeat.kinds.monthDay') }}</option>
					<option value="monthWeekday">{{ $t('household.tasks.repeat.kinds.monthWeekday') }}</option>
					<option value="months">{{ $t('household.tasks.repeat.kinds.months') }}</option>
					<option value="years">{{ $t('household.tasks.repeat.kinds.years') }}</option>
				</select>
			</div>
		</label>

		<div
			v-if="kind === 'weekly'"
			class="weekdays"
			role="group"
			:aria-label="$t('household.tasks.repeat.kinds.weekly')"
		>
			<button
				v-for="day in WEEKDAYS"
				:key="day"
				type="button"
				class="weekday"
				:class="{'is-selected': days.includes(day)}"
				:aria-pressed="days.includes(day)"
				:aria-label="weekdayName(day, 'long')"
				:disabled="disabled || undefined"
				@click="toggleDay(day)"
			>
				{{ weekdayName(day, 'narrow') }}
			</button>
		</div>

		<label
			v-if="kind === 'monthDay'"
			class="repeat-row"
		>
			<span>{{ $t('household.tasks.repeat.day') }}</span>
			<div class="select is-small">
				<select
					v-model.number="monthDay"
					:disabled="disabled || undefined"
					@change="save"
				>
					<option
						v-for="n in 31"
						:key="n"
						:value="n"
					>{{ n }}</option>
					<option :value="-1">{{ $t('household.tasks.repeat.lastDay') }}</option>
				</select>
			</div>
		</label>

		<div
			v-if="kind === 'monthWeekday'"
			class="repeat-row"
		>
			<span>{{ $t('household.tasks.repeat.the') }}</span>
			<div class="selects">
				<div class="select is-small">
					<select
						v-model.number="nth"
						:aria-label="$t('household.tasks.repeat.which')"
						:disabled="disabled || undefined"
						@change="save"
					>
						<option
							v-for="n in NTHS"
							:key="n"
							:value="n"
						>
							{{ $t(`household.tasks.repeat.nth.${n}`) }}
						</option>
					</select>
				</div>
				<div class="select is-small">
					<select
						v-model="weekday"
						:aria-label="$t('household.tasks.repeat.weekday')"
						:disabled="disabled || undefined"
						@change="save"
					>
						<option
							v-for="day in WEEKDAYS"
							:key="day"
							:value="day"
						>
							{{ weekdayName(day, 'long') }}
						</option>
					</select>
				</div>
			</div>
		</div>

		<label
			v-if="kind !== ''"
			class="repeat-row"
		>
			<span>{{ $t('household.tasks.repeat.every') }}</span>
			<span class="interval">
				<input
					v-model.number="interval"
					class="input is-small"
					type="number"
					min="1"
					max="100"
					:disabled="disabled || undefined"
					@change="save"
				>
				{{ $t(`household.tasks.repeat.units.${unit}`, interval) }}
			</span>
		</label>

		<p
			v-if="kind !== ''"
			class="help"
		>
			{{ ruleSummary }}. {{ $t('household.tasks.repeat.overrides') }}
		</p>

		<label class="repeat-row">
			<span>{{ $t('household.tasks.repeatEnd.until') }}</span>
			<input
				v-model="untilInput"
				class="input is-small"
				type="date"
				:disabled="disabled || undefined"
				@change="save"
			>
		</label>
		<label class="repeat-row">
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
			{{ endSummary }}
		</p>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'

import type {Task} from '@/client/generated'
import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'
import {MAX_REPEATS_LEFT} from './repeatEnd'
import {
	WEEKDAYS,
	firstOccurrence,
	fromRRule,
	isWorkdays,
	standInSeconds,
	toRRule,
	type Nth,
	type RepeatRule,
	type Weekday,
} from './repeatRule'
import {taskRepeatQuery, useSaveRepeatMutation, type RepeatSettings} from './queries'

const props = withDefaults(defineProps<{
	task: Task
	/** TaskDetailView's own save, so the task's stand-in interval and due date go through it first. */
	saveTask: (task: Task) => Promise<void>
	disabled?: boolean
}>(), {
	disabled: false,
})

const {t, locale} = useI18n({useScope: 'global'})

const NTHS: Nth[] = [1, 2, 3, 4, -1]

const taskId = computed(() => props.task.id ?? 0)
const settings = useQuery(computed(() => ({
	...taskRepeatQuery(taskId.value),
	enabled: taskId.value > 0,
})))
const saveSettings = useSaveRepeatMutation()

// Completing the task counts it down (or ends it) on the server, so look
// again whenever the task itself was saved.
watch(() => props.task.updated, () => settings.refetch())

// The form's state; every change saves.
const kind = ref<RepeatRule['kind'] | ''>('')
const days = ref<Weekday[]>([])
const monthDay = ref(1)
const nth = ref<Nth>(1)
const weekday = ref<Weekday>('MO')
const interval = ref(1)
const untilInput = ref('') // <input type="date"> speaks YYYY-MM-DD in local time
const countInput = ref<number | ''>('')

watch(() => settings.data.value, (value?: RepeatSettings) => {
	const rule = value ? fromRRule(value.rule) : null
	kind.value = rule?.kind ?? ''
	if (rule) fillFrom(rule)
	untilInput.value = value?.until ? toDateInput(value.until) : ''
	countInput.value = value?.repeatsLeft ? value.repeatsLeft : ''
}, {immediate: true})

function fillFrom(rule: RepeatRule) {
	interval.value = rule.interval
	if (rule.kind === 'weekly') days.value = [...rule.days]
	if (rule.kind === 'monthDay') monthDay.value = rule.day
	if (rule.kind === 'monthWeekday') {
		nth.value = rule.nth
		weekday.value = rule.weekday
	}
}

const parseDue = (value?: string | null) => value && +new Date(value) > 0 ? new Date(value) : null

// The due date rule changes count from: the one the task had before this
// page moved it to fit a rule. Counting from the moved date instead, each
// tweak (2nd, then Tuesday) would start from the last one's pick and the
// date would creep forward. A due date set by hand becomes the new base.
const baseDue = ref<Date | null>(null)
let movedTo: Date | null = null
watch(taskId, () => {
	movedTo = null
})
watch(() => props.task.due_date, value => {
	const due = parseDue(value)
	if (!movedTo || !due || +due !== +movedTo) baseDue.value = due
}, {immediate: true})

/** The due date, else today: what a new rule's defaults come from. */
const anchor = computed(() => baseDue.value ?? new Date())

function changeKind() {
	const d = anchor.value
	const day = WEEKDAYS[(d.getDay() + 6) % 7]
	interval.value = kind.value === 'months' ? 2 : 1
	days.value = [day]
	monthDay.value = d.getDate()
	weekday.value = day
	const lastWeek = d.getDate() + 7 > new Date(d.getFullYear(), d.getMonth() + 1, 0).getDate()
	nth.value = lastWeek ? -1 : Math.min(4, Math.ceil(d.getDate() / 7)) as Nth
	save()
}

function toggleDay(day: Weekday) {
	// A weekly rule needs a day: the last one stays.
	if (days.value.length === 1 && days.value[0] === day) return
	days.value = days.value.includes(day) ? days.value.filter(d => d !== day) : [...days.value, day]
	save()
}

const draft = computed<RepeatRule | null>(() => {
	const n = Math.min(100, Math.max(1, Math.round(Number(interval.value) || 1)))
	switch (kind.value) {
		case 'weekly':
			return days.value.length ? {kind: 'weekly', interval: n, days: days.value} : null
		case 'monthDay':
			return {kind: 'monthDay', interval: n, day: monthDay.value}
		case 'monthWeekday':
			return {kind: 'monthWeekday', interval: n, nth: nth.value, weekday: weekday.value}
		case 'months':
			return {kind: 'months', interval: n}
		case 'years':
			return {kind: 'years', interval: n}
	}
	return null
})

const unit = computed(() => kind.value === 'weekly' ? 'weeks' : kind.value === 'years' ? 'years' : 'months')

function toDateInput(d: Date): string {
	const pad = (n: number) => String(n).padStart(2, '0')
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function fromDateInput(value: string): Date | null {
	const [y, m, d] = value.split('-').map(Number)
	return y && m && d ? new Date(y, m - 1, d) : null
}

async function save() {
	const count = Number(countInput.value) || 0
	if (count < 0 || count > MAX_REPEATS_LEFT) return
	const rule = draft.value
	if (kind.value !== '' && !rule) return
	if (rule) {
		// The task itself first (chained, not at once: SQLite takes one writer):
		// the stand-in interval other apps see, and the due date on a day the rule falls on.
		const due = parseDue(props.task.due_date)
		const firstDue = baseDue.value ? firstOccurrence(rule, baseDue.value) : null
		const standIn = standInSeconds(rule)
		if (props.task.repeat_after !== standIn
			|| props.task.repeat_mode !== TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT
			|| (firstDue && (!due || +firstDue !== +due))) {
			movedTo = firstDue
			await props.saveTask({
				...props.task,
				repeat_after: standIn,
				repeat_mode: TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT,
				due_date: firstDue ? firstDue.toISOString() : props.task.due_date,
			})
		}
	}
	saveSettings.mutate({
		taskId: taskId.value,
		settings: {rule: rule ? toRRule(rule) : '', until: fromDateInput(untilInput.value), repeatsLeft: count},
	})
}

function weekdayName(day: Weekday, style: 'long' | 'short' | 'narrow') {
	// 5 January 2026 is a Monday.
	return new Date(2026, 0, 5 + WEEKDAYS.indexOf(day)).toLocaleDateString(locale.value, {weekday: style})
}

function listOf(names: string[]) {
	return new Intl.ListFormat(locale.value, {type: 'conjunction'}).format(names)
}

const ruleSummary = computed(() => {
	const rule = draft.value
	if (!rule) return ''
	const n = rule.interval
	switch (rule.kind) {
		case 'weekly': {
			const names = isWorkdays(rule.days)
				? t('household.tasks.repeat.weekdays')
				: listOf(rule.days.map(day => weekdayName(day, 'short')))
			return t('household.tasks.repeat.summary.weekly', {days: names, n}, n)
		}
		case 'monthDay':
			return rule.day === -1
				? t('household.tasks.repeat.summary.lastDay', {n}, n)
				: t('household.tasks.repeat.summary.monthDay', {day: rule.day, n}, n)
		case 'monthWeekday':
			return t('household.tasks.repeat.summary.monthWeekday', {
				nth: t(`household.tasks.repeat.nth.${rule.nth}`).toLowerCase(),
				weekday: weekdayName(rule.weekday, 'long'),
				n,
			}, n)
		case 'months':
			return t('household.tasks.repeat.summary.months', {n}, n)
		case 'years':
			return t('household.tasks.repeat.summary.years', {n}, n)
	}
	return ''
})

const endSummary = computed(() => {
	const value = settings.data.value
	if (!value || (!value.until && !value.repeatsLeft)) return t('household.tasks.repeatEnd.forever')
	const parts: string[] = []
	if (value.until) parts.push(t('household.tasks.repeatEnd.summaryUntil', {date: value.until.toLocaleDateString(locale.value, {weekday: 'short', day: 'numeric', month: 'short', year: 'numeric'})}))
	if (value.repeatsLeft) parts.push(t('household.tasks.repeatEnd.summaryCount', {count: value.repeatsLeft}, value.repeatsLeft))
	return parts.join(t('household.tasks.repeatEnd.or'))
})
</script>

<style lang="scss" scoped>
.repeat-settings {
	margin-block-start: .5rem;
}

.repeat-row {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: .5rem;
	margin-block-end: .25rem;

	.input {
		max-inline-size: 10rem;
	}
}

.selects {
	display: flex;
	gap: .25rem;
}

.interval {
	display: flex;
	align-items: center;
	gap: .5rem;

	.input {
		inline-size: 4.5rem;
	}
}

.weekdays {
	display: flex;
	justify-content: space-between;
	gap: .25rem;
	margin-block-end: .5rem;
}

.weekday {
	flex: 1;
	min-block-size: 2.25rem; // a comfortable tap target on phones
	border: 1px solid var(--grey-300);
	border-radius: 999px;
	background: transparent;
	color: var(--text);
	font: inherit;
	cursor: pointer;

	&.is-selected {
		border-color: var(--primary);
		background: var(--primary);
		color: var(--white);
	}

	&:disabled {
		cursor: not-allowed;
		opacity: .6;
	}
}
</style>
