<template>
	<form
		class="add-item-bar"
		@submit.prevent="submit"
	>
		<input
			ref="input"
			v-model="text"
			class="input"
			type="text"
			enterkeyhint="done"
			autocomplete="off"
			:placeholder="$t('household.shopping.addPlaceholder')"
			:aria-label="$t('household.shopping.addPlaceholder')"
		>
		<div class="select">
			<select
				v-model="categoryId"
				:aria-label="$t('household.shopping.category')"
			>
				<option
					v-for="category in categories"
					:key="category.id"
					:value="category.id"
				>
					{{ category.title }}
				</option>
				<option :value="0">
					{{ $t('household.shopping.noCategory') }}
				</option>
			</select>
		</div>
		<XButton
			type="submit"
			icon="plus"
			:disabled="text.trim() === ''"
		>
			{{ $t('household.shopping.add') }}
		</XButton>
	</form>
</template>

<script setup lang="ts">
import {ref, useTemplateRef} from 'vue'
import {useStorage} from '@vueuse/core'

import {parseItemInput, type ParsedItem} from '../quantity'
import type {ShoppingCategoryResponse} from '../queries'

defineProps<{
	categories: ShoppingCategoryResponse[]
}>()

const emit = defineEmits<{
	add: [item: ParsedItem & {category_id: number}]
}>()

const text = ref('')
// Remembered in the browser, so a run of dairy items keeps "Dairy" selected.
const categoryId = useStorage('mandarin.shopping.lastCategory', 0)
const input = useTemplateRef<HTMLInputElement>('input')

function submit() {
	const parsed = parseItemInput(text.value)
	if (parsed.name === '') {
		return
	}
	emit('add', {...parsed, category_id: categoryId.value})
	text.value = ''
	// Keep the keyboard up for the next item.
	input.value?.focus()
}
</script>

<style lang="scss" scoped>
.add-item-bar {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
	margin-block-end: 1rem;

	.input {
		flex: 1 1 14rem;
		font-size: 1rem; // 16px stops iOS zooming into the field
	}

	.select {
		flex: 0 1 auto;
	}
}
</style>
