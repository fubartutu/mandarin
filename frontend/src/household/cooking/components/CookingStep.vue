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
	</div>
</template>

<script setup lang="ts">
import type {CookingStep} from '../steps'

defineProps<{
	step: CookingStep
	count: number
}>()
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
</style>
