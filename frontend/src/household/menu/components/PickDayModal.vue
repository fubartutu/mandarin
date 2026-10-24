<template>
	<!-- Vikunja's Modal (a native <dialog>) around a Card, like CreateEdit,
		so it looks like the app's other dialogs and fits a phone screen. -->
	<Modal
		:aria-label="title"
		@close="$emit('close')"
	>
		<Card
			:title="title"
			:shadow="false"
			:show-close="true"
			class="has-text-start pick-day"
			@close="$emit('close')"
		>
			<form
				@submit.prevent="confirm"
			>
				<label
					class="label"
					:for="inputId"
				>{{ $t('household.menu.pickDay') }}</label>
				<input
					:id="inputId"
					v-model="picked"
					class="input"
					type="date"
					required
				>
			</form>

			<template #footer>
				<XButton
					variant="secondary"
					:shadow="false"
					@click="$emit('close')"
				>
					{{ $t('household.menu.cancel') }}
				</XButton>
				<XButton
					variant="primary"
					:shadow="false"
					:disabled="!canConfirm"
					:loading="busy"
					@click="confirm"
				>
					{{ confirmLabel }}
				</XButton>
			</template>
		</Card>
	</Modal>
</template>

<script setup lang="ts">
import {computed, ref, useId, watch} from 'vue'

const props = withDefaults(defineProps<{
	title: string
	/** The confirm button's text, e.g. "Plan for dinner" or "Move". */
	confirmLabel: string
	/** The day the picker starts on, YYYY-MM-DD. */
	day: string
	/** While the caller saves: the confirm button is disabled. */
	busy?: boolean
}>(), {
	busy: false,
})

const emit = defineEmits<{
	pick: [day: string]
	close: []
}>()

// useId() gives ids unique to this copy of the component, for the label's `for`.
const inputId = useId()

// A native date input's value is already YYYY-MM-DD, or '' when cleared.
// Starts on `day`, and follows it if the caller changes it.
const picked = ref('')
watch(() => props.day, day => {
	picked.value = day
}, {immediate: true})

const canConfirm = computed(() => !props.busy && /^\d{4}-\d{2}-\d{2}$/.test(picked.value))

function confirm() {
	if (canConfirm.value) emit('pick', picked.value)
}
</script>

<style lang="scss" scoped>
.pick-day :deep(.card-footer) {
	display: flex;
	justify-content: flex-end;
	gap: .5rem;
	padding: .75rem 1rem;
}
</style>
