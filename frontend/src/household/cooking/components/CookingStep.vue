<template>
	<div class="cooking-step">
		<p class="step-progress">
			<span>{{ $t('household.cooking.stepOf', {number: step.number, count}) }}</span>
			<span
				v-if="step.section"
				class="step-section"
			>{{ step.section }}</span>
		</p>
		<p
			class="step-text"
			aria-live="polite"
		>
			{{ step.text }}
		</p>

		<!-- One chip per time in the step; a running one shows its time left. -->
		<div
			v-if="chips.length > 0"
			class="timer-chips"
		>
			<button
				v-for="chip in chips"
				:key="chip.label"
				type="button"
				class="timer-chip"
				:disabled="chip.running !== undefined"
				:aria-label="chip.running
					? $t('household.cooking.timerRunning', {label: chip.label, time: formatRemaining(chip.running.remaining)})
					: $t('household.cooking.startTimer', {label: chip.label})"
				@click="emit('startTimer', chip)"
			>
				<Icon icon="stopwatch" />
				<span :class="{'timer-chip-time': chip.running}">
					{{ chip.running ? formatRemaining(chip.running.remaining) : chip.label }}
				</span>
			</button>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import type {CookingStep} from '../steps'
import {findTimers, formatRemaining, type DetectedTimer} from '../timers'
import type {RunningTimer} from '../useCookingTimers'

const props = defineProps<{
	step: CookingStep
	count: number
	running: RunningTimer[] // every running timer; this step picks its own
}>()

const emit = defineEmits<{
	startTimer: [timer: DetectedTimer]
}>()

const chips = computed(() => findTimers(props.step.text).map(found => ({
	...found,
	running: props.running.find(t => t.stepNumber === props.step.number && t.label === found.label),
})))
</script>

<style lang="scss" scoped>
.step-progress {
	display: flex;
	flex-wrap: wrap;
	gap: .25rem 1rem;
	margin-block-end: 1rem;
	color: var(--grey-600);
	font-weight: 600;
}

.step-section {
	color: var(--primary);
}

// Large enough to read from across the bench.
.step-text {
	font-size: clamp(1.5rem, 1.1rem + 2vw, 2.4rem);
	line-height: 1.4;
	white-space: pre-line;
}

.timer-chips {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
	margin-block-start: 1rem;
}

.timer-chip {
	display: inline-flex;
	align-items: center;
	gap: .5rem;
	min-block-size: 2.75rem;
	padding: .4rem 1rem;
	border: 1px solid var(--primary);
	border-radius: 999px;
	background: transparent;
	color: var(--primary);
	font: inherit;
	font-size: 1.1rem;
	font-weight: 600;
	cursor: pointer;

	&:disabled {
		border-color: transparent;
		background: var(--grey-100);
		color: var(--text);
		cursor: default;
	}
}

.timer-chip-time {
	font-variant-numeric: tabular-nums;
}
</style>
