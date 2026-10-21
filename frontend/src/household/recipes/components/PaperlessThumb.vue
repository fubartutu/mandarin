<template>
	<img
		v-if="url"
		class="paperless-thumb"
		:src="url"
		alt=""
	>
	<span
		v-else
		class="paperless-thumb is-placeholder"
		aria-hidden="true"
	>
		<Icon icon="file-lines" />
	</span>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {useObjectUrl} from '@vueuse/core'

import {paperlessThumbQuery} from '../queries'

const props = defineProps<{
	docId: number
}>()

// Fetched through Mandarin (Paperless needs its token), shown as a blob: URL.
const thumb = useQuery(computed(() => paperlessThumbQuery(props.docId)))
const url = useObjectUrl(computed(() => thumb.data.value))
</script>

<style lang="scss" scoped>
.paperless-thumb {
	display: block;
	inline-size: 100%;
	block-size: 100%;
	object-fit: cover;
	object-position: top;
	background: var(--grey-100);
}

.is-placeholder {
	display: flex;
	align-items: center;
	justify-content: center;
	color: var(--grey-300);
}
</style>
