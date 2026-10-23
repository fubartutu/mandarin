<template>
	<div class="recipe-page">
		<p
			v-if="recipe.isPending.value"
			class="has-text-centered has-text-grey"
		>
			{{ $t('household.recipes.loading') }}
		</p>
		<p
			v-else-if="!data"
			class="has-text-centered has-text-grey is-italic"
		>
			{{ $t('household.recipes.notLoaded') }}
		</p>

		<article
			v-if="data"
			class="recipe"
		>
			<div
				v-if="data.image_ids.length > 0"
				class="pictures"
			>
				<RecipeImage
					class="main-picture"
					:recipe-id="data.id"
					:image-id="shownImage"
					:alt="data.title"
				/>
				<div
					v-if="data.image_ids.length > 1"
					class="thumbs"
				>
					<button
						v-for="id in data.image_ids"
						:key="id"
						type="button"
						class="thumb"
						:class="{'is-active': id === shownImage}"
						:aria-label="$t('household.recipes.showPicture')"
						@click="shownImageId = id"
					>
						<RecipeImage
							:recipe-id="data.id"
							:image-id="id"
							alt=""
						/>
					</button>
				</div>
			</div>

			<header class="recipe-header">
				<h1 class="title">
					{{ data.title }}
				</h1>
				<div class="recipe-actions">
					<XButton
						v-if="wakeLock.isSupported.value"
						variant="secondary"
						icon="lightbulb"
						:class="{'is-active': wakeLock.isActive.value}"
						:aria-pressed="wakeLock.isActive.value"
						@click="toggleWakeLock"
					>
						{{ $t('household.recipes.keepAwake') }}
					</XButton>
					<RouterLink
						:to="{name: 'recipe.edit', params: {recipeId: data.id}}"
						class="button is-secondary"
					>
						<Icon icon="pen" />
						<span>{{ $t('household.recipes.edit') }}</span>
					</RouterLink>
				</div>
			</header>

			<p class="meta">
				<span v-if="data.prep_minutes">{{ $t('household.recipes.prep') }} {{ formatMinutes(data.prep_minutes) }}</span>
				<span v-if="data.cook_minutes">{{ $t('household.recipes.cook') }} {{ formatMinutes(data.cook_minutes) }}</span>
				<span v-if="data.total_minutes">{{ $t('household.recipes.total') }} {{ formatMinutes(data.total_minutes) }}</span>
				<a
					v-if="data.source_url"
					:href="data.source_url"
					target="_blank"
					rel="noopener noreferrer"
				>
					<Icon icon="link" /> {{ sourceHost }}
				</a>
			</p>
			<p
				v-if="data.tags.length"
				class="tags"
			>
				<span
					v-for="tag in data.tags"
					:key="tag"
					class="tag"
				>{{ tag }}</span>
			</p>
			<p
				v-if="data.description"
				class="description"
			>
				{{ data.description }}
			</p>

			<!-- Scaling: by servings when the recipe says how many, else by a multiple. -->
			<div class="scale">
				<span class="scale-label">{{ data.servings ? $t('household.recipes.servings') : $t('household.recipes.scale') }}</span>
				<BaseButton
					class="scale-button"
					:aria-label="$t('household.recipes.less')"
					:disabled="amount <= step"
					@click="amount -= step"
				>
					<Icon icon="minus" />
				</BaseButton>
				<span class="scale-value">{{ data.servings ? amount : `×${amount}` }}</span>
				<BaseButton
					class="scale-button"
					:aria-label="$t('household.recipes.more')"
					@click="amount += step"
				>
					<Icon icon="plus" />
				</BaseButton>
				<span
					v-if="data.yields && factor === 1"
					class="yields"
				>{{ data.yields }}</span>
			</div>

			<div class="columns-wrap">
				<IngredientPicker
					class="ingredients-column"
					:lines="data.ingredients"
					:factor="factor"
				/>

				<section class="method">
					<h2 class="section-title">
						{{ $t('household.recipes.method') }}
					</h2>
					<!-- Tap a step to grey it out, to keep your place while cooking. -->
					<ol class="steps">
						<template
							v-for="(entry, i) in method"
							:key="i"
						>
							<li
								v-if="entry.number === 0"
								class="step-heading"
							>
								{{ entry.text }}
							</li>
							<li
								v-else
								class="step"
								:class="{'is-done': doneSteps.has(i)}"
								@click="toggleStep(i)"
							>
								<span class="step-number">{{ entry.number }}</span>
								<span>{{ entry.text }}</span>
							</li>
						</template>
					</ol>
					<template v-if="data.notes">
						<h2 class="section-title">
							{{ $t('household.recipes.notes') }}
						</h2>
						<p class="notes">
							{{ data.notes }}
						</p>
					</template>
				</section>
			</div>
		</article>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'
import {useWakeLock} from '@vueuse/core'

import BaseButton from '@/components/base/BaseButton.vue'
import {useTitle} from '@/composables/useTitle'

import IngredientPicker from './components/IngredientPicker.vue'
import RecipeImage from './components/RecipeImage.vue'
import {recipeQuery} from './queries'
import {rememberRecipeViewed} from './recent'
import {formatMinutes, numberSteps} from './scaling'

const props = defineProps<{
	recipeId: number
}>()

const {t} = useI18n({useScope: 'global'})
const recipe = useQuery(computed(() => recipeQuery(props.recipeId)))
// The cached copy stays here if a refetch fails without signal.
const data = computed(() => recipe.data.value)
useTitle(() => data.value?.title ?? t('household.recipes.title'))

// Listed under "Recently viewed" on the Home page once it has loaded.
watch(() => data.value?.id, id => {
	if (id) rememberRecipeViewed(id)
}, {immediate: true})

const shownImageId = ref<number>()
const shownImage = computed(() =>
	data.value?.image_ids.includes(shownImageId.value ?? 0) ? shownImageId.value : data.value?.image_ids[0])

const sourceHost = computed(() => {
	try {
		return new URL(data.value?.source_url ?? '').hostname.replace(/^www\./, '')
	} catch {
		return data.value?.source_url
	}
})

// amount is servings (when known) or a multiple of the recipe.
const step = computed(() => data.value?.servings ? 1 : 0.5)
const amount = ref(1)
watch(() => data.value?.servings, servings => {
	amount.value = servings || 1
}, {immediate: true})
const factor = computed(() => data.value?.servings ? amount.value / data.value.servings : amount.value)

const method = computed(() => numberSteps(data.value?.steps ?? []))
const doneSteps = ref(new Set<number>())
function toggleStep(i: number) {
	const next = new Set(doneSteps.value)
	if (!next.delete(i)) next.add(i)
	doneSteps.value = next
}

// Keeps the screen from dimming while cooking; released when leaving the page.
const wakeLock = useWakeLock()
function toggleWakeLock() {
	if (wakeLock.isActive.value) wakeLock.release()
	else wakeLock.request('screen')
}
</script>

<style lang="scss" scoped>
.recipe-page {
	max-inline-size: 64rem;
	margin-inline: auto;
}

.pictures {
	margin-block-end: 1rem;
}

.main-picture {
	max-block-size: 24rem;
	border-radius: $radius;
}

.thumbs {
	display: flex;
	gap: .5rem;
	margin-block-start: .5rem;
}

.thumb {
	inline-size: 4rem;
	block-size: 3rem;
	padding: 0;
	overflow: hidden;
	border: 2px solid transparent;
	border-radius: $radius;
	cursor: pointer;

	&.is-active {
		border-color: var(--primary);
	}
}

.recipe-header {
	display: flex;
	flex-wrap: wrap;
	align-items: flex-start;
	justify-content: space-between;
	gap: .75rem;

	.title {
		margin: 0;
		word-break: normal;
	}
}

.recipe-actions {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;

	.is-active {
		background: var(--primary);
		color: var(--white);
	}
}

.meta {
	display: flex;
	flex-wrap: wrap;
	gap: .25rem 1rem;
	margin-block: .75rem .5rem;
	color: var(--grey-600);
	font-size: .9rem;
}

.tags {
	display: flex;
	flex-wrap: wrap;
	gap: .35rem;
	margin-block-end: .75rem;
}

.description {
	margin-block-end: 1rem;
	color: var(--grey-700);
}

.scale {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .5rem;
	margin-block: 1rem;
}

.scale-label {
	font-weight: 600;
}

.scale-button {
	display: flex;
	align-items: center;
	justify-content: center;
	inline-size: 2rem;
	block-size: 2rem;
	border-radius: 50%;
	background: var(--grey-100);

	&:disabled {
		opacity: .4;
	}
}

.scale-value {
	min-inline-size: 2.5rem;
	text-align: center;
	font-size: 1.1rem;
	font-weight: 600;
}

.yields {
	color: var(--grey-500);
	font-size: .9rem;
}

// Ingredients beside the method on a laptop or tablet, above it on a phone.
.columns-wrap {
	display: grid;
	grid-template-columns: 1fr;
	gap: 1.5rem;

	@media screen and (min-width: $tablet) {
		grid-template-columns: minmax(16rem, 2fr) 3fr;
	}
}

.section-title {
	margin-block-end: .5rem;
	font-size: 1.15rem;
	font-weight: 600;
}

.steps {
	margin: 0 0 1.5rem;
	padding: 0;
	list-style: none;
}

.step {
	display: flex;
	gap: .75rem;
	margin-block-end: .9rem;
	line-height: 1.55;
	cursor: pointer;

	&.is-done {
		color: var(--grey-400);
	}
}

.step-number {
	flex: none;
	min-inline-size: 1.5rem;
	color: var(--primary);
	font-weight: 600;
	text-align: end;
}

.step-heading {
	margin-block: 1.25rem .5rem;
	color: var(--grey-600);
	font-weight: 600;

	&:first-child {
		margin-block-start: 0;
	}
}

.notes {
	white-space: pre-line;
}
</style>
