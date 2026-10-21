<template>
	<div class="paperless-picker">
		<input
			v-model="search"
			class="input"
			type="search"
			:placeholder="$t('household.recipes.paperless.search')"
			:aria-label="$t('household.recipes.paperless.search')"
		>
		<p
			v-if="docs.isPending.value"
			class="help"
		>
			{{ $t('household.recipes.loading') }}
		</p>
		<p
			v-else-if="docs.isError.value"
			class="help is-danger"
		>
			{{ errorText }}
		</p>
		<p
			v-else-if="(docs.data.value?.documents ?? []).length === 0"
			class="help"
		>
			{{ $t('household.recipes.paperless.none', {tag: 'recipe'}) }}
		</p>
		<ul class="scans">
			<li
				v-for="doc in docs.data.value?.documents ?? []"
				:key="doc.id"
			>
				<button
					type="button"
					class="scan"
					:disabled="busy"
					@click="emit('pick', doc.id)"
				>
					<PaperlessThumb
						class="scan-thumb"
						:doc-id="doc.id"
					/>
					<span class="scan-title">{{ doc.title }}</span>
					<span class="scan-date">{{ doc.created.slice(0, 10) }}</span>
				</button>
			</li>
		</ul>
	</div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {refDebounced} from '@vueuse/core'

import {getErrorText} from '@/message'

import PaperlessThumb from './PaperlessThumb.vue'
import {paperlessQuery} from '../queries'

defineProps<{
	busy: boolean // an import is running
}>()

const emit = defineEmits<{
	pick: [docId: number]
}>()

// Paperless searches the scans' OCR text; wait for a pause in typing.
const search = ref('')
const debounced = refDebounced(search, 400)
const docs = useQuery(computed(() => paperlessQuery(debounced.value.trim())))
const errorText = computed(() => getErrorText(docs.error.value))
</script>

<style lang="scss" scoped>
.scans {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(8rem, 1fr));
	gap: .75rem;
	margin: .75rem 0 0;
	padding: 0;
	list-style: none;
}

.scan {
	display: flex;
	flex-direction: column;
	inline-size: 100%;
	padding: 0;
	overflow: hidden;
	border: 0;
	border-radius: $radius;
	background: var(--white);
	box-shadow: var(--shadow-xs);
	color: var(--text);
	text-align: start;
	cursor: pointer;

	&:disabled {
		opacity: .5;
		cursor: wait;
	}
}

.scan-thumb {
	aspect-ratio: 3 / 4;
	block-size: auto;
}

.scan-title {
	padding: .4rem .5rem 0;
	font-size: .85rem;
	font-weight: 600;
	line-height: 1.3;
}

.scan-date {
	padding: 0 .5rem .4rem;
	color: var(--grey-500);
	font-size: .75rem;
}
</style>
