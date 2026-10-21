<template>
	<!-- The mark and the name. The text takes the surrounding colour (light/dark mode). -->
	<svg
		xmlns="http://www.w3.org/2000/svg"
		viewBox="0 0 980 256"
		width="980"
		height="256"
		role="img"
	>
		<title>{{ APP_NAME }}</title>
		<MandarinMark
			x="0"
			y="0"
			width="256"
			height="256"
		/>
		<text
			x="276"
			y="178"
			fill="currentColor"
			font-family="Quicksand, sans-serif"
			font-size="150"
			font-weight="600"
			letter-spacing="-2"
		>{{ APP_NAME }}</text>
	</svg>
</template>

<script setup lang="ts">
import MandarinMark from './MandarinMark.vue'
import {APP_NAME} from '../branding'
</script>
