<template>
	<!--
		The mandarin: a picture (assets/mark.webp, cut out of
		assets/source/fruit-checkerboard.webp by scripts/household-icons.mjs)
		inside an <svg>, so MandarinLogo can place it next to the name with x,
		y, width and height like any other SVG shape.
	-->
	<svg
		xmlns="http://www.w3.org/2000/svg"
		viewBox="0 0 256 256"
		width="256"
		height="256"
		role="img"
	>
		<title>{{ APP_NAME }}</title>
		<image
			:href="markUrl"
			width="256"
			height="256"
		/>
	</svg>
</template>

<script setup lang="ts">
// Vite turns an image import into its URL (hashed in production builds).
import markUrl from '../assets/mark.webp'
import {APP_NAME} from '../branding'
</script>
