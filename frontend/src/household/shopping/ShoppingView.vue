<template>
	<div
		class="shopping"
		:class="{'is-loading': listsPending}"
	>
		<ListSwitcher
			:lists="lists"
			:active-id="activeListId"
			@create="createList"
		/>

		<p
			v-if="!listsPending && lists.length === 0"
			class="has-text-centered has-text-grey is-italic"
		>
			{{ $t('household.shopping.noLists') }}
		</p>

		<template v-if="activeList">
			<header class="list-header">
				<h1 class="title">
					{{ activeList.title }}
				</h1>
				<div class="list-actions">
					<BaseButton
						class="list-action"
						:aria-label="$t('household.shopping.renameList')"
						@click="renameList"
					>
						<Icon icon="pen" />
					</BaseButton>
					<BaseButton
						class="list-action"
						:aria-label="$t('household.shopping.deleteList')"
						@click="deleteList"
					>
						<Icon icon="trash-alt" />
					</BaseButton>
				</div>
			</header>

			<ShopTabs
				v-model="activeShopId"
				:list-id="activeList.id"
				:shops="activeList.shops"
			/>

			<AddItemBar
				:categories="categories"
				:catalog="catalog"
				:on-list="openNames"
				@add="addItem"
			/>

			<div :class="{'is-loading': itemsPending}">
				<p
					v-if="!itemsPending && grouped.groups.length === 0 && grouped.done.length === 0"
					class="has-text-centered has-text-grey is-italic"
				>
					{{ $t('household.shopping.empty') }}
				</p>
				<p
					v-if="hiddenCount > 0"
					class="hidden-note"
				>
					{{ $t('household.shopping.shops.hidden', {count: hiddenCount, shop: activeShop?.title}, hiddenCount) }}
				</p>

				<CategoryGroup
					v-for="group in grouped.groups"
					:key="group.categoryId"
					:storage-key="String(group.categoryId)"
					:title="group.title"
					:count="group.items.length"
				>
					<template
						v-for="item in group.items"
						:key="item.id"
					>
						<ItemEditor
							v-if="editingId === item.id"
							:item="item"
							:categories="categories"
							:shops="activeList.shops"
							@save="saveItem"
							@delete="deleteItem(item)"
							@cancel="editingId = null"
						/>
						<ItemRow
							v-else
							:item="item"
							:hint="notStockedHint(item)"
							@toggle="toggleItem(item)"
							@edit="editingId = item.id"
						/>
					</template>
				</CategoryGroup>

				<CategoryGroup
					v-if="grouped.done.length > 0"
					class="done-group"
					storage-key="done"
					:title="$t('household.shopping.ticked')"
					:count="grouped.done.length"
				>
					<template #actions>
						<BaseButton
							class="clear-done"
							@click="clearDone.mutate(activeListId)"
						>
							{{ $t('household.shopping.clearTicked') }}
						</BaseButton>
					</template>
					<ItemRow
						v-for="item in grouped.done"
						:key="item.id"
						:item="item"
						@toggle="toggleItem(item)"
						@edit="toggleItem(item)"
					/>
				</CategoryGroup>
			</div>
		</template>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {useStorage} from '@vueuse/core'

import BaseButton from '@/components/base/BaseButton.vue'
import {useTitle} from '@/composables/useTitle'

import AddItemBar from './components/AddItemBar.vue'
import CategoryGroup from './components/CategoryGroup.vue'
import ItemEditor from './components/ItemEditor.vue'
import ItemRow from './components/ItemRow.vue'
import ListSwitcher from './components/ListSwitcher.vue'
import ShopTabs from './components/ShopTabs.vue'
import {
	useClearDoneItemsMutation,
	useCreateShoppingItemMutation,
	useCreateShoppingListMutation,
	useDeleteShoppingItemMutation,
	useDeleteShoppingListMutation,
	useRenameShoppingListMutation,
	useSetNotStockedMutation,
	useUpdateShoppingItemMutation,
	type ShoppingItemResponse,
} from './queries'
import {findCatalogEntry} from './autocomplete'
import type {ParsedItem} from './quantity'
import {
	groupItems,
	itemsForShop,
	useShoppingCatalog,
	useShoppingCategories,
	useShoppingItems,
	useShoppingLists,
} from './useShopping'

const props = defineProps<{
	listId: number // 0 when the URL is just /shopping
}>()

const router = useRouter()
const {t} = useI18n({useScope: 'global'})

const {lists, isPending: listsPending} = useShoppingLists()
const {categories} = useShoppingCategories()
const {catalog} = useShoppingCatalog()

// ---- Which list is open ----

const lastListId = useStorage('mandarin.shopping.lastList', 0)
const activeList = computed(() => lists.value.find(l => l.id === props.listId))
const activeListId = computed(() => activeList.value?.id ?? 0)

// /shopping (or a deleted list's URL) → the last-used list, else the first.
// `watch` re-runs whenever one of the watched values changes.
watch([() => props.listId, lists], () => {
	if (listsPending.value || lists.value.length === 0 || activeList.value) {
		return
	}
	const fallback = lists.value.find(l => l.id === lastListId.value) ?? lists.value[0]
	router.replace({name: 'shopping', params: {listId: fallback.id}})
}, {immediate: true})

watch(activeListId, id => {
	if (id > 0) lastListId.value = id
})

useTitle(() => activeList.value?.title ?? t('household.shopping.title'))

// ---- Which shop tab is open (remembered per list on this device) ----

const shopTabs = useStorage<Record<number, number>>('mandarin.shopping.shopTab', {})
const activeShop = computed(() => activeList.value?.shops.find(s => s.id === shopTabs.value[activeListId.value]))
// A computed with get and set can be used with v-model.
const activeShopId = computed({
	get: () => activeShop.value?.id ?? 0, // a deleted shop falls back to All
	set: id => {
		shopTabs.value = {...shopTabs.value, [activeListId.value]: id}
	},
})

// ---- Items ----

const {items, isPending: itemsPending} = useShoppingItems(activeListId)
const shopItems = computed(() => itemsForShop(items.value, activeShopId.value))
const hiddenCount = computed(() => items.value.filter(i => !i.done).length - shopItems.value.filter(i => !i.done).length)
const grouped = computed(() => groupItems(shopItems.value, categories.value, t('household.shopping.noCategory')))
const openNames = computed(() => items.value.filter(i => !i.done).map(i => i.name))
const editingId = ref<number | null>(null)

// On the All tab, say which shops don't stock an item.
function notStockedHint(item: ShoppingItemResponse): string | undefined {
	if (activeShopId.value !== 0 || item.not_stocked_shop_ids.length === 0) return undefined
	const titles = (activeList.value?.shops ?? [])
		.filter(shop => item.not_stocked_shop_ids.includes(shop.id))
		.map(shop => shop.title)
	return titles.length ? t('household.shopping.shops.notAt', {shops: titles.join(', ')}) : undefined
}

const createItem = useCreateShoppingItemMutation()
const updateItem = useUpdateShoppingItemMutation()
const removeItem = useDeleteShoppingItemMutation()
const clearDone = useClearDoneItemsMutation()
const setNotStocked = useSetNotStockedMutation()

function addItem(item: ParsedItem & {category_id: number}) {
	// Fill blanks from the catalogue the way the server will, so the item
	// appears in the right group straight away instead of jumping there.
	const known = findCatalogEntry(catalog.value, item.name)
	const categoryId = categories.value.some(c => c.id === item.category_id)
		? item.category_id
		: known?.category_id ?? 0
	const keepLast = known && item.quantity === 0 && item.unit === ''
	createItem.mutate({
		name: known?.name ?? item.name,
		quantity: keepLast ? known.quantity : item.quantity,
		unit: keepLast ? known.unit : item.unit,
		list_id: activeListId.value,
		category_id: categoryId,
	})
}

function toggleItem(item: ShoppingItemResponse) {
	updateItem.mutate({...item, done: !item.done})
}

async function saveItem(item: ShoppingItemResponse, notStockedShopIds: number[]) {
	editingId.value = null
	const changed = [...notStockedShopIds].sort().join() !== [...item.not_stocked_shop_ids].sort().join()
	try {
		// One after the other: both write, and SQLite allows one writer at a time.
		const saved = await updateItem.mutateAsync(item)
		if (changed && saved.catalog_item_id) {
			setNotStocked.mutate({catalogId: saved.catalog_item_id, shopIds: notStockedShopIds})
		}
	} catch {
		return // already toasted by the mutation
	}
}

function deleteItem(item: ShoppingItemResponse) {
	editingId.value = null
	removeItem.mutate(item)
}

// ---- Lists ----

const createListMutation = useCreateShoppingListMutation()
const renameListMutation = useRenameShoppingListMutation()
const deleteListMutation = useDeleteShoppingListMutation()

async function createList(title: string) {
	try {
		const list = await createListMutation.mutateAsync(title)
		router.push({name: 'shopping', params: {listId: list.id}})
	} catch {
		return // already toasted by the mutation
	}
}

function renameList() {
	const list = activeList.value
	const title = list && window.prompt(t('household.shopping.renameList'), list.title)?.trim()
	if (list && title) {
		renameListMutation.mutate({...list, title})
	}
}

function deleteList() {
	const list = activeList.value
	if (list && window.confirm(t('household.shopping.deleteListConfirm', {title: list.title}))) {
		deleteListMutation.mutate(list.id)
	}
}
</script>

<style lang="scss" scoped>
.shopping {
	max-inline-size: 42rem;
	margin-inline: auto;
}

.list-header {
	display: flex;
	align-items: center;
	gap: .5rem;
	margin-block-end: 1rem;

	.title {
		margin: 0;
	}
}

.list-actions {
	display: flex;
	gap: .25rem;
	margin-inline-start: auto;
}

.list-action {
	padding: .5rem;
	color: var(--grey-500);
}

.hidden-note {
	margin-block-end: .75rem;
	color: var(--grey-500);
	font-size: .85rem;
}

.done-group {
	opacity: .85;
}

.clear-done {
	padding: .6rem .75rem;
	color: var(--primary);
	font-size: .85rem;
	font-weight: normal;
	white-space: nowrap;
}
</style>
