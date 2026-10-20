<template>
	<li
		class="item-row"
		:class="{'is-done': item.done, 'is-pending': item.id < 0}"
	>
		<!-- The big tick target: the whole left edge is one button. -->
		<button
			type="button"
			class="tick"
			:aria-pressed="item.done"
			:aria-label="$t(item.done ? 'household.shopping.untick' : 'household.shopping.tick', {name: item.name})"
			:disabled="item.id < 0"
			@click="emit('toggle')"
		>
			<Icon
				:icon="item.done ? 'circle-check' : ['far', 'circle']"
				class="tick-icon"
			/>
		</button>

		<button
			type="button"
			class="details"
			:disabled="item.id < 0"
			@click="emit('edit')"
		>
			<span class="name">{{ item.name }}</span>
			<span
				v-if="quantity"
				class="quantity"
			>{{ quantity }}</span>
			<span
				v-if="item.note || hint"
				class="note"
			>{{ [item.note, hint].filter(Boolean).join(' · ') }}</span>
		</button>

		<ItemPhoto
			v-if="item.has_photo && item.catalog_item_id"
			class="thumb"
			:catalog-id="item.catalog_item_id"
			:name="item.name"
		/>
	</li>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import ItemPhoto from './ItemPhoto.vue'
import {formatQuantity} from '../quantity'
import type {ShoppingItemResponse} from '../queries'

// defineProps / defineEmits are compiler macros: they declare this
// component's inputs and the events it can send to its parent.
const props = defineProps<{
	item: ShoppingItemResponse
	// Extra muted text, e.g. "Not at Aldi" on the All tab.
	hint?: string
}>()

const emit = defineEmits<{
	toggle: []
	edit: []
}>()

const quantity = computed(() => formatQuantity(props.item.quantity, props.item.unit))
</script>

<style lang="scss" scoped>
.item-row {
	display: flex;
	align-items: stretch;
	border-block-end: 1px solid var(--grey-200);
	background: var(--white);

	&.is-pending {
		opacity: .6;
	}
}

.tick {
	flex: 0 0 3.25rem;
	min-block-size: 3rem;
	display: flex;
	align-items: center;
	justify-content: center;
	border: 0;
	background: transparent;
	color: var(--grey-400);
	cursor: pointer;

	.tick-icon {
		font-size: 1.5rem;
	}

	.is-done & {
		color: var(--success);
	}
}

.details {
	flex: 1;
	display: flex;
	flex-wrap: wrap;
	align-items: baseline;
	gap: 0 .5rem;
	padding: .75rem .75rem .75rem 0;
	border: 0;
	background: transparent;
	color: var(--text);
	font: inherit;
	text-align: start;
	cursor: pointer;
}

.name {
	font-size: 1.1rem;

	.is-done & {
		color: var(--grey-400);
		text-decoration: line-through;
	}
}

.thumb {
	align-self: center;
	margin-inline-end: .75rem;
}

.quantity {
	color: var(--grey-500);
	font-variant-numeric: tabular-nums;
}

.note {
	flex-basis: 100%;
	color: var(--grey-500);
	font-size: .85rem;
}
</style>
