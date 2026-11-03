<template>
	<!-- Mandarin: "Snoozed until 10:40 · Cancel" under Mark done on the task page (FORK in TaskDetailView.vue),
	     after Snooze 1h on a reminder notification. Only the viewer's own snooze. -->
	<p
		v-if="until"
		class="household-snooze"
	>
		<Icon icon="bell" />
		<span>{{ $t('household.push.snooze.until', {time: label}) }}</span>
		<BaseButton
			class="household-snooze__cancel"
			:disabled="cancelMutation.isPending.value"
			@click="cancel"
		>
			{{ $t('household.push.snooze.cancel') }}
		</BaseButton>
	</p>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'

import BaseButton from '@/components/base/BaseButton.vue'
import {useGlobalNow} from '@/composables/useGlobalNow'
import {success} from '@/message'
import {snoozeLabel} from './snoozeLabel'
import {taskSnoozeQuery, useCancelSnoozeMutation} from './snoozeQueries'

const props = defineProps<{
	taskId: number
}>()

const {t, locale} = useI18n({useScope: 'global'})
// Ticks every minute, so the notice goes once the reminder has come back.
const {now} = useGlobalNow()

const snooze = useQuery(computed(() => ({
	...taskSnoozeQuery(props.taskId),
	enabled: props.taskId > 0,
})))
const until = computed(() => {
	const at = snooze.data.value
	return at && at > now.value ? at : null
})
const label = computed(() => until.value ? snoozeLabel(until.value, now.value, locale.value) : '')

const cancelMutation = useCancelSnoozeMutation()

async function cancel() {
	await cancelMutation.mutateAsync(props.taskId)
	success({message: t('household.push.snooze.cancelled')})
}
</script>

<style lang="scss" scoped>
.household-snooze {
	display: flex;
	align-items: center;
	flex-wrap: wrap;
	gap: .4rem;
	margin: .25rem 0 .75rem;
	font-size: .9rem;
	color: var(--grey-600);
}

.household-snooze__cancel {
	color: var(--primary);
	text-decoration: underline;
}
</style>
