<template>
	<Card
		:title="$t('household.home.habitsToday')"
		class="home-card"
	>
		<p
			v-if="!sync.hasData.value"
			class="has-text-grey is-italic"
		>
			{{ sync.online.value ? $t('household.home.loading') : $t('household.habits.notLoaded') }}
		</p>
		<p
			v-else-if="sync.habits.value.length === 0"
			class="has-text-grey"
		>
			{{ $t('household.home.noHabits') }}
			<RouterLink :to="{name: 'habits'}">
				{{ $t('household.habits.addFirst') }}
			</RouterLink>
		</p>
		<p
			v-else-if="today.length === 0"
			class="has-text-grey"
		>
			{{ $t('household.home.noHabitsToday') }}
		</p>
		<ul
			v-else
			class="habit-rows"
		>
			<li
				v-for="habit in today"
				:key="habit.id"
				class="habit-row"
				:class="{'is-done': habit.progress.done_today}"
			>
				<HabitTick :habit="habit" />
				<span class="habit-title">{{ habit.title }}</span>
				<span
					v-if="habit.progress.streak > 1"
					class="streak"
				>
					<Icon icon="fire" />
					{{ habit.progress.streak }}
				</span>
			</li>
		</ul>
		<template #footer>
			<RouterLink
				:to="{name: 'habits'}"
				class="card-footer-item"
			>
				{{ $t('household.home.openHabits') }}
			</RouterLink>
		</template>
	</Card>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import Card from '@/components/misc/Card.vue'

import {useHabitsSync} from '../sync/useHabitsSync'
import {homeHabits} from '../today'
import HabitTick from './HabitTick.vue'

// The Home page's "Habits today" card: what's still due, then what's done,
// each with its tick button. Rest days, paused and bad habits stay on the Habits page.
// Read from the phone's copy, so this works offline.

const sync = useHabitsSync()
const today = computed(() => homeHabits(sync.habits.value, sync.groups.value))
</script>

<style lang="scss" scoped>
// Matches the other cards on Home (household/HomeView.vue).
.home-card {
	display: flex;
	flex-direction: column;
	margin: 0;

	:deep(.card-content) {
		flex: 1;
	}
}

.habit-rows {
	margin: 0;
	padding: 0;
	list-style: none;
}

.habit-row {
	display: flex;
	align-items: center;
	gap: .75rem;
	padding: .25rem 0;

	:deep(.habit-tick) {
		--size: 2rem;
	}
}

.habit-title {
	flex: 1;
	font-weight: 600;
}

.is-done .habit-title {
	color: var(--grey-600);
	text-decoration: line-through;
}

.streak {
	color: var(--primary);
	font-size: .85rem;
	font-weight: 600;
}
</style>
