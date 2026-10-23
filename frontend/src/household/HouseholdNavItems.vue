<template>
	<li>
		<RouterLink
			v-shortcut="SHORTCUTS.navigation.overview"
			:to="{name: 'home'}"
		>
			<span class="menu-item-icon icon">
				<Icon icon="house" />
			</span>
			{{ $t('household.home.title') }}
		</RouterLink>
	</li>
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
	<!-- Vikunja's task entries (Upcoming, Projects, …) follow in Navigation.vue. -->
	<li
		class="menu-divider"
		role="separator"
	>
		{{ $t('household.home.tasksHeading') }}
	</li>
</template>

<script setup lang="ts">
import {watch} from 'vue'

import {SHORTCUTS} from '@/constants/shortcuts'
import {useAuthStore} from '@/stores/auth'
import {keepOfflineApiCacheFor} from './offlineTasks'
import {refreshSubscription} from './push/device'
import {useShoppingSync} from './shopping/sync/useShoppingSync'

// Starting the engine here (the menu is on every page) means changes made
// offline are sent even if the shopping list isn't open when signal returns.
const sync = useShoppingSync()

// Tasks kept for offline reading belong to one user (see offlineTasks.ts).
const auth = useAuthStore()
watch(() => auth.info?.id, id => {
	keepOfflineApiCacheFor(id ?? 0)
	// Push: re-send this device's subscription for whoever is signed in.
	if (id) refreshSubscription()
}, {immediate: true})
</script>

<style lang="scss" scoped>
// Mirrors `.top-menu .menu-list li` (and `li > a`) in components/home/Navigation.vue:
// that rule is scoped to Navigation's own template, so it can't reach the
// entries inside this component. The selector must be just as specific to beat
// the global `.menu-list a` padding. Keep in sync if upstream changes it.
.top-menu .menu-list li {
	font-weight: 600;
	font-family: $vikunja-font;
}

.top-menu .menu-list li > a {
	display: inline-block;
	padding-inline-start: 2rem;

	.icon {
		padding-block-end: .25rem;
	}
}

.menu-divider {
	margin-block: 1rem .25rem;
	margin-inline: 1rem;
	padding-block-start: .75rem;
	padding-inline-start: 1rem;
	border-block-start: 1px solid var(--grey-200);
	color: var(--grey-500);
	font-family: inherit;
	font-size: .75rem;
	font-weight: normal;
	letter-spacing: .05em;
	text-transform: uppercase;
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
