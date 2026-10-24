<template>
	<section
		class="timer-tray"
		:aria-label="$t('household.cooking.timers')"
	>
		<ul>
			<li
				v-for="timer in running"
				:key="timer.id"
				class="timer-row"
			>
				<span class="timer-name">{{ timerName(timer) }}</span>
				<span class="timer-time">{{ formatRemaining(timer.remaining) }}</span>
				<BaseButton
					class="timer-cancel"
					:aria-label="$t('household.cooking.cancelTimer', {name: timerName(timer)})"
					:title="$t('household.cooking.cancelTimer', {name: timerName(timer)})"
					@click="emit('cancel', timer.id)"
				>
					<Icon icon="xmark" />
				</BaseButton>
			</li>
		</ul>
	</section>
</template>

<script setup lang="ts">
import {useI18n} from 'vue-i18n'

import BaseButton from '@/components/base/BaseButton.vue'

import {formatRemaining} from '../timers'
import type {CookingTimer, RunningTimer} from '../useCookingTimers'

defineProps<{
	running: RunningTimer[] // in order of time left
}>()

const emit = defineEmits<{
	cancel: [id: string]
}>()

const {t} = useI18n({useScope: 'global'})
const timerName = (timer: CookingTimer) => t('household.cooking.timerName', {number: timer.stepNumber, label: timer.label})
</script>

<style lang="scss" scoped>
.timer-tray ul {
	display: flex;
	flex-direction: column;
	gap: .5rem;
	margin: 0;
	padding: 0;
	list-style: none;
}

.timer-row {
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: .5rem .5rem .5rem 1rem;
	border-radius: $radius;
	background: var(--grey-100);
}

.timer-name {
	flex: 1;
	min-inline-size: 0;
	color: var(--grey-700);
	font-weight: 600;
}

.timer-time {
	font-size: 1.75rem;
	font-weight: 700;
	font-variant-numeric: tabular-nums;
}

.timer-cancel {
	display: flex;
	flex: none;
	align-items: center;
	justify-content: center;
	inline-size: 2.75rem;
	block-size: 2.75rem;
	border-radius: 50%;
	font-size: 1.1rem;
}
</style>
