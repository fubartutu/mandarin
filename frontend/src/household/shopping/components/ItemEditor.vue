<template>
	<li class="item-editor">
		<form @submit.prevent="emit('save', draft, notStocked)">
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
						v-model="quantity"
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
				<fieldset
					v-if="shops.length > 0"
					class="field not-stocked"
				>
					<legend class="label">
						{{ $t('household.shopping.edit.notStockedAt') }}
					</legend>
					<label
						v-for="shop in shops"
						:key="shop.id"
						class="checkbox"
					>
						<input
							v-model="notStocked"
							type="checkbox"
							:value="shop.id"
						>
						{{ shop.title }}
					</label>
				</fieldset>
				<div
					v-if="item.catalog_item_id"
					class="field photo"
				>
					<span class="label">{{ $t('household.shopping.photo.label') }}</span>
					<div class="photo-row">
						<ItemPhoto
							v-if="item.has_photo"
							:catalog-id="item.catalog_item_id"
							:name="item.name"
							size="large"
						/>
						<!-- capture="environment" opens the back camera on a phone. -->
						<label class="button is-small">
							<Icon icon="camera" />
							<span>{{ $t(item.has_photo ? 'household.shopping.photo.replace' : 'household.shopping.photo.add') }}</span>
							<input
								class="is-sr-only"
								type="file"
								accept="image/*"
								capture="environment"
								@change="choosePhoto"
							>
						</label>
						<XButton
							v-if="item.has_photo"
							variant="tertiary"
							:loading="deletePhoto.isPending.value"
							@click="deletePhoto.mutate(item.catalog_item_id)"
						>
							{{ $t('household.shopping.photo.remove') }}
						</XButton>
						<span
							v-if="uploadPhoto.isPending.value"
							class="uploading"
						>{{ $t('household.shopping.photo.uploading') }}</span>
					</div>
				</div>
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
import {computed, reactive, ref} from 'vue'

import ItemPhoto from './ItemPhoto.vue'
import {
	useDeletePhotoMutation,
	useUploadPhotoMutation,
	type ShoppingCategoryResponse,
	type ShoppingItemResponse,
	type ShoppingShopResponse,
} from '../queries'

const props = defineProps<{
	item: ShoppingItemResponse
	categories: ShoppingCategoryResponse[]
	shops: ShoppingShopResponse[]
}>()

const emit = defineEmits<{
	save: [item: ShoppingItemResponse, notStockedShopIds: number[]]
	delete: []
	cancel: []
}>()

// Photos save straight away: they belong to the catalogue entry, not this edit.
const uploadPhoto = useUploadPhotoMutation()
const deletePhoto = useDeletePhotoMutation()

function choosePhoto(event: Event) {
	const input = event.target as HTMLInputElement
	const file = input.files?.[0]
	if (file && props.item.catalog_item_id) {
		uploadPhoto.mutate({catalogId: props.item.catalog_item_id, file})
	}
	input.value = '' // so picking the same file again still fires
}

// A draft copy: edits only reach the list when Save is pressed. The parent
// mounts this component fresh (keyed by item id) each time editing starts.
// eslint-disable-next-line vue/no-setup-props-reactivity-loss -- seeded once so a refetch keeps the draft; mounted fresh per edit
const draft = reactive<ShoppingItemResponse>({...props.item, quantity: props.item.quantity || 0})
// 0 means "no quantity", so the field shows empty rather than 0.
const quantity = computed({
	get: () => draft.quantity || '',
	set: (value: string | number) => {
		draft.quantity = Math.max(0, Number(value) || 0)
	},
})
// eslint-disable-next-line vue/no-setup-props-reactivity-loss -- same: a draft of the marks
const notStocked = ref<number[]>([...props.item.not_stocked_shop_ids])
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

	.name, .note, .category, .not-stocked, .photo {
		grid-column: 1 / -1;
	}

	.not-stocked {
		display: flex;
		flex-wrap: wrap;
		gap: .25rem 1rem;
		margin: 0;
		padding: 0;
		border: 0;

		legend {
			inline-size: 100%;
		}
	}

	.photo-row {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: .5rem;
	}

	.uploading {
		color: var(--grey-500);
		font-size: .85rem;
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
