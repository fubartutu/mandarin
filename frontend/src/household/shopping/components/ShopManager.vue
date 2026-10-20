<template>
	<div class="shop-manager">
		<p class="help">
			{{ $t('household.shopping.shops.help') }}
		</p>
		<ul class="shops">
			<li
				v-for="shop in shops"
				:key="shop.id"
				class="shop"
			>
				<input
					class="input is-small"
					type="text"
					maxlength="250"
					:value="shop.title"
					:aria-label="$t('household.shopping.shops.rename', {title: shop.title})"
					@change="rename(shop, ($event.target as HTMLInputElement).value)"
				>
				<BaseButton
					class="remove"
					:aria-label="$t('household.shopping.shops.delete', {title: shop.title})"
					@click="remove(shop)"
				>
					<Icon icon="trash-alt" />
				</BaseButton>
			</li>
		</ul>
		<form
			class="add-shop"
			@submit.prevent="add"
		>
			<input
				v-model="newTitle"
				class="input is-small"
				type="text"
				maxlength="250"
				:placeholder="$t('household.shopping.shops.newPlaceholder')"
			>
			<XButton
				type="submit"
				variant="secondary"
				icon="plus"
				:disabled="newTitle.trim() === ''"
			>
				{{ $t('household.shopping.shops.add') }}
			</XButton>
			<XButton
				variant="tertiary"
				@click="emit('close')"
			>
				{{ $t('household.shopping.shops.done') }}
			</XButton>
		</form>
	</div>
</template>

<script setup lang="ts">
import {ref} from 'vue'
import {useI18n} from 'vue-i18n'

import BaseButton from '@/components/base/BaseButton.vue'
import {
	useCreateShopMutation,
	useDeleteShopMutation,
	useRenameShopMutation,
	type ShoppingShopResponse,
} from '../queries'

const props = defineProps<{
	listId: number
	shops: ShoppingShopResponse[]
}>()

const emit = defineEmits<{
	close: []
}>()

const {t} = useI18n({useScope: 'global'})
const newTitle = ref('')
const createShop = useCreateShopMutation()
const renameShop = useRenameShopMutation()
const deleteShop = useDeleteShopMutation()

function add() {
	const title = newTitle.value.trim()
	if (title !== '') {
		createShop.mutate({listId: props.listId, title})
		newTitle.value = ''
	}
}

function rename(shop: ShoppingShopResponse, value: string) {
	const title = value.trim()
	if (title !== '' && title !== shop.title) {
		renameShop.mutate({...shop, title})
	}
}

function remove(shop: ShoppingShopResponse) {
	if (window.confirm(t('household.shopping.shops.deleteConfirm', {title: shop.title}))) {
		deleteShop.mutate(shop)
	}
}
</script>

<style lang="scss" scoped>
.shop-manager {
	margin-block-start: .75rem;
	padding: .75rem;
	border-radius: $radius;
	background: var(--grey-100);
}

.help {
	margin-block-end: .5rem;
	color: var(--grey-500);
	font-size: .85rem;
}

.shops {
	margin: 0 0 .5rem;
	padding: 0;
	list-style: none;
}

.shop,
.add-shop {
	display: flex;
	align-items: center;
	gap: .5rem;
	margin-block-end: .5rem;

	.input {
		flex: 1;
		font-size: 1rem;
	}
}

.add-shop {
	flex-wrap: wrap;
}

.remove {
	padding: .5rem;
	color: var(--grey-500);
}
</style>
