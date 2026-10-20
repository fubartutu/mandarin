<template>
	<li class="item-editor">
		<form @submit.prevent="emit('save', draft)">
			<h3 class="is-sr-only">
				{{ $t('household.shopping.edit.title', {name: item.name}) }}
			</h3>
			<div class="fields">
				<label class="field name">
					<span class="label">{{ $t('household.shopping.edit.name') }}</span>
					<input
						v-model="draft.name"
						v-focus
						class="input"
						type="text"
						required
						maxlength="250"
					>
				</label>
				<label class="field quantity">
					<span class="label">{{ $t('household.shopping.edit.quantity') }}</span>
					<input
						v-model.number="draft.quantity"
						class="input"
						type="number"
						inputmode="decimal"
						min="0"
						step="any"
					>
				</label>
				<label class="field unit">
					<span class="label">{{ $t('household.shopping.edit.unit') }}</span>
					<input
						v-model="draft.unit"
						class="input"
						type="text"
						maxlength="50"
						:placeholder="$t('household.shopping.edit.unitPlaceholder')"
					>
				</label>
				<label class="field category">
					<span class="label">{{ $t('household.shopping.category') }}</span>
					<div class="select is-fullwidth">
						<select v-model="draft.category_id">
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
				</label>
				<label class="field note">
					<span class="label">{{ $t('household.shopping.edit.note') }}</span>
					<input
						v-model="draft.note"
						class="input"
						type="text"
						maxlength="2000"
						:placeholder="$t('household.shopping.edit.notePlaceholder')"
					>
				</label>
			</div>
			<div class="actions">
				<XButton
					variant="secondary"
					class="is-danger"
					icon="trash-alt"
					@click="emit('delete')"
				>
					{{ $t('household.shopping.edit.delete') }}
				</XButton>
				<XButton
					variant="tertiary"
					@click="emit('cancel')"
				>
					{{ $t('household.shopping.edit.cancel') }}
				</XButton>
				<XButton type="submit">
					{{ $t('household.shopping.edit.save') }}
				</XButton>
			</div>
		</form>
	</li>
</template>

<script setup lang="ts">
import {reactive} from 'vue'

import type {ShoppingCategoryResponse, ShoppingItemResponse} from '../queries'

const props = defineProps<{
	item: ShoppingItemResponse
	categories: ShoppingCategoryResponse[]
}>()

const emit = defineEmits<{
	save: [item: ShoppingItemResponse]
	delete: []
	cancel: []
}>()

// A draft copy: edits only reach the list when Save is pressed. The parent
// mounts this component fresh (keyed by item id) each time editing starts.
// eslint-disable-next-line vue/no-setup-props-reactivity-loss -- seeded once so a refetch keeps the draft; mounted fresh per edit
const draft = reactive<ShoppingItemResponse>({...props.item, quantity: props.item.quantity || 0})
</script>

<style lang="scss" scoped>
.item-editor {
	padding: .75rem;
	border-block-end: 1px solid var(--grey-200);
	background: var(--grey-100);
}

.fields {
	display: grid;
	grid-template-columns: 1fr 1fr;
	gap: .5rem .75rem;

	.name, .note, .category {
		grid-column: 1 / -1;
	}

	.label {
		display: block;
		margin-block-end: .25rem;
		font-size: .85rem;
	}

	.input {
		font-size: 1rem;
	}
}

.actions {
	display: flex;
	flex-wrap: wrap;
	justify-content: flex-end;
	gap: .5rem;
	margin-block-start: .75rem;

	.is-danger {
		margin-inline-end: auto;
	}
}
</style>
