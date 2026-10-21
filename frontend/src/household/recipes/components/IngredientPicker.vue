<template>
	<section class="ingredients">
		<h2 class="section-title">
			{{ $t('household.recipes.ingredients') }}
		</h2>
		<template
			v-for="group in groups"
			:key="group.title"
		>
			<h3
				v-if="group.title"
				class="group-title"
			>
				{{ group.title }}
			</h3>
			<ul class="ingredient-list">
				<li
					v-for="entry in group.lines"
					:key="entry.index"
				>
					<label class="ingredient">
						<input
							v-model="picked"
							type="checkbox"
							:value="entry.index"
						>
						<span>{{ ingredientLabel(entry.line, factor) }}</span>
					</label>
				</li>
			</ul>
		</template>

		<!-- Adding goes through the shopping sync engine, so it works offline too. -->
		<div
			v-if="lists.length > 0"
			class="add-to-list"
		>
			<div class="select">
				<select
					v-model="listId"
					:aria-label="$t('household.recipes.toList')"
				>
					<option
						v-for="list in lists"
						:key="list.id"
						:value="list.id"
					>
						{{ list.title }}
					</option>
				</select>
			</div>
			<XButton
				icon="basket-shopping"
				:disabled="picked.length === 0"
				@click="addToList"
			>
				{{ $t('household.recipes.addToList', {count: picked.length}, picked.length) }}
			</XButton>
		</div>
		<BaseButton
			class="pick-all"
			@click="picked = picked.length === lines.length ? [] : lines.map((_, i) => i)"
		>
			{{ picked.length === lines.length ? $t('household.recipes.pickNone') : $t('household.recipes.pickAll') }}
		</BaseButton>
	</section>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useStorage} from '@vueuse/core'

import BaseButton from '@/components/base/BaseButton.vue'
import {success} from '@/message'

import {planAdds, prepareNewItem} from '../../shopping/adding'
import {createOp, updateOp} from '../../shopping/sync/model'
import {useShoppingData} from '../../shopping/useShopping'
import {useShoppingSync} from '../../shopping/sync/useShoppingSync'
import type {IngredientLine} from '../queries'
import {ingredientLabel, scaledQuantity} from '../scaling'

const props = defineProps<{
	lines: IngredientLine[]
	factor: number
}>()

const {t} = useI18n({useScope: 'global'})
const sync = useShoppingSync()
const {lists, categories, catalog} = useShoppingData()

// Lines in their groups, keeping each line's index for the checkboxes.
const groups = computed(() => {
	const out: {title: string, lines: {index: number, line: IngredientLine}[]}[] = []
	props.lines.forEach((line, index) => {
		const last = out[out.length - 1]
		if (last && last.title === line.group) last.lines.push({index, line})
		else out.push({title: line.group, lines: [{index, line}]})
	})
	return out
})

// Everything is ticked to start with; untick what's already in the cupboard.
const picked = ref<number[]>([])
watch(() => props.lines, lines => {
	picked.value = lines.map((_, i) => i)
}, {immediate: true})

// The list last added to (or the shopping list last opened), else the first.
const lastShoppingList = useStorage('mandarin.shopping.lastList', 0)
const lastRecipeList = useStorage('mandarin.recipes.lastList', 0)
const listId = ref(0)
watch(lists, all => {
	if (all.some(l => l.id === listId.value)) return
	listId.value = [lastRecipeList.value, lastShoppingList.value]
		.find(id => all.some(l => l.id === id)) ?? all[0]?.id ?? 0
}, {immediate: true})

function addToList() {
	const items = [...sync.data.value.items.values()]
	const adds = [...picked.value].sort((a, b) => a - b).map(i => {
		const line = props.lines[i]
		return prepareNewItem({
			name: line.name || line.text,
			quantity: scaledQuantity(line, props.factor),
			unit: line.quantity > 0 ? line.unit : '',
		}, listId.value, catalog.value, categories.value)
	})
	const plan = planAdds(adds, items)
	for (const item of plan.creates) sync.change(createOp(item))
	for (const {item, quantity} of plan.topUps) {
		const op = updateOp(item, {quantity, unit: item.unit})
		if (op) sync.change(op)
	}
	lastRecipeList.value = listId.value

	const list = lists.value.find(l => l.id === listId.value)?.title ?? ''
	const added = plan.creates.length + plan.topUps.length
	success({
		message: t('household.recipes.added', {count: added, list}, added) +
			(plan.skipped.length ? ' ' + t('household.recipes.alreadyThere', {names: plan.skipped.join(', ')}) : ''),
	})
}
</script>

<style lang="scss" scoped>
.section-title {
	margin-block-end: .5rem;
	font-size: 1.15rem;
	font-weight: 600;
}

.group-title {
	margin-block: .75rem .25rem;
	color: var(--grey-600);
	font-size: .95rem;
	font-weight: 600;
}

.ingredient-list {
	margin: 0;
	padding: 0;
	list-style: none;
}

.ingredient {
	display: flex;
	align-items: baseline;
	gap: .6rem;
	padding-block: .3rem;
	cursor: pointer;

	input {
		flex: none;
		inline-size: 1.1rem;
		block-size: 1.1rem;
	}
}

.add-to-list {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
	margin-block-start: 1rem;
}

.pick-all {
	margin-block-start: .5rem;
	color: var(--primary);
	font-size: .85rem;
}
</style>
