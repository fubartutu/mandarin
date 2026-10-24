<template>
	<!-- One day's entries on a line: recipes link to their page and to cooking mode, notes are plain text. -->
	<span>
		<!-- A <template> v-for repeats several nodes without a wrapper element. -->
		<template
			v-for="(entry, i) in entries"
			:key="entry.id"
		>
			<template v-if="i > 0">{{ SEPARATOR }}</template>
			<template v-if="entry.recipeId">
				<RouterLink :to="{name: 'recipe', params: {recipeId: entry.recipeId}}">
					{{ entryLabel(entry) }}
				</RouterLink>
				<RouterLink
					:to="cookRoute(entry)"
					class="cook-link"
					:aria-label="$t('household.home.cookRecipe', {recipe: entryLabel(entry)})"
					:title="$t('household.home.cookRecipe', {recipe: entryLabel(entry)})"
				>
					<Icon icon="utensils" />
				</RouterLink>
			</template>
			<template v-else>{{ entry.note }}</template>
		</template>
	</span>
</template>

<script setup lang="ts">
import {useI18n} from 'vue-i18n'

import {cookRoute} from '../planning'
import type {MenuEntryResponse} from '../queries'

defineProps<{
	entries: MenuEntryResponse[]
}>()

const SEPARATOR = ' · '

const {t} = useI18n({useScope: 'global'})

function entryLabel(entry: MenuEntryResponse) {
	return entry.recipeTitle || t('household.menu.missingRecipe')
}
</script>

<style lang="scss" scoped>
.cook-link {
	margin-inline-start: .35rem;
}
</style>
