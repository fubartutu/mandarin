<template>
	<!-- One banner per finished timer, stacked; each stays until dismissed. -->
	<div class="timer-banners">
		<div
			v-for="timer in finished"
			:key="timer.id"
			class="timer-banner"
			role="alert"
		>
			<Icon
				icon="stopwatch"
				class="timer-banner-icon"
			/>
			<span class="timer-banner-text">
				{{ $t('household.cooking.timerDone', {name: timerName(timer)}) }}
			</span>
			<button
				type="button"
				class="timer-dismiss"
				@click="emit('dismiss', timer.id)"
			>
				{{ $t('household.cooking.dismiss') }}
			</button>
		</div>
	</div>
</template>

<script setup lang="ts">
import {useTimerName} from '../timerName'
import type {CookingTimer} from '../useCookingTimers'

defineProps<{
	finished: CookingTimer[]
}>()

const emit = defineEmits<{
	dismiss: [id: string]
}>()

const timerName = useTimerName()
</script>

<style lang="scss" scoped>
.timer-banners {
	display: flex;
	flex-direction: column;
	gap: .5rem;
}

.timer-banner {
	display: flex;
	align-items: center;
	gap: .75rem;
	padding: .75rem 1rem;
	border-radius: $radius;
	background: var(--primary);
	color: var(--white);
	box-shadow: var(--shadow-md);
	font-size: 1.15rem;
	font-weight: 600;
}

.timer-banner-icon {
	flex: none;
}

.timer-banner-text {
	flex: 1;
	min-inline-size: 0;
}

.timer-dismiss {
	flex: none;
	min-block-size: 2.75rem;
	padding: .4rem 1rem;
	border: 2px solid var(--white);
	border-radius: $radius;
	background: transparent;
	color: var(--white);
	font: inherit;
	cursor: pointer;
}
</style>
