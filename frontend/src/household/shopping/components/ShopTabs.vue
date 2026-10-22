<template>
	<div class="shop-tabs">
		<div
			class="tabs-row"
			role="tablist"
			:aria-label="$t('household.shopping.shops.label')"
		>
			<button
				type="button"
				role="tab"
				class="shop-tab"
				:class="{'is-active': modelValue === 0}"
				:aria-selected="modelValue === 0"
				@click="emit('update:modelValue', 0)"
			>
				{{ $t('household.shopping.shops.all') }}
			</button>
			<button
				v-for="shop in shops"
				:key="shop.id"
				type="button"
				role="tab"
				class="shop-tab"
				:class="{'is-active': modelValue === shop.id}"
				:aria-selected="modelValue === shop.id"
				@click="emit('update:modelValue', shop.id)"
			>
				{{ shop.title }}
			</button>
			<BaseButton
				class="manage"
				:disabled="!online"
				:title="online ? undefined : $t('household.shopping.sync.needsSignal')"
				:aria-label="$t('household.shopping.shops.manage')"
				:aria-expanded="managing"
				@click="managing = !managing"
			>
				<Icon icon="store" />
				<span v-if="shops.length === 0">{{ $t('household.shopping.shops.addFirst') }}</span>
			</BaseButton>
		</div>
		<ShopManager
			v-if="managing"
			:list-id="listId"
			:shops="shops"
			:categories="categories"
			@close="managing = false"
		/>
	</div>
</template>

<script setup lang="ts">
import {ref} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import {useShoppingOnline} from '../sync/useShoppingSync'
import ShopManager from './ShopManager.vue'
import type {ShoppingCategoryResponse, ShoppingShopResponse} from '../queries'

// v-model on this component: modelValue is the selected shop id (0 = all),
// and the update:modelValue event sends a new one back to the parent.
defineProps<{
	listId: number
	shops: ShoppingShopResponse[]
	categories: ShoppingCategoryResponse[]
	modelValue: number
}>()

const emit = defineEmits<{
	'update:modelValue': [shopId: number]
}>()

const managing = ref(false)

// Adding lists and shops needs the server.
const online = useShoppingOnline()
</script>

<style lang="scss" scoped>
.shop-tabs {
	margin-block-end: 1rem;
}

.tabs-row {
	display: flex;
	align-items: center;
	gap: .25rem;
	border-block-end: 2px solid var(--grey-200);
	overflow-x: auto;
}

.shop-tab {
	flex: none;
	margin-block-end: -2px;
	padding: .5rem .9rem;
	border: 0;
	border-block-end: 2px solid transparent;
	background: transparent;
	color: var(--grey-500);
	font: inherit;
	white-space: nowrap;
	cursor: pointer;

	&.is-active {
		border-block-end-color: var(--primary);
		color: var(--text);
		font-weight: 600;
	}
}

.manage {
	display: flex;
	align-items: center;
	gap: .4rem;
	margin-inline-start: auto;
	padding: .5rem;
	color: var(--grey-500);
	white-space: nowrap;
}
</style>
