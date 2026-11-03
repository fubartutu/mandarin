<template>
	<Modal
		:aria-label="$t('household.habits.routines.title')"
		@close="$emit('close')"
	>
		<Card
			:title="$t('household.habits.routines.title')"
			:shadow="false"
			:show-close="true"
			class="has-text-start routines-editor"
			@close="$emit('close')"
		>
			<p class="help">
				{{ $t('household.habits.routines.help') }}
			</p>
			<ul class="routines">
				<li
					v-for="group in sorted"
					:key="group.id"
					class="routine"
				>
					<input
						class="input"
						type="text"
						maxlength="250"
						:value="group.title"
						:aria-label="$t('household.habits.routines.name')"
						@change="rename(group, ($event.target as HTMLInputElement).value)"
					>
					<label class="checkbox">
						<input
							type="checkbox"
							:checked="group.shared"
							:disabled="group.shared && group.ownerId !== authStore.info?.id"
							@change="save(group, {shared: ($event.target as HTMLInputElement).checked})"
						>
						{{ $t('household.habits.routines.shared') }}
					</label>
					<button
						type="button"
						class="button is-small is-text"
						:title="$t('household.habits.routines.delete', {title: group.title})"
						:aria-label="$t('household.habits.routines.delete', {title: group.title})"
						@click="remove(group)"
					>
						<Icon icon="trash-alt" />
					</button>
				</li>
			</ul>
			<form
				class="routine add"
				@submit.prevent="add"
			>
				<input
					v-model="newTitle"
					class="input"
					type="text"
					maxlength="250"
					:placeholder="$t('household.habits.routines.newPlaceholder')"
					:aria-label="$t('household.habits.routines.name')"
				>
				<XButton
					variant="secondary"
					:shadow="false"
					:disabled="newTitle.trim() === ''"
					@click="add"
				>
					{{ $t('household.habits.routines.add') }}
				</XButton>
			</form>
		</Card>
	</Modal>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useI18n} from 'vue-i18n'

import {useAuthStore} from '@/stores/auth'

import {type HabitGroupResponse, useDeleteGroupMutation, useSaveGroupMutation} from '../queries'

// Routines are groups of habits done together, e.g. "Morning". Personal ones
// are only seen by whoever made them; shared ones by the whole household.

const props = defineProps<{
	groups: HabitGroupResponse[]
}>()

defineEmits<{
	close: []
}>()

const {t} = useI18n({useScope: 'global'})
const authStore = useAuthStore()
const saving = useSaveGroupMutation()
const deleting = useDeleteGroupMutation()

const sorted = computed(() => [...props.groups].sort((a, b) => a.position - b.position || a.id - b.id))
const newTitle = ref('')

function save(group: HabitGroupResponse, changes: Partial<Pick<HabitGroupResponse, 'title' | 'shared'>>) {
	saving.mutate({...group, ...changes})
}

function rename(group: HabitGroupResponse, title: string) {
	// Nothing to write when the name is unchanged or emptied.
	if (title.trim() !== '' && title.trim() !== group.title) save(group, {title: title.trim()})
}

function add() {
	const title = newTitle.value.trim()
	if (title === '') return
	saving.mutate({id: 0, title, shared: false, position: 0}, {onSuccess: () => {
		newTitle.value = ''
	}})
}

function remove(group: HabitGroupResponse) {
	if (window.confirm(t('household.habits.routines.deleteConfirm', {title: group.title}))) deleting.mutate(group.id)
}
</script>

<style lang="scss" scoped>
.help {
	color: var(--grey-600);
}

.routines {
	margin: 0 0 1rem;
	padding: 0;
	list-style: none;
}

.routine {
	display: flex;
	align-items: center;
	gap: .5rem;
	padding-block: .25rem;

	.input {
		flex: 1;
	}
}

.checkbox {
	white-space: nowrap;
}
</style>
