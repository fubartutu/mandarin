<template>
	<section class="category-group">
		<h2 class="group-header">
			<button
				type="button"
				:aria-expanded="!collapsed"
				:aria-label="$t(collapsed ? 'household.shopping.expand' : 'household.shopping.collapse', {title})"
				@click="collapsed = !collapsed"
			>
				<Icon
					:icon="collapsed ? 'chevron-right' : 'chevron-down'"
					class="chevron"
				/>
				<span class="group-title">{{ title }}</span>
				<span class="count">{{ count }}</span>
			</button>
			<!-- A named slot lets the parent put extra buttons in the header. -->
			<slot name="actions" />
		</h2>
		<ul
			v-show="!collapsed"
			class="items"
		>
			<slot />
		</ul>
	</section>
</template>

<script setup lang="ts">
import {useStorage} from '@vueuse/core'

const props = defineProps<{
	// Used to remember whether this group is collapsed on this device.
	storageKey: string
	title: string
	count: number
}>()

// eslint-disable-next-line vue/no-setup-props-reactivity-loss -- a group's key never changes; v-for keys groups by category
const collapsed = useStorage(`mandarin.shopping.collapsed.${props.storageKey}`, false)
</script>

<style lang="scss" scoped>
.category-group {
	margin-block-end: 1rem;
	border-radius: $radius;
	overflow: hidden;
	box-shadow: var(--shadow-sm);
}

.group-header {
	display: flex;
	align-items: center;
	margin: 0;
	background: var(--grey-100);
	font-size: .9rem;
	font-weight: 600;

	button {
		flex: 1;
		display: flex;
		align-items: center;
		gap: .5rem;
		padding: .6rem .75rem;
		border: 0;
		background: transparent;
		color: var(--text);
		font: inherit;
		text-align: start;
		cursor: pointer;
	}

	.chevron {
		inline-size: .75rem;
		color: var(--grey-500);
	}

	.count {
		margin-inline-start: auto;
		color: var(--grey-500);
		font-weight: normal;
	}
}

.items {
	margin: 0;
	padding: 0;
	list-style: none;
}
</style>
