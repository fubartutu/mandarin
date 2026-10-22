<template>
	<!-- Its own small app (see offlineTasks.ts), so it shows on every page,
		including when the phone's menu is closed. -->
	<p
		v-if="show"
		class="offline-banner"
		role="status"
	>
		{{ i18n.global.t('household.offline.tasks') }}
	</p>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import type {Router} from 'vue-router'
import {useOnline} from '@vueuse/core'

import {i18n} from '@/i18n'

const props = defineProps<{
	router: Router
}>()

const online = useOnline()
// The shopping list has its own sync badge; signed-out pages have Vikunja's offline page.
const QUIET = ['/shopping', '/login', '/register', '/password-reset']
const show = computed(() => !online.value &&
	!QUIET.some(path => props.router.currentRoute.value.path.startsWith(path)))
</script>

<style lang="scss" scoped>
.offline-banner {
	position: fixed;
	inset-block-end: max(1rem, env(safe-area-inset-bottom));
	inset-inline: 0;
	z-index: 50;
	inline-size: fit-content;
	max-inline-size: calc(100vw - 2rem);
	margin: 0 auto;
	padding: .5rem 1rem;
	border-radius: 999px;
	background: var(--grey-800);
	color: var(--grey-50);
	box-shadow: var(--shadow-md);
	font-size: .85rem;
	text-align: center;
	pointer-events: none;
}
</style>
