<template>
	<section
		class="usual-shop"
		:aria-label="$t('household.shopping.usual.title')"
	>
		<header class="usual-header">
			<h2 class="usual-title">
				<Icon icon="basket-shopping" />
				{{ $t('household.shopping.usual.title') }}
			</h2>
			<BaseButton
				class="close"
				:aria-label="$t('household.shopping.usual.close')"
				@click="emit('close')"
			>
				<Icon icon="times" />
			</BaseButton>
		</header>

		<p
			v-if="entries.length === 0"
			class="hint"
		>
			{{ $t('household.shopping.usual.empty') }}
		</p>
		<template v-else>
			<p class="hint">
				{{ $t('household.shopping.usual.intro') }}
			</p>
			<ul class="usual-list">
				<li
					v-for="{entry, onList} in entries"
					:key="entry.id"
					class="usual-row"
					:class="{'is-on-list': onList}"
				>
					<label class="checkbox">
						<input
							v-model="picked"
							type="checkbox"
							:value="entry.name"
							:disabled="onList"
						>
						<span class="name">{{ entry.name }}</span>
						<span
							v-if="onList"
							class="on-list"
						>{{ $t('household.shopping.usual.onList') }}</span>
						<span
							v-else-if="entry.quantity"
							class="amount"
						>{{ formatAmount(entry) }}</span>
					</label>
					<BaseButton
						class="remove"
						:aria-label="$t('household.shopping.usual.remove', {name: entry.name})"
						:title="$t('household.shopping.usual.remove', {name: entry.name})"
						@click="emit('setUsual', entry.name, false)"
					>
						<Icon icon="times" />
					</BaseButton>
				</li>
			</ul>
		</template>

		<form
			class="add-usual"
			@submit.prevent="addToUsual"
		>
			<input
				v-model="newName"
				class="input"
				type="text"
				maxlength="250"
				list="usual-shop-names"
				:placeholder="$t('household.shopping.usual.addPlaceholder')"
			>
			<!-- A datalist gives the browser's own suggestions from the catalogue. -->
			<datalist id="usual-shop-names">
				<option
					v-for="name in suggestions"
					:key="name"
					:value="name"
				/>
			</datalist>
			<XButton
				type="submit"
				variant="secondary"
				:disabled="newName.trim() === ''"
			>
				{{ $t('household.shopping.usual.addToUsual') }}
			</XButton>
		</form>

		<div
			v-if="entries.length > 0"
			class="actions"
		>
			<XButton
				icon="plus"
				:disabled="toAdd.length === 0"
				@click="emit('add', toAdd)"
			>
				{{ $t('household.shopping.usual.addToList', {count: toAdd.length}, toAdd.length) }}
			</XButton>
		</div>
	</section>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'

import type {UsualEntry} from '../adding'
import type {CatalogEntry} from '../queries'

const props = defineProps<{
	entries: UsualEntry[]
	catalog: CatalogEntry[]
}>()

const emit = defineEmits<{
	/** The names to put on the list. */
	add: [names: string[]]
	/** Put a name in the usual shop, or take it out. */
	setUsual: [name: string, usual: boolean]
	close: []
}>()

// Picked by name, so a stand-in entry swapped for the real one (new id)
// stays picked. Everything not already on the list starts picked; entries
// that show up later (added here, or by someone else) are picked too.
const picked = ref<string[]>([])
const seen = new Set<string>()
watch(() => props.entries, entries => {
	for (const {entry, onList} of entries) {
		if (!seen.has(entry.name) && !onList) picked.value.push(entry.name)
		seen.add(entry.name)
	}
}, {immediate: true})

const toAdd = computed(() => props.entries
	.filter(u => !u.onList && picked.value.includes(u.entry.name))
	.map(u => u.entry.name))

// The catalogue's names that aren't in the usual shop yet, most used first
// (the catalogue already comes in that order from the server).
const suggestions = computed(() => props.catalog.filter(e => !e.usual && e.id > 0).map(e => e.name))

const newName = ref('')
function addToUsual() {
	const name = newName.value.trim()
	if (name === '') return
	emit('setUsual', name, true)
	newName.value = ''
}

function formatAmount(entry: CatalogEntry): string {
	return entry.unit ? `${entry.quantity} ${entry.unit}` : `× ${entry.quantity}`
}
</script>

<style lang="scss" scoped>
.usual-shop {
	margin-block-end: 1rem;
	padding: .75rem;
	border: 1px solid var(--grey-200);
	border-radius: $radius;
	background: var(--grey-100);
}

.usual-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
}

.usual-title {
	display: flex;
	align-items: center;
	gap: .5rem;
	margin: 0;
	font-size: 1.1rem;
	font-weight: 600;
}

.close, .remove {
	padding: .5rem;
	color: var(--grey-500);
}

.hint {
	margin-block: .25rem .5rem;
	color: var(--grey-500);
	font-size: .85rem;
}

.usual-list {
	margin: 0 0 .75rem;
	padding: 0;
	list-style: none;
}

.usual-row {
	display: flex;
	align-items: center;
	border-block-end: 1px solid var(--grey-200);

	.checkbox {
		display: flex;
		flex: 1;
		align-items: center;
		gap: .6rem;
		min-block-size: 2.75rem; // a thumb-sized target
		min-inline-size: 0;
	}

	input {
		inline-size: 1.15rem;
		block-size: 1.15rem;
	}

	.name {
		overflow-wrap: anywhere;
	}

	.amount, .on-list {
		margin-inline-start: auto;
		color: var(--grey-500);
		font-size: .85rem;
		white-space: nowrap;
	}

	&.is-on-list .name {
		color: var(--grey-500);
	}
}

.add-usual {
	display: flex;
	gap: .5rem;

	.input {
		flex: 1;
		min-inline-size: 0;
		font-size: 1rem;
	}
}

.actions {
	display: flex;
	justify-content: flex-end;
	margin-block-start: .75rem;
}
</style>
