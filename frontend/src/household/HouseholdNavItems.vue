<template>
	<li>
		<RouterLink :to="{name: 'shopping'}">
			<span class="menu-item-icon icon">
				<Icon icon="basket-shopping" />
			</span>
			{{ $t('household.shopping.title') }}
		</RouterLink>
	</li>
</template>

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
</style>
