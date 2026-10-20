<template>
	<nav
		class="list-switcher"
		:aria-label="$t('household.shopping.lists')"
	>
		<RouterLink
			v-for="list in lists"
			:key="list.id"
			:to="{name: 'shopping', params: {listId: list.id}}"
			class="list-tab"
			:class="{'is-active': list.id === activeId}"
			:aria-current="list.id === activeId ? 'page' : undefined"
		>
			{{ list.title }}
		</RouterLink>

		<form
			v-if="adding"
			class="new-list"
			@submit.prevent="submitNew"
		>
			<input
				v-model="newTitle"
				v-focus
				class="input is-small"
				type="text"
				maxlength="250"
				:placeholder="$t('household.shopping.newListPlaceholder')"
				@keydown.esc="adding = false"
				@blur="newTitle === '' && (adding = false)"
			>
		</form>
		<BaseButton
			v-else
			class="list-tab is-add"
			@click="adding = true"
		>
			<Icon icon="plus" />
			{{ $t('household.shopping.newList') }}
		</BaseButton>
	</nav>
</template>

<script setup lang="ts">
import {ref} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import type {ShoppingListResponse} from '../queries'

defineProps<{
	lists: ShoppingListResponse[]
	activeId: number
}>()

const emit = defineEmits<{
	create: [title: string]
}>()

const adding = ref(false)
const newTitle = ref('')

function submitNew() {
	const title = newTitle.value.trim()
	if (title !== '') {
		emit('create', title)
	}
	newTitle.value = ''
	adding.value = false
}
</script>

<style lang="scss" scoped>
.list-switcher {
	display: flex;
	gap: .5rem;
	margin-block-end: 1rem;
	padding-block-end: .25rem;
	overflow-x: auto; // many lists scroll sideways on a phone
}

.list-tab {
	flex: 0 0 auto;
	padding: .4rem .9rem;
	border-radius: 999px;
	background: var(--grey-100);
	color: var(--text);
	white-space: nowrap;

	&.is-active {
		background: var(--primary);
		color: var(--white);
	}

	&.is-add {
		color: var(--grey-500);
	}
}

.new-list .input {
	inline-size: 12rem;
}
</style>
