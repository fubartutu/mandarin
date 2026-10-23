<template>
	<form
		class="add-item-bar"
		@submit.prevent="submit"
	>
		<div class="input-wrap">
			<input
				ref="input"
				v-model="text"
				class="input"
				type="text"
				enterkeyhint="done"
				autocomplete="off"
				role="combobox"
				aria-autocomplete="list"
				:aria-expanded="showSuggestions"
				aria-controls="shopping-suggestions"
				:aria-activedescendant="highlighted >= 0 ? `shopping-suggestion-${highlighted}` : undefined"
				:placeholder="$t('household.shopping.addPlaceholder')"
				:aria-label="$t('household.shopping.addPlaceholder')"
				@focus="focused = true"
				@blur="focused = false"
				@keydown.down.prevent="move(1)"
				@keydown.up.prevent="move(-1)"
				@keydown.esc="highlighted = -1; focused = false"
				@input="highlighted = -1"
			>
			<ul
				v-show="showSuggestions"
				id="shopping-suggestions"
				class="suggestions"
				role="listbox"
			>
				<!-- mousedown.prevent keeps focus in the input, so the list
				     doesn't close before the click lands. -->
				<li
					v-for="(entry, index) in suggestions"
					:id="`shopping-suggestion-${index}`"
					:key="entry.id"
					role="option"
					class="suggestion"
					:class="{'is-highlighted': index === highlighted}"
					:aria-selected="index === highlighted"
					@mousedown.prevent
					@click="pick(entry)"
				>
					<span class="name">{{ entry.name }}</span>
					<span class="meta">{{ describe(entry) }}</span>
				</li>
			</ul>
		</div>
		<DictateButton v-model="text" />
		<div class="select">
			<select
				v-model="categoryId"
				:aria-label="$t('household.shopping.category')"
			>
				<option :value="0">
					{{ $t('household.shopping.autoCategory') }}
				</option>
				<option
					v-for="category in categories"
					:key="category.id"
					:value="category.id"
				>
					{{ category.title }}
				</option>
			</select>
		</div>
		<div
			v-if="shops.length > 0"
			class="select"
		>
			<select
				v-model="shopId"
				:aria-label="$t('household.shopping.shops.buyAt')"
			>
				<option :value="0">
					{{ $t('household.shopping.shops.any') }}
				</option>
				<option
					v-for="shop in shops"
					:key="shop.id"
					:value="shop.id"
				>
					{{ $t('household.shopping.shops.only', {shop: shop.title}) }}
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
import {computed, ref, useTemplateRef} from 'vue'

import DictateButton from '../../dictation/DictateButton.vue'
import {suggestCatalogEntries} from '../autocomplete'
import {formatQuantity, parseItemInput, type ParsedItem} from '../quantity'
import type {CatalogEntry, ShoppingCategoryResponse, ShoppingShopResponse} from '../queries'

const props = defineProps<{
	categories: ShoppingCategoryResponse[]
	shops: ShoppingShopResponse[]
	catalog: CatalogEntry[]
	// Names already open on the list; not suggested again.
	onList: string[]
}>()

const emit = defineEmits<{
	add: [item: ParsedItem & {category_id: number, shop_id: number}]
}>()

const text = ref('')
const categoryId = ref(0) // 0 = let the catalogue decide
const shopId = ref(0) // 0 = any shop
const focused = ref(false)
const highlighted = ref(-1)
const input = useTemplateRef<HTMLInputElement>('input')

const parsed = computed(() => parseItemInput(text.value))
const suggestions = computed(() => suggestCatalogEntries(props.catalog, parsed.value.name, props.onList))
const showSuggestions = computed(() => focused.value && suggestions.value.length > 0)

const categoryTitles = computed(() => new Map(props.categories.map(c => [c.id, c.title])))

function describe(entry: CatalogEntry): string {
	return [categoryTitles.value.get(entry.category_id), formatQuantity(entry.quantity, entry.unit)]
		.filter(Boolean)
		.join(' · ')
}

function move(step: number) {
	const count = suggestions.value.length
	if (count === 0) return
	// Wraps around: -1 means "nothing highlighted, use what's typed".
	highlighted.value = (highlighted.value + step + count + 1) % (count + 1) - 1
}

function add(name: string) {
	const {quantity, unit} = parsed.value
	emit('add', {name, quantity, unit, category_id: categoryId.value, shop_id: shopId.value})
	text.value = ''
	// Both go back to automatic, so the next item isn't quietly tied to a shop.
	categoryId.value = 0
	shopId.value = 0
	highlighted.value = -1
	// Keep the keyboard up for the next item.
	input.value?.focus()
}

function pick(entry: CatalogEntry) {
	add(entry.name)
}

function submit() {
	if (highlighted.value >= 0 && suggestions.value[highlighted.value]) {
		pick(suggestions.value[highlighted.value])
		return
	}
	if (parsed.value.name !== '') {
		add(parsed.value.name)
	}
}
</script>

<style lang="scss" scoped>
.add-item-bar {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
	margin-block-end: 1rem;
}

.input-wrap {
	position: relative;
	flex: 1 1 14rem;

	.input {
		font-size: 1rem; // 16px stops iOS zooming into the field
	}
}

.suggestions {
	position: absolute;
	inset-inline: 0;
	inset-block-start: 100%;
	z-index: 10;
	margin: .25rem 0 0;
	padding: .25rem 0;
	list-style: none;
	border-radius: $radius;
	background: var(--white);
	box-shadow: var(--shadow-md);
}

.suggestion {
	display: flex;
	align-items: baseline;
	gap: .75rem;
	padding: .65rem .75rem; // tall enough to tap
	cursor: pointer;

	&.is-highlighted,
	&:hover {
		background: var(--grey-100);
	}

	.meta {
		margin-inline-start: auto;
		color: var(--grey-500);
		font-size: .85rem;
		white-space: nowrap;
	}
}
</style>
