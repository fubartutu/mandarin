<template>
	<div
		class="habit-grid"
		:style="colorStyle(habit.color)"
	>
		<p
			v-if="history.isPending.value"
			class="has-text-grey is-italic"
		>
			{{ $t('household.home.loading') }}
		</p>
		<p
			v-else-if="!history.data.value"
			class="has-text-grey is-italic"
		>
			{{ $t('household.habits.notLoaded') }}
		</p>

		<!-- Weekly habits: one square per week, oldest first. -->
		<ol
			v-else-if="habit.kind === 'weekly'"
			class="weeks"
		>
			<li
				v-for="period in history.data.value.periods"
				:key="period.start"
				class="cell"
				:class="`is-${period.status}`"
				:title="cellTitle(period)"
			>
				<span class="is-sr-only">{{ cellTitle(period) }}</span>
			</li>
		</ol>

		<!-- Day habits: a column per week, Monday at the top. -->
		<div
			v-else
			class="days"
			role="grid"
			:aria-label="$t('household.habits.grid.label', {title: habit.title})"
		>
			<div
				v-for="(column, w) in columns"
				:key="w"
				class="column-week"
				role="row"
			>
				<template
					v-for="(period, d) in column"
					:key="d"
				>
					<span
						v-if="!period"
						class="cell is-empty"
						role="gridcell"
					/>
					<!-- One-a-day habits: a past day can be ticked or unticked here. -->
					<button
						v-else-if="editable"
						type="button"
						class="cell"
						:class="[`is-${period.status}`, countClass(period)]"
						:title="cellTitle(period)"
						:aria-label="cellTitle(period)"
						:aria-pressed="period.count > 0"
						role="gridcell"
						@click="toggle(period)"
					/>
					<span
						v-else
						class="cell"
						:class="[`is-${period.status}`, countClass(period)]"
						:title="cellTitle(period)"
						role="gridcell"
					>
						<span class="is-sr-only">{{ cellTitle(period) }}</span>
					</span>
				</template>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'

import {fromKey} from '../../menu/week'
import {colorStyle} from '../colors'
import {gridColumns, gridRange} from '../grid'
import {habitHistoryQuery, type HabitResponse, type PeriodResponse} from '../queries'
import {useHabitsSync} from '../sync/useHabitsSync'
import {useHabitTicks} from '../useHabitTicks'

const props = defineProps<{
	habit: HabitResponse
}>()

const {t, locale} = useI18n({useScope: 'global'})

const range = computed(() => gridRange(props.habit.kind, props.habit.progress.today))
// A computed query: when the day (and so the range) changes, it fetches again.
const history = useQuery(computed(() => habitHistoryQuery(props.habit.id, range.value.from, range.value.to)))

const columns = computed(() => gridColumns(history.data.value?.periods ?? [], range.value.from))

const editable = computed(() => props.habit.kind === 'daily')

const ticks = useHabitTicks()

// The grid comes from the server (online only); after the queue has sent
// a change, fetch it again so the change shows.
const sync = useHabitsSync()
watch(sync.lastSyncedAt, () => void history.refetch())

function toggle(period: PeriodResponse) {
	const uid = history.data.value?.entries.get(period.start)?.at(-1)
	if (uid) ticks.remove(props.habit, uid)
	else ticks.add(props.habit, period.start)
}

/** Bad habits: darker the more times it happened. */
function countClass(period: PeriodResponse) {
	return period.status === 'count' && period.count > 0 ? `has-count-${Math.min(period.count, 3)}` : ''
}

function cellTitle(period: PeriodResponse) {
	const date = fromKey(period.start).toLocaleDateString(locale.value, {weekday: 'short', day: 'numeric', month: 'short'})
	const status = period.status === 'count'
		? t('household.habits.grid.times', {count: period.count}, period.count)
		: t(`household.habits.grid.${period.status}`, {count: period.count})
	return props.habit.kind === 'weekly'
		? t('household.habits.grid.weekOf', {date, status})
		: `${date}: ${status}`
}
</script>

<style lang="scss" scoped>
.habit-grid {
	overflow-x: auto;
}

.days {
	display: flex;
	gap: 3px;
}

.column-week {
	display: flex;
	flex-direction: column;
	gap: 3px;
}

.weeks {
	display: flex;
	flex-wrap: wrap;
	gap: 3px;
	margin: 0;
	padding: 0;
	list-style: none;
}

.cell {
	display: block;
	inline-size: 1rem;
	block-size: 1rem;
	padding: 0;
	border: 1px solid transparent;
	border-radius: 3px;
	background: var(--grey-200);
}

.weeks .cell {
	inline-size: 1.4rem;
	block-size: 1.4rem;
}

button.cell {
	cursor: pointer;

	&:focus-visible {
		outline: 2px solid var(--habit-color);
		outline-offset: 1px;
	}
}

.is-empty {
	background: transparent;
}

.is-done {
	background: var(--habit-color);
}

.is-partial {
	background: color-mix(in srgb, var(--habit-color) 40%, var(--grey-100));
}

.is-missed {
	background: var(--grey-300);
}

.is-pending {
	border-color: var(--habit-color);
	background: transparent;
}

.is-skipped {
	background: repeating-linear-gradient(45deg, var(--grey-200) 0 2px, transparent 2px 5px);
}

.is-paused {
	border: 1px dashed var(--grey-300);
	background: transparent;
}

// Not a habit day: an outline, so it reads as "nothing expected".
.is-rest {
	border-color: var(--grey-300);
	background: transparent;
}

// Before the habit started: there, but faint.
.is-before {
	background: var(--grey-200);
	opacity: .4;
}

.is-count {
	background: var(--grey-200);
}

.has-count-1 {
	background: color-mix(in srgb, var(--grey-600) 35%, transparent);
}

.has-count-2 {
	background: color-mix(in srgb, var(--grey-600) 65%, transparent);
}

.has-count-3 {
	background: var(--grey-700);
}
</style>
