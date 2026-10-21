<template>
	<div class="recipes">
		<header class="recipes-header">
			<h1 class="title">
				{{ $t('household.recipes.title') }}
			</h1>
			<RouterLink
				:to="{name: 'recipe.new'}"
				class="button is-primary"
			>
				<Icon icon="plus" />
				<span>{{ $t('household.recipes.add') }}</span>
			</RouterLink>
		</header>

		<input
			v-model="search"
			class="input search"
			type="search"
			:placeholder="$t('household.recipes.search')"
			:aria-label="$t('household.recipes.search')"
		>

		<div
			v-if="tags.length > 0"
			class="tags-filter"
		>
			<button
				v-for="tag in tags"
				:key="tag"
				type="button"
				class="tag"
				:class="{'is-primary': activeTag === tag}"
				:aria-pressed="activeTag === tag"
				@click="activeTag = activeTag === tag ? '' : tag"
			>
				{{ tag }}
			</button>
		</div>

		<p
			v-if="recipes.isPending.value"
			class="has-text-centered has-text-grey"
		>
			{{ $t('household.recipes.loading') }}
		</p>
		<p
			v-else-if="(recipes.data.value ?? []).length === 0"
			class="has-text-centered has-text-grey is-italic"
		>
			{{ $t('household.recipes.empty') }}
		</p>
		<p
			v-else-if="shown.length === 0"
			class="has-text-centered has-text-grey is-italic"
		>
			{{ $t('household.recipes.noMatch') }}
		</p>

		<ul class="recipe-grid">
			<li
				v-for="recipe in shown"
				:key="recipe.id"
			>
				<RouterLink
					:to="{name: 'recipe', params: {recipeId: recipe.id}}"
					class="recipe-card"
				>
					<RecipeImage
						class="card-image"
						:recipe-id="recipe.id"
						:image-id="recipe.image_ids[0]"
						:alt="''"
					/>
					<span class="card-body">
						<span class="card-title">{{ recipe.title }}</span>
						<span class="card-meta">
							<span v-if="recipe.total_minutes">
								<Icon icon="clock" /> {{ formatMinutes(recipe.total_minutes) }}
							</span>
							<span v-if="recipe.tags.length">{{ recipe.tags.join(' · ') }}</span>
						</span>
					</span>
				</RouterLink>
			</li>
		</ul>
	</div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'
import {useStorage} from '@vueuse/core'

import {useTitle} from '@/composables/useTitle'

import RecipeImage from './components/RecipeImage.vue'
import {recipesQuery} from './queries'
import {formatMinutes} from './scaling'

const {t} = useI18n({useScope: 'global'})
useTitle(() => t('household.recipes.title'))

const recipes = useQuery(recipesQuery())
const search = ref('')
// Remembered on this device, like the shopping tabs.
const activeTag = useStorage('mandarin.recipes.tag', '')

// Every tag in use, most used first.
const tags = computed(() => {
	const counts = new Map<string, number>()
	for (const recipe of recipes.data.value ?? []) {
		for (const tag of recipe.tags) counts.set(tag, (counts.get(tag) ?? 0) + 1)
	}
	return [...counts.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).map(([tag]) => tag)
})

// Filtered here rather than by the server, so it works as you type, and
// with no signal on recipes already loaded.
const shown = computed(() => {
	const words = search.value.toLowerCase().split(/\s+/).filter(Boolean)
	return (recipes.data.value ?? []).filter(recipe => {
		if (activeTag.value && !recipe.tags.includes(activeTag.value)) return false
		const haystack = [recipe.title, ...recipe.tags, ...recipe.ingredients.map(i => i.name)].join(' ').toLowerCase()
		return words.every(word => haystack.includes(word))
	})
})
</script>

<style lang="scss" scoped>
.recipes {
	max-inline-size: 64rem;
	margin-inline: auto;
}

.recipes-header {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: space-between;
	gap: .75rem;
	margin-block-end: 1rem;

	.title {
		margin: 0;
	}
}

.search {
	margin-block-end: .75rem;
}

.tags-filter {
	display: flex;
	flex-wrap: wrap;
	gap: .4rem;
	margin-block-end: 1rem;

	.tag {
		border: 0;
		cursor: pointer;
	}
}

.recipe-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(14rem, 1fr));
	gap: 1rem;
	margin: 0;
	padding: 0;
	list-style: none;
}

.recipe-card {
	display: flex;
	flex-direction: column;
	block-size: 100%;
	overflow: hidden;
	border-radius: $radius;
	background: var(--white);
	box-shadow: var(--shadow-sm);
	color: var(--text);

	&:hover {
		box-shadow: var(--shadow-md);
	}
}

.card-image {
	aspect-ratio: 4 / 3;
	block-size: auto;
}

// On a phone: one row per recipe, thumbnail on the left, so a few fit on screen.
@media screen and (max-width: $tablet) {
	.recipe-grid {
		grid-template-columns: 1fr;
		gap: .6rem;
	}

	.recipe-card {
		flex-direction: row;
	}

	.card-image {
		flex: none;
		inline-size: 6.5rem;
	}
}

.card-body {
	display: flex;
	flex-direction: column;
	gap: .25rem;
	padding: .75rem;
}

.card-title {
	font-weight: 600;
	line-height: 1.3;
}

.card-meta {
	display: flex;
	flex-wrap: wrap;
	gap: .25rem .75rem;
	color: var(--grey-500);
	font-size: .85rem;
}
</style>
