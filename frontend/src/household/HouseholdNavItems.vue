<template>
	<li>
		<RouterLink :to="{name: 'shopping'}">
			<span class="menu-item-icon icon">
				<Icon icon="basket-shopping" />
			</span>
			{{ $t('household.shopping.title') }}
			<!-- Changes made offline that haven't reached the server yet. -->
			<span
				v-if="sync.pendingCount.value > 0"
				class="pending"
				:title="$t('household.shopping.sync.pendingHelp', {count: sync.pendingCount.value}, sync.pendingCount.value)"
			>{{ sync.pendingCount.value }}</span>
		</RouterLink>
	</li>
	<li>
		<RouterLink :to="{name: 'recipes'}">
			<span class="menu-item-icon icon">
				<Icon icon="book-open" />
			</span>
			{{ $t('household.recipes.title') }}
		</RouterLink>
	</li>
</template>

<script setup lang="ts">
import {watch} from 'vue'

import {useAuthStore} from '@/stores/auth'
import {keepOfflineApiCacheFor} from './offlineTasks'
import {useShoppingSync} from './shopping/sync/useShoppingSync'

// Starting the engine here (the menu is on every page) means changes made
// offline are sent even if the shopping list isn't open when signal returns.
const sync = useShoppingSync()

// Tasks kept for offline reading belong to one user (see offlineTasks.ts).
const auth = useAuthStore()
watch(() => auth.info?.id, id => keepOfflineApiCacheFor(id ?? 0), {immediate: true})
</script>

<style lang="scss" scoped>
// Mirrors `.top-menu .menu-list li > a` in components/home/Navigation.vue:
// that rule is scoped to Navigation's own template, so it can't reach the
// link inside this component. The selector must be just as specific to beat
// the global `.menu-list a` padding. Keep in sync if upstream changes it.
.top-menu .menu-list li > a {
	display: inline-block;
	padding-inline-start: 2rem;

	.icon {
		padding-block-end: .25rem;
	}
}

.pending {
	margin-inline-start: .35rem;
	padding: 0 .4rem;
	border-radius: 999px;
	background: var(--grey-200);
	color: var(--grey-700);
	font-size: .75rem;
}
</style>
