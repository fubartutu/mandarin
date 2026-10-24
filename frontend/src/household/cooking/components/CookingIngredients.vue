<template>
	<section class="cooking-ingredients">
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
					<label
						class="ingredient"
						:class="{'is-ticked': ticked.includes(entry.index)}"
					>
						<input
							v-model="ticked"
							type="checkbox"
							:value="entry.index"
						>
						<span>{{ ingredientLabel(entry.line, factor) }}</span>
					</label>
				</li>
			</ul>
		</template>
	</section>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import type {IngredientLine} from '../../recipes/queries'
import {ingredientLabel} from '../../recipes/scaling'

const props = defineProps<{
	lines: IngredientLine[]
	factor: number
}>()

// defineModel: the parent binds the ticked line indexes with v-model.
const ticked = defineModel<number[]>({required: true})

// Lines in their groups ("Sauce"), keeping each line's index for ticking.
const groups = computed(() => {
	const out: {title: string, lines: {index: number, line: IngredientLine}[]}[] = []
	props.lines.forEach((line, index) => {
		const last = out[out.length - 1]
		if (last && last.title === line.group) last.lines.push({index, line})
		else out.push({title: line.group, lines: [{index, line}]})
	})
	return out
})
</script>

<style lang="scss" scoped>
.group-title {
	margin-block: 1rem .25rem;
	color: var(--grey-600);
	font-size: 1rem;
	font-weight: 600;

	&:first-child {
		margin-block-start: 0;
	}
}

.ingredient-list {
	margin: 0;
	padding: 0;
	list-style: none;
}

// Big targets: tapped with a floury finger.
.ingredient {
	display: flex;
	align-items: baseline;
	gap: .75rem;
	padding-block: .5rem;
	font-size: 1.15rem;
	line-height: 1.4;
	cursor: pointer;

	input {
		flex: none;
		inline-size: 1.35rem;
		block-size: 1.35rem;
		accent-color: var(--primary);
	}

	&.is-ticked span {
		color: var(--grey-400);
		text-decoration: line-through;
	}
}
</style>
