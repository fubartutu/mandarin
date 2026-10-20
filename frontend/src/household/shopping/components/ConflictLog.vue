<template>
	<div class="conflict-log">
		<h2 class="log-title">
			{{ $t('household.shopping.sync.conflicts.title') }}
		</h2>
		<p
			v-if="conflicts.length === 0"
			class="muted"
		>
			{{ $t('household.shopping.sync.conflicts.none') }}
		</p>
		<ul v-else>
			<li
				v-for="conflict in conflicts"
				:key="conflict.id"
				:class="{'is-new': conflict.id > seenUpTo}"
			>
				<template v-if="conflict.field === 'deleted'">
					{{ $t('household.shopping.sync.conflicts.deleted', {name: conflict.item_name}) }}
					<span class="muted">{{ who(conflict.loser_by, conflict.loser_at) }}</span>
				</template>
				<template v-else>
					<strong>{{ $t(`household.shopping.sync.conflicts.fields.${conflict.field}`, {name: conflict.item_name}) }}</strong>
					<span class="kept">
						{{ $t('household.shopping.sync.conflicts.kept', {value: value(conflict.field, conflict.winner_value)}) }}
						<span class="muted">· {{ who(conflict.winner_by, conflict.winner_at) }}</span>
					</span>
					<span>
						<!-- Only the lost value is struck through; a parent's line-through can't be undone in a child. -->
						<span class="lost">{{ $t('household.shopping.sync.conflicts.lost', {value: value(conflict.field, conflict.loser_value)}) }}</span>
						<span class="muted"> · {{ who(conflict.loser_by, conflict.loser_at) }}</span>
					</span>
				</template>
			</li>
		</ul>
	</div>
</template>

<script setup lang="ts">
import {useI18n} from 'vue-i18n'

import type {User} from '@/client/generated'
import {getDisplayName} from '@/helpers/user'
import {formatDateSince} from '@/helpers/time/formatDate'
import type {ShoppingCategoryResponse} from '../queries'
import type {ConflictEntry} from '../sync/model'

const props = defineProps<{
	conflicts: ConflictEntry[]
	categories: ShoppingCategoryResponse[]
	seenUpTo: number // conflicts with a higher id are highlighted as new
}>()

const {t} = useI18n({useScope: 'global'})

// The server stores values as text; turn the ones that aren't words into words.
function value(field: string, raw: string): string {
	if (field === 'done') {
		return t(raw === 'true' ? 'household.shopping.sync.conflicts.ticked' : 'household.shopping.sync.conflicts.notTicked')
	}
	if (field === 'category_id') {
		return props.categories.find(c => String(c.id) === raw)?.title ?? t('household.shopping.noCategory')
	}
	return raw === '' ? t('household.shopping.sync.conflicts.empty') : `"${raw}"`
}

function who(user: User | undefined, at: string): string {
	return t('household.shopping.sync.conflicts.by', {
		who: user ? getDisplayName(user) : t('household.shopping.sync.conflicts.someone'),
		when: formatDateSince(at),
	})
}
</script>

<style lang="scss" scoped>
.conflict-log {
	margin-block-start: 1rem;
}

.log-title {
	margin-block-end: .5rem;
	font-size: 1rem;
	font-weight: 600;
}

ul {
	margin: 0;
	list-style: none;
}

li {
	display: flex;
	flex-direction: column;
	padding-block: .5rem;
	border-block-start: 1px solid var(--grey-200);

	&.is-new {
		border-inline-start: 3px solid var(--warning);
		padding-inline-start: .5rem;
	}
}

.lost {
	color: var(--grey-500);
	text-decoration: line-through;
	text-decoration-color: var(--grey-400);
}

.muted {
	color: var(--grey-500);
	font-size: .85rem;
}
</style>
