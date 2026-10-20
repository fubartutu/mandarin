<template>
	<div class="sync-badge">
		<button
			type="button"
			class="badge"
			:class="`is-${shown}`"
			:aria-expanded="open"
			:aria-label="$t('household.shopping.sync.details')"
			@click="toggle"
		>
			<Icon
				:icon="icons[shown]"
				:spin="shown === 'syncing'"
			/>
			<span>{{ label }}</span>
			<Icon
				v-if="unseenConflicts > 0"
				icon="triangle-exclamation"
				class="warning"
				:title="$t('household.shopping.sync.conflicts.new')"
			/>
		</button>

		<section
			v-if="open"
			class="panel"
		>
			<p>{{ help }}</p>
			<p class="muted">
				{{ lastSyncedAt ? $t('household.shopping.sync.lastSynced', {when: formatDateSince(new Date(lastSyncedAt))}) : $t('household.shopping.sync.neverSynced') }}
				{{ $t('household.shopping.sync.pendingHelp', {count: pendingCount}, pendingCount) }}
			</p>
			<XButton
				variant="secondary"
				icon="arrows-rotate"
				:loading="status === 'syncing'"
				@click="sync.sync()"
			>
				{{ $t('household.shopping.sync.syncNow') }}
			</XButton>

			<ConflictLog
				:conflicts="sync.conflicts.value"
				:categories="categories"
				:seen-up-to="seenAtOpen"
			/>
		</section>
	</div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {useStorage} from '@vueuse/core'

import {formatDateSince} from '@/helpers/time/formatDate'
import ConflictLog from './ConflictLog.vue'
import type {ShoppingCategoryResponse} from '../queries'
import {useShoppingSync} from '../sync/useShoppingSync'

defineProps<{
	categories: ShoppingCategoryResponse[]
}>()

const {t} = useI18n({useScope: 'global'})
const sync = useShoppingSync()
const status = sync.status
const pendingCount = sync.pendingCount
const lastSyncedAt = sync.lastSyncedAt

// A background check with nothing to send still reads "Synced", so the
// badge doesn't flicker every couple of minutes.
const shown = computed(() => {
	if (status.value === 'syncing' && pendingCount.value === 0 && lastSyncedAt.value) return 'synced'
	if (status.value === 'synced' && pendingCount.value > 0) return 'syncing'
	return status.value
})

const icons = {
	loading: 'arrows-rotate',
	syncing: 'arrows-rotate',
	synced: 'circle-check',
	offline: 'cloud',
	error: 'triangle-exclamation',
} as const

const label = computed(() => {
	const waiting = pendingCount.value > 0
		? ' · ' + t('household.shopping.sync.waiting', {count: pendingCount.value}, pendingCount.value)
		: ''
	switch (shown.value) {
		case 'offline': return t('household.shopping.sync.offline') + waiting
		case 'error': return t('household.shopping.sync.error') + waiting
		case 'syncing': return t('household.shopping.sync.syncing')
		case 'loading': return t('household.shopping.sync.loading')
		default: return t('household.shopping.sync.synced')
	}
})

const help = computed(() => {
	switch (status.value) {
		case 'offline': return t('household.shopping.sync.offlineHelp')
		case 'error': return t('household.shopping.sync.errorHelp') + (sync.lastError.value ? ` (${sync.lastError.value})` : '')
		default: return t('household.shopping.sync.conflicts.help')
	}
})

// Conflicts newer than the last one looked at get a warning on the badge.
const seenConflict = useStorage('mandarin.shopping.conflictSeen', 0)
const unseenConflicts = computed(() => sync.conflicts.value.filter(c => c.id > seenConflict.value).length)

const open = ref(false)
// Remembered when the panel opens, so the new ones stay highlighted while it's open.
const seenAtOpen = ref(0)

function toggle() {
	open.value = !open.value
	if (open.value) {
		seenAtOpen.value = seenConflict.value
		seenConflict.value = Math.max(seenConflict.value, ...sync.conflicts.value.map(c => c.id))
	}
}
</script>

<style lang="scss" scoped>
.badge {
	display: inline-flex;
	align-items: center;
	gap: .35rem;
	padding: .2rem .6rem;
	border: 1px solid var(--grey-200);
	border-radius: 999px;
	background: var(--white);
	color: var(--grey-600);
	font-size: .8rem;
	white-space: nowrap;
	cursor: pointer;

	&.is-synced .icon {
		color: var(--success);
	}

	&.is-offline {
		background: var(--grey-100);
	}

	&.is-error .icon {
		color: var(--danger);
	}

	.warning {
		color: var(--warning);
	}
}

// The parent header is position: relative, so the panel spans its width.
.panel {
	position: absolute;
	inset-block-start: calc(100% + .25rem);
	inset-inline: 0;
	z-index: 5;
	padding: 1rem;
	border: 1px solid var(--grey-200);
	border-radius: $radius;
	background: var(--white);
	box-shadow: var(--shadow-md);
	font-size: .9rem;

	p {
		margin-block-end: .5rem;
	}

	.muted {
		color: var(--grey-500);
	}
}
</style>
