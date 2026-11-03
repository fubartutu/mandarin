<template>
	<div class="habits-page">
		<header class="habits-header">
			<h1 class="title">
				{{ $t('household.habits.title') }}
			</h1>
			<div class="header-actions">
				<XButton
					variant="secondary"
					:shadow="false"
					@click="routinesOpen = true"
				>
					{{ $t('household.habits.routines.title') }}
				</XButton>
				<XButton
					icon="plus"
					:shadow="false"
					@click="editing = {}"
				>
					{{ $t('household.habits.add') }}
				</XButton>
			</div>
		</header>

		<p
			v-if="habits.isPending.value"
			class="has-text-centered has-text-grey"
		>
			{{ $t('household.home.loading') }}
		</p>
		<p
			v-else-if="habits.isError.value && !habits.data.value"
			class="has-text-centered has-text-grey is-italic"
		>
			{{ $t('household.habits.notLoaded') }}
		</p>
		<div
			v-else-if="list.length === 0"
			class="empty has-text-centered"
		>
			<p>{{ $t('household.habits.empty') }}</p>
			<XButton
				icon="plus"
				:shadow="false"
				@click="editing = {}"
			>
				{{ $t('household.habits.addFirst') }}
			</XButton>
		</div>

		<template v-else>
			<p
				v-if="summary.total > 0"
				class="today-summary"
			>
				{{ $t('household.habits.summary', {done: summary.done, total: summary.total}) }}
			</p>

			<section
				v-for="section in sections"
				:key="section.group?.id ?? 0"
				class="habit-section card"
			>
				<h2 class="section-title">
					{{ section.group?.title ?? (sections.length > 1 ? $t('household.habits.otherHabits') : $t('household.habits.today')) }}
					<Icon
						v-if="section.group?.shared"
						icon="users"
						class="shared-icon"
						:title="$t('household.habits.sharedRoutine')"
					/>
				</h2>
				<ul class="habit-list">
					<HabitRow
						v-for="habit in section.habits"
						:key="habit.id"
						:habit="habit"
						@edit="editing = {habit: $event}"
					/>
				</ul>
			</section>

			<section
				v-if="bad.length > 0"
				class="habit-section card"
			>
				<h2 class="section-title">
					{{ $t('household.habits.badTitle') }}
				</h2>
				<p class="section-help">
					{{ $t('household.habits.badHelp') }}
				</p>
				<ul class="habit-list">
					<HabitRow
						v-for="habit in bad"
						:key="habit.id"
						:habit="habit"
						@edit="editing = {habit: $event}"
					/>
				</ul>
			</section>
		</template>

		<p class="archived-toggle">
			<label class="checkbox">
				<input
					v-model="showArchived"
					type="checkbox"
				>
				{{ $t('household.habits.showArchived') }}
			</label>
		</p>

		<HabitEditor
			v-if="editing"
			:habit="editing.habit"
			:groups="groups.data.value ?? []"
			@close="editing = null"
		/>
		<RoutinesEditor
			v-if="routinesOpen"
			:groups="groups.data.value ?? []"
			@close="routinesOpen = false"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useNow} from '@vueuse/core'
import {useQuery, useQueryClient} from '@tanstack/vue-query'

import {useTitle} from '@/composables/useTitle'
import {useI18n} from 'vue-i18n'

import {dayKey} from '../menu/week'
import HabitEditor from './components/HabitEditor.vue'
import HabitRow from './components/HabitRow.vue'
import RoutinesEditor from './components/RoutinesEditor.vue'
import {habitGroupsQuery, habitKeys, habitsQuery, type HabitResponse} from './queries'
import {badHabits, habitSections, habitState} from './today'

const {t} = useI18n({useScope: 'global'})
useTitle(() => t('household.habits.title'))

const showArchived = ref(false)
// computed(): the query changes (and fetches) when the box is ticked.
const habits = useQuery(computed(() => habitsQuery(showArchived.value)))
const groups = useQuery(habitGroupsQuery())

// With "show archived" on the list has both; the sections show what's asked for.
const list = computed(() => (habits.data.value ?? []).filter(h => h.archived === showArchived.value))
const sections = computed(() => habitSections(list.value, groups.data.value ?? []))
const bad = computed(() => badHabits(list.value))

// "3 of 5 done today": the habits that count today.
const summary = computed(() => {
	const counted = list.value.filter(h => ['due', 'done'].includes(habitState(h)))
	return {done: counted.filter(h => habitState(h) === 'done').length, total: counted.length}
})

// The editor: {} for a new habit, {habit} to edit one, null when closed.
const editing = ref<{habit?: HabitResponse} | null>(null)
const routinesOpen = ref(false)

// A new day with the page open: fetch again, so ticks count for the new day.
const client = useQueryClient()
const now = useNow({interval: 60 * 1000})
watch(() => dayKey(now.value), () => client.invalidateQueries({queryKey: habitKeys.all}))
</script>

<style lang="scss" scoped>
.habits-page {
	max-inline-size: 50rem;
	margin-inline: auto;
}

.habits-header {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: space-between;
	gap: .5rem;
	margin-block-end: 1rem;

	.title {
		margin: 0;
	}
}

.header-actions {
	display: flex;
	gap: .5rem;
}

.today-summary {
	margin-block-end: 1rem;
	color: var(--grey-600);
}

.habit-section {
	margin-block-end: 1rem;
	padding: 1rem;
}

.section-title {
	margin-block-end: .25rem;
	font-size: 1.1rem;
	font-weight: 700;
}

.section-help {
	color: var(--grey-600);
	font-size: .9rem;
}

.shared-icon {
	margin-inline-start: .25rem;
	color: var(--grey-500);
	font-size: .8em;
}

.habit-list {
	margin: 0;
	padding: 0;
	list-style: none;
}

.empty {
	padding: 2rem 0;

	p {
		margin-block-end: 1rem;
	}
}

.archived-toggle {
	margin-block-start: 1.5rem;
	color: var(--grey-600);
}
</style>
