<template>
	<!-- One root element, so a class set by the parent (e.g. "thumb") lands here. -->
	<span
		v-if="url"
		class="item-photo-wrap"
	>
		<button
			type="button"
			class="item-photo"
			:class="`is-${size}`"
			:aria-label="$t('household.shopping.photo.view', {name})"
			@click="open = true"
		>
			<img
				:src="url"
				:alt="name"
			>
		</button>
		<!-- Teleport renders the lightbox at the end of <body>, outside the list. -->
		<Teleport to="body">
			<ImageLightbox
				v-if="open"
				:blob-url="url"
				:alt="name"
				@close="open = false"
			/>
		</Teleport>
	</span>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {useObjectUrl} from '@vueuse/core'

import ImageLightbox from '@/components/misc/ImageLightbox.vue'
import {shoppingPhotoQuery} from '../queries'
import {useShoppingSync} from '../sync/useShoppingSync'

const props = withDefaults(defineProps<{
	catalogId: number
	name: string
	size?: 'small' | 'large'
}>(), {size: 'small'})

// The photo needs the login token, so it's fetched as a blob and shown via a
// blob: URL, which useObjectUrl frees again when the component goes away.
// photo_id changes when the photo is replaced, so a new one is fetched then.
const sync = useShoppingSync()
const photoId = computed(() => sync.data.value.catalog.get(props.catalogId)?.photo_id ?? 0)
const photo = useQuery(computed(() => ({
	...shoppingPhotoQuery(props.catalogId, photoId.value, sync.storage),
	enabled: photoId.value > 0,
})))
const url = useObjectUrl(computed(() => photo.data.value))
const open = ref(false)
</script>

<style lang="scss" scoped>
.item-photo-wrap {
	display: flex;
	flex: none;
}

.item-photo {
	flex: none;
	padding: 0;
	border: 0;
	border-radius: $radius;
	overflow: hidden;
	background: var(--grey-100);
	cursor: zoom-in;

	img {
		display: block;
		inline-size: 100%;
		block-size: 100%;
		object-fit: cover;
	}

	&.is-small {
		inline-size: 2.5rem;
		block-size: 2.5rem;
	}

	&.is-large {
		inline-size: 6rem;
		block-size: 6rem;
	}
}
</style>
