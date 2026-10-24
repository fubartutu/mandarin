<template>
	<Card
		:title="$t('household.home.menuThisWeek')"
		class="home-card"
	>
		<p
			v-if="entries.isPending.value"
			class="has-text-grey is-italic"
		>
			{{ $t('household.home.loading') }}
		</p>
		<p
			v-else-if="entries.isError.value"
			class="has-text-grey is-italic"
		>
			{{ $t('household.menu.notLoaded') }}
		</p>
		<p
			v-else-if="week.empty"
			class="has-text-grey"
		>
			{{ $t('household.home.nothingThisWeek') }}
			<RouterLink :to="{name: 'menu'}">
				{{ $t('household.home.planWeek') }}
			</RouterLink>
		</p>
		<ul
			v-else
			class="menu-rows"
		>
			<li class="menu-row is-tonight">
				<span class="menu-day">{{ $t('household.home.tonight') }}</span>
				<span
					v-if="week.tonight.length > 0"
					class="menu-entries"
				>
					<!-- A <template> v-for repeats several nodes without a wrapper element. -->
					<template
						v-for="(entry, i) in week.tonight"
						:key="entry.id"
					>
						<template v-if="i > 0">{{ SEPARATOR }}</template>
						<RouterLink
							v-if="entry.recipeId"
							:to="{name: 'recipe', params: {recipeId: entry.recipeId}}"
						>
							{{ entryLabel(entry) }}
						</RouterLink>
						<RouterLink
							v-if="entry.recipeId"
							:to="cookRoute(entry)"
							class="cook-link"
							:aria-label="$t('household.home.cookRecipe', {recipe: entryLabel(entry)})"
							:title="$t('household.home.cookRecipe', {recipe: entryLabel(entry)})"
						>
							<Icon icon="utensils" />
						</RouterLink>
						<template v-else>{{ entry.note }}</template>
					</template>
				</span>
				<span
					v-else
					class="menu-entries has-text-grey"
				>
					{{ $t('household.home.nothingTonight') }}
					<RouterLink :to="{name: 'menu'}">
						{{ $t('household.home.plan') }}
					</RouterLink>
				</span>
			</li>
			<li
				v-for="day in week.rest"
				:key="day.day"
				class="menu-row"
			>
				<span class="menu-day">{{ shortDay(day.day) }}</span>
				<span class="menu-entries">
					<template
						v-for="(entry, i) in day.entries"
						:key="entry.id"
					>
						<template v-if="i > 0">{{ SEPARATOR }}</template>
						<RouterLink
							v-if="entry.recipeId"
							:to="{name: 'recipe', params: {recipeId: entry.recipeId}}"
						>
							{{ entryLabel(entry) }}
						</RouterLink>
						<template v-else>{{ entry.note }}</template>
					</template>
				</span>
			</li>
		</ul>
		<template #footer>
			<RouterLink
				:to="{name: 'menu'}"
				class="card-footer-item"
			>
				{{ $t('household.home.openMenu') }}
			</RouterLink>
		</template>
	</Card>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {useNow} from '@vueuse/core'
import {useQuery} from '@tanstack/vue-query'

import Card from '@/components/misc/Card.vue'

import {homeWeek} from '../homeWeek'
import {cookRoute} from '../planning'
import {menuRangeQuery, type MenuEntryResponse} from '../queries'
import {dayKey, fromKey, mondayOf, weekDays} from '../week'

// The Home page's "This week" card: tonight's dinner and the rest of the week.

const SEPARATOR = ' · '

const {t, locale} = useI18n({useScope: 'global'})

// Re-checked every minute, so "tonight" moves on at midnight with the page open.
const now = useNow({interval: 60 * 1000})
const days = computed(() => weekDays(mondayOf(now.value)))

// The same query (and so the same cache entry) as this week on the Menu page.
const entries = useQuery(computed(() => menuRangeQuery(days.value[0], days.value[6])))
const week = computed(() => homeWeek(entries.data.value ?? [], dayKey(now.value)))

function entryLabel(entry: MenuEntryResponse) {
	return entry.recipeTitle || t('household.menu.missingRecipe')
}

function shortDay(day: string) {
	return fromKey(day).toLocaleDateString(locale.value, {weekday: 'short'})
}
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

.menu-rows {
	margin: 0;
	padding: 0;
	list-style: none;
}

.menu-row {
	display: flex;
	gap: .75rem;
	padding: .3rem 0;
}

.menu-row + .menu-row {
	border-block-start: 1px solid var(--grey-200);
}

.menu-day {
	flex: none;
	min-inline-size: 2.5rem;
	color: var(--grey-600);
}

.is-tonight {
	padding-block-end: .5rem;

	.menu-day {
		color: var(--text);
		font-weight: 600;
	}

	.menu-entries {
		font-weight: 600;
	}
}

.cook-link {
	margin-inline-start: .35rem;
}
</style>
