<template>
	<!-- Covers the whole app (menu and top bar too) while cooking. -->
	<div class="cooking">
		<header class="cooking-header">
			<div class="cooking-title">
				<h1 class="title">
					{{ data?.title ?? $t('household.recipes.loading') }}
				</h1>
				<span
					v-if="amountLabel"
					class="cooking-amount"
				>{{ amountLabel }}</span>
			</div>
			<BaseButton
				class="close-button"
				:aria-label="$t('household.cooking.close')"
				:title="$t('household.cooking.close')"
				@click="close"
			>
				<Icon icon="xmark" />
			</BaseButton>
		</header>

		<p
			v-if="!data && !recipe.isPending.value"
			class="cooking-message"
		>
			{{ $t('household.recipes.notLoaded') }}
		</p>

		<template v-if="data">
			<!-- On a phone one panel shows at a time; tablet and laptop show both. -->
			<nav
				class="panel-tabs"
				role="tablist"
			>
				<button
					v-for="tab in tabs"
					:key="tab"
					type="button"
					role="tab"
					class="panel-tab"
					:class="{'is-active': panel === tab}"
					:aria-selected="panel === tab"
					@click="panel = tab"
				>
					{{ $t(`household.cooking.${tab}`) }}
				</button>
			</nav>

			<div class="cooking-body">
				<CookingIngredients
					v-model="ticked"
					class="panel ingredients-panel"
					:class="{'is-shown': panel === 'ingredients'}"
					:lines="data.ingredients"
					:factor="scale.factor"
				/>

				<section
					ref="stepsPanel"
					class="panel steps-panel"
					:class="{'is-shown': panel === 'steps'}"
				>
					<CookingStep
						v-if="currentStep"
						class="current-step"
						:step="currentStep"
						:count="steps.length"
					/>
					<p
						v-else
						class="cooking-message"
					>
						{{ $t('household.cooking.noSteps') }}
					</p>

					<div
						v-if="steps.length > 0"
						class="step-buttons"
					>
						<XButton
							variant="secondary"
							icon="chevron-left"
							:disabled="stepIndex === 0"
							@click="go(-1)"
						>
							{{ $t('household.cooking.back') }}
						</XButton>
						<XButton
							v-if="stepIndex < steps.length - 1"
							icon="chevron-right"
							@click="go(1)"
						>
							{{ $t('household.cooking.next') }}
						</XButton>
						<XButton
							v-else
							icon="check"
							@click="close"
						>
							{{ $t('household.cooking.done') }}
						</XButton>
					</div>
				</section>
			</div>
		</template>
	</div>
</template>

<script setup lang="ts">
import {computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useRoute, useRouter} from 'vue-router'
import {useQuery} from '@tanstack/vue-query'
import {useEventListener, useSwipe} from '@vueuse/core'

import BaseButton from '@/components/base/BaseButton.vue'
import {useTitle} from '@/composables/useTitle'

import {recipeQuery} from '../recipes/queries'
import CookingIngredients from './components/CookingIngredients.vue'
import CookingStep from './components/CookingStep.vue'
import {cookingScale} from './servings'
import {cookingSteps, moveStep} from './steps'
import {holdScreenAwake} from './wakeLock'

const props = defineProps<{
	recipeId: number
}>()

const {t} = useI18n({useScope: 'global'})
const route = useRoute()
const router = useRouter()

// Same query as the recipe page, so it opens from the cache straight away.
const recipe = useQuery(computed(() => recipeQuery(props.recipeId)))
const data = computed(() => recipe.data.value)
useTitle(() => data.value?.title ?? t('household.recipes.title'))

const scale = computed(() => cookingScale(data.value?.servings ?? 0, route.query))
const amountLabel = computed(() => {
	if (!data.value) return ''
	if (scale.value.servings) return t('household.cooking.servings', {count: scale.value.servings}, scale.value.servings)
	return scale.value.factor === 1 ? '' : t('household.cooking.scale', {scale: scale.value.factor})
})

const ticked = ref<number[]>([])

const steps = computed(() => cookingSteps(data.value?.steps ?? []))
const stepIndex = ref(0)
const currentStep = computed(() => steps.value[stepIndex.value])
function go(delta: number) {
	stepIndex.value = moveStep(stepIndex.value, delta, steps.value.length)
}
// A refetch can change the method under us; stay on a real step.
watch(() => steps.value.length, () => go(0))

type Panel = 'ingredients' | 'steps'
const tabs: Panel[] = ['ingredients', 'steps']
const panel = ref<Panel>('ingredients')
// Nothing to gather: start on the steps.
watch(() => data.value?.ingredients.length, count => {
	if (count === 0) panel.value = 'steps'
}, {immediate: true})

// Swipe left for the next step, right for the one before.
const stepsPanel = useTemplateRef<HTMLElement>('stepsPanel')
useSwipe(stepsPanel, {
	onSwipeEnd(_, direction) {
		if (direction === 'left') go(1)
		else if (direction === 'right') go(-1)
	},
})

// Laptop: arrow keys move between steps, Escape closes.
useEventListener(document, 'keydown', (e: KeyboardEvent) => {
	if (e.altKey || e.ctrlKey || e.metaKey) return
	if (e.key === 'ArrowRight') go(1)
	else if (e.key === 'ArrowLeft') go(-1)
	else if (e.key === 'Escape') close()
})

// The screen stays on while cooking mode is open.
let releaseWakeLock = () => {}
onMounted(() => {
	releaseWakeLock = holdScreenAwake()
})
onBeforeUnmount(() => releaseWakeLock())

// Back to the recipe: a history step when we came from it, so the browser's
// back button doesn't then return to cooking mode.
function close() {
	const recipePage = router.resolve({name: 'recipe', params: {recipeId: props.recipeId}}).fullPath
	if (window.history.state?.back === recipePage) router.back()
	else router.replace(recipePage)
}
</script>

<style lang="scss" scoped>
.cooking {
	position: fixed;
	inset: 0;
	z-index: 4600; // above the menu, top bar and modals (4500)
	display: flex;
	flex-direction: column;
	padding:
		max(.75rem, env(safe-area-inset-top))
		max(1rem, env(safe-area-inset-right))
		max(.75rem, env(safe-area-inset-bottom))
		max(1rem, env(safe-area-inset-left));
	background: var(--site-background);
	color: var(--text);
}

.cooking-header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 1rem;
	margin-block-end: .75rem;
}

.cooking-title {
	min-inline-size: 0;

	.title {
		margin: 0;
		font-size: 1.35rem;
		word-break: normal;
	}
}

.cooking-amount {
	color: var(--grey-600);
}

.close-button {
	display: flex;
	flex: none;
	align-items: center;
	justify-content: center;
	inline-size: 2.75rem;
	block-size: 2.75rem;
	border-radius: 50%;
	background: var(--grey-100);
	font-size: 1.25rem;
}

.cooking-message {
	color: var(--grey-600);
	font-style: italic;
}

.panel-tabs {
	display: flex;
	gap: .5rem;
	margin-block-end: .75rem;

	@media screen and (min-width: $tablet) {
		display: none;
	}
}

.panel-tab {
	flex: 1;
	padding: .6rem;
	border: 0;
	border-radius: $radius;
	background: var(--grey-100);
	color: var(--text);
	font: inherit;
	font-weight: 600;
	cursor: pointer;

	&.is-active {
		background: var(--primary);
		color: var(--white);
	}
}

// Ingredients beside the step on a tablet or laptop.
.cooking-body {
	display: grid;
	flex: 1;
	grid-template-columns: 1fr;
	gap: 2rem;
	min-block-size: 0;

	@media screen and (min-width: $tablet) {
		grid-template-columns: minmax(16rem, 1fr) 2fr;
	}
}

.panel {
	min-block-size: 0;
	overflow-y: auto;

	@media screen and (max-width: $tablet - 1px) {
		&:not(.is-shown) {
			display: none;
		}
	}
}

.steps-panel {
	display: flex;
	flex-direction: column;
	// Vertical scrolling stays with the browser; sideways swipes are ours.
	touch-action: pan-y;
}

.current-step {
	flex: 1;
}

.step-buttons {
	display: flex;
	justify-content: space-between;
	gap: 1rem;
	padding-block-start: 1rem;

	.button {
		flex: 1;
		max-inline-size: 14rem;
		block-size: 3.25rem;
		font-size: 1.1rem;
	}
}
</style>
