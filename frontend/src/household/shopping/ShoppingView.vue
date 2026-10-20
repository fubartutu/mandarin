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

			<AddItemBar
				:categories="categories"
				@add="addItem"
			/>

			<div :class="{'is-loading': itemsPending}">
				<p
					v-if="!itemsPending && grouped.groups.length === 0 && grouped.done.length === 0"
					class="has-text-centered has-text-grey is-italic"
				>
					{{ $t('household.shopping.empty') }}
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
							@save="saveItem"
							@delete="deleteItem(item)"
							@cancel="editingId = null"
						/>
						<ItemRow
							v-else
							:item="item"
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
import {
	useClearDoneItemsMutation,
	useCreateShoppingItemMutation,
	useCreateShoppingListMutation,
	useDeleteShoppingItemMutation,
	useDeleteShoppingListMutation,
	useRenameShoppingListMutation,
	useUpdateShoppingItemMutation,
	type ShoppingItemResponse,
} from './queries'
import type {ParsedItem} from './quantity'
import {groupItems, useShoppingCategories, useShoppingItems, useShoppingLists} from './useShopping'

const props = defineProps<{
	listId: number // 0 when the URL is just /shopping
}>()

const router = useRouter()
const {t} = useI18n({useScope: 'global'})

const {lists, isPending: listsPending} = useShoppingLists()
const {categories} = useShoppingCategories()

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

// ---- Items ----

const {items, isPending: itemsPending} = useShoppingItems(activeListId)
const grouped = computed(() => groupItems(items.value, categories.value, t('household.shopping.noCategory')))
const editingId = ref<number | null>(null)

const createItem = useCreateShoppingItemMutation()
const updateItem = useUpdateShoppingItemMutation()
const removeItem = useDeleteShoppingItemMutation()
const clearDone = useClearDoneItemsMutation()

function addItem(item: ParsedItem & {category_id: number}) {
	const categoryExists = categories.value.some(c => c.id === item.category_id)
	createItem.mutate({
		...item,
		list_id: activeListId.value,
		category_id: categoryExists ? item.category_id : 0,
	})
}

function toggleItem(item: ShoppingItemResponse) {
	updateItem.mutate({...item, done: !item.done})
}

function saveItem(item: ShoppingItemResponse) {
	editingId.value = null
	updateItem.mutate(item)
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
