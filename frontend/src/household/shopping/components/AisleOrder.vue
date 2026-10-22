<template>
	<div class="aisle-order">
		<p class="help">
			{{ $t('household.shopping.shops.aisleHelp', {title: shop.title}) }}
		</p>
		<ol class="categories">
			<li
				v-for="(category, index) in order"
				:key="category.id"
				class="category"
			>
				<span class="position">{{ index + 1 }}</span>
				<span class="name">{{ category.title }}</span>
				<BaseButton
					class="move"
					:disabled="index === 0"
					:aria-label="$t('household.shopping.shops.moveUp', {title: category.title})"
					@click="move(index, -1)"
				>
					<Icon icon="chevron-up" />
				</BaseButton>
				<BaseButton
					class="move"
					:disabled="index === order.length - 1"
					:aria-label="$t('household.shopping.shops.moveDown', {title: category.title})"
					@click="move(index, 1)"
				>
					<Icon icon="chevron-down" />
				</BaseButton>
			</li>
		</ol>
		<div class="actions">
			<XButton
				variant="tertiary"
				class="reset"
				@click="emit('save', [])"
			>
				{{ $t('household.shopping.shops.resetOrder') }}
			</XButton>
			<XButton
				variant="tertiary"
				@click="emit('cancel')"
			>
				{{ $t('household.shopping.shops.cancel') }}
			</XButton>
			<XButton @click="emit('save', order.map(c => c.id))">
				{{ $t('household.shopping.shops.saveOrder') }}
			</XButton>
		</div>
	</div>
</template>

<script setup lang="ts">
import {ref} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import {inAisleOrder} from '../useShopping'
import type {ShoppingCategoryResponse, ShoppingShopResponse} from '../queries'

// A shop's aisle order: every category, moved up and down with buttons
// (big enough to tap, unlike a drag handle). Saved as a whole.
const props = defineProps<{
	shop: ShoppingShopResponse
	categories: ShoppingCategoryResponse[]
}>()

const emit = defineEmits<{
	save: [categoryIds: number[]] // [] = back to the household order
	cancel: []
}>()

// A draft, seeded once: the parent mounts this fresh each time it opens.
// eslint-disable-next-line vue/no-setup-props-reactivity-loss -- a draft of the order
const order = ref(inAisleOrder(props.categories, props.shop.category_order))

function move(index: number, step: number) {
	const next = [...order.value]
	const [category] = next.splice(index, 1)
	next.splice(index + step, 0, category)
	order.value = next
}
</script>

<style lang="scss" scoped>
.aisle-order {
	margin-block: 0 .75rem;
	padding: .5rem .75rem;
	border-radius: $radius;
	background: var(--white);
}

.help {
	margin-block-end: .5rem;
	color: var(--grey-500);
	font-size: .85rem;
}

.categories {
	margin: 0;
	padding: 0;
	list-style: none;
}

.category {
	display: flex;
	align-items: center;
	gap: .5rem;
	border-block-end: 1px solid var(--grey-200);

	.position {
		min-inline-size: 1.5rem;
		color: var(--grey-500);
		font-size: .85rem;
		text-align: end;
	}

	.name {
		flex: 1;
	}
}

.move {
	padding: .6rem .75rem; // tall enough to tap
	color: var(--grey-500);

	&:disabled {
		opacity: .3;
	}
}

.actions {
	display: flex;
	flex-wrap: wrap;
	justify-content: flex-end;
	gap: .5rem;
	margin-block-start: .75rem;

	.reset {
		margin-inline-end: auto;
	}
}
</style>
