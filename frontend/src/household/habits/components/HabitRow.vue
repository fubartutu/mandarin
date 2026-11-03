<template>
	<li
		class="habit-row"
		:class="`is-${state}`"
	>
		<div class="habit-main">
			<HabitTick :habit="habit" />
			<button
				type="button"
				class="habit-text"
				:aria-expanded="open"
				@click="open = !open"
			>
				<span class="habit-title">
					{{ habit.title }}
					<Icon
						v-if="habit.shared"
						icon="users"
						class="shared-icon"
						:title="$t('household.habits.sharedRoutine')"
					/>
				</span>
				<span class="habit-status">
					<span
						v-if="streak > 0"
						class="streak"
						:class="{'is-grace': habit.progress.grace_used}"
						:title="habit.progress.grace_used ? $t('household.habits.graceHelp') : undefined"
					>
						<Icon icon="fire" />
						{{ streakText }}
					</span>
					<span>{{ statusText }}</span>
					<span
						v-if="habit.reminderTimes.length > 0 && habit.kind !== 'bad'"
						class="reminder-times"
					>
						<Icon icon="bell" />
						{{ habit.reminderTimes.join(', ') }}
					</span>
				</span>
			</button>
			<button
				v-if="habit.progress.today_count > 0 && (habit.kind === 'bad' || habit.progress.goal > 1)"
				type="button"
				class="button is-small is-text undo"
				:title="$t('household.habits.tick.undo', {title: habit.title})"
				:aria-label="$t('household.habits.tick.undo', {title: habit.title})"
				@click="ticks.undo(habit)"
			>
				<Icon icon="minus" />
			</button>
		</div>

		<div
			v-if="open"
			class="habit-details"
		>
			<HabitGrid :habit="habit" />
			<p
				v-if="habit.kind !== 'bad'"
				class="stats"
			>
				{{ recentText }}
				<template v-if="habit.progress.best_streak > 0">
					· {{ $t(`household.habits.best.${unit}`, {count: habit.progress.best_streak}, habit.progress.best_streak) }}
				</template>
			</p>
			<p
				v-else
				class="stats"
			>
				{{ $t('household.habits.lastWeek', {count: habit.progress.last_week}, habit.progress.last_week) }}
			</p>
			<div class="actions">
				<XButton
					v-if="habit.kind !== 'bad' && !habit.paused && !habit.progress.done_today"
					variant="secondary"
					:shadow="false"
					@click="ticks.skip(habit, !habit.progress.skipped)"
				>
					{{ habit.progress.skipped
						? $t(habit.kind === 'weekly' ? 'household.habits.unskipWeek' : 'household.habits.unskipToday')
						: $t(habit.kind === 'weekly' ? 'household.habits.skipWeek' : 'household.habits.skipToday') }}
				</XButton>
				<XButton
					v-if="habit.kind !== 'bad'"
					variant="secondary"
					:shadow="false"
					@click="change.mutate({habit, changes: {paused: !habit.paused}})"
				>
					{{ habit.paused ? $t('household.habits.resume') : $t('household.habits.pause') }}
				</XButton>
				<XButton
					variant="secondary"
					:shadow="false"
					@click="change.mutate({habit, changes: {archived: !habit.archived}})"
				>
					{{ habit.archived ? $t('household.habits.unarchive') : $t('household.habits.archive') }}
				</XButton>
				<XButton
					variant="secondary"
					:shadow="false"
					@click="$emit('edit', habit)"
				>
					{{ $t('household.habits.edit.editTitle') }}
				</XButton>
			</div>
		</div>
	</li>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useI18n} from 'vue-i18n'

import {streakUnit} from '../grid'
import {type HabitResponse, useChangeHabitMutation} from '../queries'
import {habitState} from '../today'
import {useHabitTicks} from '../useHabitTicks'
import HabitGrid from './HabitGrid.vue'
import HabitTick from './HabitTick.vue'

const props = defineProps<{
	habit: HabitResponse
}>()

defineEmits<{
	edit: [habit: HabitResponse]
}>()

const {t} = useI18n({useScope: 'global'})
const ticks = useHabitTicks()
const change = useChangeHabitMutation()

const open = ref(false)
const state = computed(() => habitState(props.habit))
const unit = computed(() => streakUnit(props.habit.kind))
const streak = computed(() => props.habit.progress.streak)

const streakText = computed(() => t(`household.habits.streak.${unit.value}`, {count: streak.value}, streak.value))

// The line under the title: what today (this week) looks like.
const statusText = computed(() => {
	const p = props.habit.progress
	switch (state.value) {
		case 'bad': return t('household.habits.thisWeekTimes', {count: p.this_week}, p.this_week)
		case 'paused': return t('household.habits.paused')
		case 'skipped': return t(props.habit.kind === 'weekly' ? 'household.habits.skippedWeek' : 'household.habits.skippedToday')
		case 'rest': return t('household.habits.restDay')
	}
	if (props.habit.kind === 'weekly') return t('household.habits.weekProgress', {count: p.today_count, goal: p.goal})
	if (p.goal > 1) return t('household.habits.dayProgress', {count: p.today_count, goal: p.goal})
	return p.done_today ? t('household.habits.doneToday') : t('household.habits.dueToday')
})

const recentText = computed(() => {
	const p = props.habit.progress
	return t(`household.habits.recent.${unit.value}`, {done: p.done, due: p.due})
})
</script>

<style lang="scss" scoped>
.habit-row {
	padding: .5rem 0;
}

.habit-row + .habit-row {
	border-block-start: 1px solid var(--grey-200);
}

.habit-main {
	display: flex;
	align-items: center;
	gap: .75rem;
}

.habit-text {
	display: flex;
	flex: 1;
	flex-direction: column;
	min-inline-size: 0;
	padding: 0;
	border: 0;
	background: transparent;
	color: var(--text);
	cursor: pointer;
	text-align: start;
}

.habit-title {
	font-weight: 600;
}

.is-done .habit-title {
	color: var(--grey-600);
}

.shared-icon {
	margin-inline-start: .25rem;
	color: var(--grey-500);
	font-size: .8em;
}

.habit-status {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
	color: var(--grey-600);
	font-size: .85rem;
}

.reminder-times {
	color: var(--grey-500);
}

.streak {
	color: var(--primary);
	font-weight: 600;
}

// Alive thanks to a forgiven miss: one more and it ends.
.streak.is-grace {
	color: var(--grey-700);
	font-style: italic;
}

.habit-details {
	padding-block: .75rem .25rem;
	padding-inline-start: 3.25rem;
}

.stats {
	margin-block: .5rem;
	color: var(--grey-600);
	font-size: .9rem;
}

.actions {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
}
</style>
