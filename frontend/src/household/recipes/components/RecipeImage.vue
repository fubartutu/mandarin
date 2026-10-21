<template>
	<img
		v-if="url"
		class="recipe-image"
		:src="url"
		:alt="alt"
	>
	<span
		v-else
		class="recipe-image is-placeholder"
		aria-hidden="true"
	>
		<Icon icon="book-open" />
	</span>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {useObjectUrl} from '@vueuse/core'

import {recipeImageQuery} from '../queries'

const props = defineProps<{
	recipeId: number
	imageId: number | undefined // undefined: the recipe has no picture
	alt: string
}>()

// The picture needs the login token, so it's fetched as a blob and shown
// through a blob: URL, which useObjectUrl frees when the component goes.
const image = useQuery(computed(() => ({
	...recipeImageQuery(props.recipeId, props.imageId ?? 0),
	enabled: Boolean(props.imageId),
})))
const url = useObjectUrl(computed(() => image.data.value))
</script>

<style lang="scss" scoped>
.recipe-image {
	display: block;
	inline-size: 100%;
	block-size: 100%;
	object-fit: cover;
	background: var(--grey-100);
}

.is-placeholder {
	display: flex;
	align-items: center;
	justify-content: center;
	color: var(--grey-300);
	font-size: 2rem;
}
</style>
