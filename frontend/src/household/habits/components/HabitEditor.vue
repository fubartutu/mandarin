<template>
	<Modal
		:aria-label="heading"
		@close="$emit('close')"
	>
		<Card
			:title="heading"
			:shadow="false"
			:show-close="true"
			class="has-text-start habit-editor"
			@close="$emit('close')"
		>
			<form @submit.prevent="save">
				<label class="field">
					<span class="label">{{ $t('household.habits.edit.title') }}</span>
					<input
						v-model="draft.title"
						v-focus
						class="input"
						type="text"
						required
						maxlength="250"
						:placeholder="$t('household.habits.edit.titlePlaceholder')"
					>
				</label>

				<!-- The kind can't change once there are ticks, so only new habits pick one. -->
				<fieldset
					v-if="!habit"
					class="field"
				>
					<legend class="label">
						{{ $t('household.habits.edit.kind') }}
					</legend>
					<label
						v-for="kind in KINDS"
						:key="kind"
						class="kind-option"
					>
						<input
							v-model="draft.kind"
							type="radio"
							name="kind"
							:value="kind"
						>
						<span>
							<strong>{{ $t(`household.habits.kinds.${kind}`) }}</strong>
							<span class="help">{{ $t(`household.habits.kinds.${kind}Help`) }}</span>
						</span>
					</label>
				</fieldset>
				<p
					v-else
					class="field help"
				>
					{{ $t(`household.habits.kinds.${draft.kind}`) }}: {{ $t(`household.habits.kinds.${draft.kind}Help`) }}
				</p>

				<fieldset
					v-if="draft.kind === 'daily' || draft.kind === 'daily_count'"
					class="field"
				>
					<legend class="label">
						{{ $t('household.habits.edit.days') }}
					</legend>
					<div class="weekdays">
						<label
							v-for="(name, i) in weekdayNames"
							:key="i"
							class="weekday"
						>
							<input
								type="checkbox"
								:checked="draft.weekdays[i] === '1'"
								@change="setWeekday(i, ($event.target as HTMLInputElement).checked)"
							>
							<span>{{ name }}</span>
						</label>
					</div>
				</fieldset>

				<label
					v-if="draft.kind === 'daily_count' || draft.kind === 'weekly'"
					class="field goal"
				>
					<span class="label">{{ draft.kind === 'weekly' ? $t('household.habits.edit.goalWeek') : $t('household.habits.edit.goalDay') }}</span>
					<input
						v-model.number="draft.goal"
						class="input"
						type="number"
						inputmode="numeric"
						min="1"
						max="99"
						required
					>
				</label>

				<label
					v-if="draft.kind !== 'bad'"
					class="field"
				>
					<span class="label">{{ $t('household.habits.edit.routine') }}</span>
					<div class="select is-fullwidth">
						<select v-model="draft.groupId">
							<option :value="0">
								{{ $t('household.habits.edit.noRoutine') }}
							</option>
							<option
								v-for="group in groups"
								:key="group.id"
								:value="group.id"
							>
								{{ group.title }}
							</option>
						</select>
					</div>
				</label>

				<fieldset class="field">
					<legend class="label">
						{{ $t('household.habits.edit.color') }}
					</legend>
					<div class="colors">
						<label
							v-for="color in HABIT_COLORS"
							:key="color"
							class="swatch"
							:style="colorStyle(color)"
						>
							<input
								v-model="draft.color"
								type="radio"
								name="color"
								:value="color"
								:aria-label="color ? `#${color}` : $t('household.habits.edit.defaultColor')"
							>
						</label>
					</div>
				</fieldset>

				<fieldset
					v-if="draft.kind !== 'bad'"
					class="field"
				>
					<legend class="label">
						{{ $t('household.habits.edit.reminders') }}
					</legend>
					<div class="reminders">
						<span
							v-for="(time, i) in draft.reminderTimes"
							:key="i"
							class="reminder"
						>
							<input
								v-model="draft.reminderTimes[i]"
								class="input"
								type="time"
								required
								:aria-label="$t('household.habits.edit.reminders')"
							>
							<button
								type="button"
								class="button is-small is-text"
								:title="$t('household.habits.edit.removeReminder', {time})"
								:aria-label="$t('household.habits.edit.removeReminder', {time})"
								@click="draft.reminderTimes.splice(i, 1)"
							>
								<Icon icon="times" />
							</button>
						</span>
						<XButton
							v-if="draft.reminderTimes.length < MAX_REMINDERS"
							variant="secondary"
							icon="plus"
							:shadow="false"
							@click="addReminder"
						>
							{{ $t('household.habits.edit.addReminder') }}
						</XButton>
					</div>
					<p class="help">
						{{ $t('household.habits.edit.remindersHint') }}
						<RouterLink :to="{name: 'user.settings.household-notifications'}">
							{{ $t('household.habits.edit.remindersSettings') }}
						</RouterLink>
					</p>
				</fieldset>

				<label class="checkbox field">
					<input
						v-model="draft.shared"
						type="checkbox"
						:disabled="habit !== undefined && habit.shared && !isOwner"
					>
					{{ $t('household.habits.edit.shared') }}
					<span class="help">{{ $t('household.habits.edit.sharedHelp') }}</span>
				</label>
			</form>

			<template #footer>
				<XButton
					v-if="habit"
					variant="secondary"
					danger
					class="delete-button"
					:shadow="false"
					@click="remove"
				>
					{{ $t('household.habits.edit.delete') }}
				</XButton>
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
					:loading="saving.isPending.value"
					:disabled="!canSave"
					@click="save"
				>
					{{ $t('household.habits.edit.save') }}
				</XButton>
			</template>
		</Card>
	</Modal>
</template>

<script setup lang="ts">
import {computed, reactive} from 'vue'
import {useI18n} from 'vue-i18n'

import {useAuthStore} from '@/stores/auth'

import {HABIT_COLORS, colorStyle} from '../colors'
import {
	type HabitDraft,
	type HabitGroupResponse,
	type HabitKind,
	type HabitResponse,
	useDeleteHabitMutation,
	useSaveHabitMutation,
} from '../queries'

const props = defineProps<{
	/** The habit to edit; left out to make a new one. */
	habit?: HabitResponse
	groups: HabitGroupResponse[]
	/** The routine a new habit goes in (its "+" was used). */
	groupId?: number
}>()

const emit = defineEmits<{
	close: []
}>()

const KINDS: HabitKind[] = ['daily', 'daily_count', 'weekly', 'bad']
// The server takes at most 8 reminder times a habit.
const MAX_REMINDERS = 8

const {t, locale} = useI18n({useScope: 'global'})
const authStore = useAuthStore()

const heading = computed(() => props.habit ? t('household.habits.edit.editTitle') : t('household.habits.edit.newTitle'))
const isOwner = computed(() => props.habit?.ownerId === authStore.info?.id)

// A copy to edit, so Cancel leaves the habit as it was. reactive() makes
// the plain object's fields trackable, like a ref for each.
// eslint-disable-next-line vue/no-setup-props-reactivity-loss -- seeded once so a refetch keeps the draft; mounted fresh per edit
const draft = reactive<HabitDraft>({
	title: props.habit?.title ?? '',
	kind: props.habit?.kind ?? 'daily',
	weekdays: props.habit?.weekdays ?? '1111111',
	goal: props.habit?.goal ?? 1,
	color: props.habit?.color ?? '',
	groupId: props.habit?.groupId ?? props.groupId ?? 0,
	shared: props.habit?.shared ?? false,
	reminderTimes: [...(props.habit?.reminderTimes ?? [])],
})

// Monday to Sunday in the user's language (2024-01-01 was a Monday).
const weekdayNames = computed(() => Array.from({length: 7}, (_, i) =>
	new Date(2024, 0, 1 + i).toLocaleDateString(locale.value, {weekday: 'short'})))

function setWeekday(i: number, on: boolean) {
	const days = [...draft.weekdays]
	days[i] = on ? '1' : '0'
	draft.weekdays = days.join('')
}

const saving = useSaveHabitMutation()
const deleting = useDeleteHabitMutation()

const canSave = computed(() => draft.title.trim() !== '' && draft.weekdays.includes('1') &&
	draft.goal >= 1 && draft.goal <= 99 && !saving.isPending.value)

/** A new reminder: an hour after the last one, or 9:00 for the first. */
function addReminder() {
	const last = draft.reminderTimes.at(-1)
	const hour = last ? Math.min(Number(last.slice(0, 2)) + 1, 23) : 9
	draft.reminderTimes.push(`${String(hour).padStart(2, '0')}:00`)
}

function save() {
	if (!canSave.value) return
	// Only the kinds that use them keep days, a goal and a routine.
	const body: HabitDraft = {
		...draft,
		title: draft.title.trim(),
		weekdays: draft.kind === 'daily' || draft.kind === 'daily_count' ? draft.weekdays : '1111111',
		goal: draft.kind === 'daily_count' || draft.kind === 'weekly' ? draft.goal : 1,
		groupId: draft.kind === 'bad' ? 0 : draft.groupId,
		// Sorted and without repeats; bad habits never remind.
		reminderTimes: draft.kind === 'bad' ? [] : [...new Set(draft.reminderTimes.filter(t => t !== ''))].sort(),
	}
	saving.mutate({habit: props.habit, draft: body}, {onSuccess: () => emit('close')})
}

function remove() {
	if (!props.habit) return
	if (!window.confirm(t('household.habits.edit.deleteConfirm', {title: props.habit.title}))) return
	deleting.mutate(props.habit.id, {onSuccess: () => emit('close')})
}
</script>

<style lang="scss" scoped>
.habit-editor :deep(.card-footer) {
	display: flex;
	flex-wrap: wrap;
	justify-content: flex-end;
	gap: .5rem;
	padding: .75rem 1rem;
}

.delete-button {
	margin-inline-end: auto;
}

.field {
	display: block;
	margin-block-end: 1rem;
	border: 0;
	padding: 0;
}

.help {
	display: block;
	color: var(--grey-600);
}

.kind-option {
	display: flex;
	gap: .5rem;
	align-items: baseline;
	padding: .25rem 0;
	cursor: pointer;
}

.weekdays,
.colors {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
}

.weekday {
	display: flex;
	flex-direction: column;
	align-items: center;
	cursor: pointer;
}

.goal .input {
	max-inline-size: 6rem;
}

.reminders {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .5rem;
}

.reminder {
	display: inline-flex;
	align-items: center;

	.input {
		inline-size: 7.5rem;
	}
}

.swatch {
	display: inline-flex;
	inline-size: 2rem;
	block-size: 2rem;
	border-radius: 50%;
	background: var(--habit-color);
	cursor: pointer;

	input {
		inline-size: 100%;
		block-size: 100%;
		margin: 0;
		opacity: 0;
		cursor: pointer;
	}

	&:has(input:checked) {
		outline: 3px solid var(--grey-700);
		outline-offset: 2px;
	}

	&:has(input:focus-visible) {
		outline: 3px solid var(--primary);
		outline-offset: 2px;
	}
}
</style>
