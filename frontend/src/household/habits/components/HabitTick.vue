<template>
	<!-- aria-pressed makes it a toggle for screen readers when it is one. -->
	<button
		type="button"
		class="habit-tick"
		:class="`is-${state}`"
		:style="colorStyle(habit.color)"
		:aria-pressed="toggles ? habit.progress.done_today : undefined"
		:aria-label="label"
		:title="label"
		@click="ticks.tap(habit)"
	>
		<Icon
			v-if="habit.kind === 'bad'"
			icon="plus"
		/>
		<span
			v-else-if="habit.progress.goal > 1"
			class="count"
		>{{ habit.progress.today_count }}/{{ habit.progress.goal }}</span>
		<Icon
			v-else-if="habit.progress.done_today"
			icon="check"
		/>
	</button>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'

import {colorStyle} from '../colors'
import type {HabitResponse} from '../queries'
import {habitState} from '../today'
import {useHabitTicks} from '../useHabitTicks'

// The round button that ticks a habit off (see useHabitTicks for what a tap does).

const props = defineProps<{
	habit: HabitResponse
}>()

const {t} = useI18n({useScope: 'global'})
const ticks = useHabitTicks()

const state = computed(() => habitState(props.habit))
const toggles = computed(() => props.habit.kind !== 'bad' && props.habit.progress.goal === 1)

const label = computed(() => {
	const title = props.habit.title
	if (props.habit.kind === 'bad') return t('household.habits.tick.happened', {title})
	if (toggles.value && props.habit.progress.done_today) return t('household.habits.tick.undo', {title})
	return t('household.habits.tick.do', {title})
})
</script>

<style lang="scss" scoped>
.habit-tick {
	--size: 2.5rem;

	display: inline-flex;
	flex: none;
	align-items: center;
	justify-content: center;
	inline-size: var(--size);
	block-size: var(--size);
	padding: 0;
	border: 2px solid var(--habit-color);
	border-radius: 50%;
	background: transparent;
	color: var(--habit-color);
	cursor: pointer;
	font-size: .8rem;
	font-weight: 700;
	transition: background-color $transition, color $transition, transform $transition;

	&:active {
		transform: scale(.92);
	}

	&:focus-visible {
		outline: 2px solid var(--habit-color);
		outline-offset: 2px;
	}
}

.is-done {
	background: var(--habit-color);
	color: var(--white);
}

// Not due today: still tappable (an extra tick counts), but quieter.
.is-rest,
.is-skipped,
.is-paused {
	border-style: dashed;
	opacity: .6;
}

.is-bad {
	border-color: var(--grey-400);
	color: var(--grey-600);
}
</style>
