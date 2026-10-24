<template>
	<!-- Vikunja's Modal + Card, like PickDayModal. -->
	<Modal
		:aria-label="title"
		@close="$emit('close')"
	>
		<Card
			:title="title"
			:shadow="false"
			:show-close="true"
			class="has-text-start add-to-list-modal"
			@close="$emit('close')"
		>
			<p
				v-if="servings > 0"
				class="servings"
			>
				{{ $t('household.menu.servings', {count: servings}, servings) }}
			</p>

			<p
				v-if="recipe.isPending.value"
				class="has-text-grey"
			>
				{{ $t('household.menu.loading') }}
			</p>
			<p
				v-else-if="!recipe.data.value"
				class="has-text-grey is-italic"
			>
				{{ $t('household.recipes.notLoaded') }}
			</p>
			<!-- The recipe page's own picker: it adds through the shopping sync engine. -->
			<IngredientPicker
				v-else
				:lines="recipe.data.value.ingredients"
				:factor="factor"
				@added="$emit('close')"
			/>

			<template #footer>
				<XButton
					variant="secondary"
					:shadow="false"
					@click="$emit('close')"
				>
					{{ $t('household.menu.close') }}
				</XButton>
			</template>
		</Card>
	</Modal>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'

import IngredientPicker from '../../recipes/components/IngredientPicker.vue'
import {recipeQuery} from '../../recipes/queries'
import {entryFactor} from '../planning'

const props = defineProps<{
	recipeId: number
	recipeTitle: string
	/** The servings planned for the entry, 0 when not set. */
	servings: number
}>()

defineEmits<{
	close: []
}>()

const {t} = useI18n({useScope: 'global'})
const title = computed(() => t('household.menu.addToListTitle', {recipe: props.recipeTitle}))

// Passing a computed lets the query follow the recipe id if it changes.
const recipe = useQuery(computed(() => recipeQuery(props.recipeId)))
const factor = computed(() => entryFactor(props.servings, recipe.data.value?.servings ?? 0))
</script>

<style lang="scss" scoped>
.servings {
	margin-block-end: .75rem;
	color: var(--grey-500);
}

.add-to-list-modal :deep(.card-footer) {
	display: flex;
	justify-content: flex-end;
	gap: .5rem;
	padding: .75rem 1rem;
}
</style>
