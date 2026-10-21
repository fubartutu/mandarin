<template>
	<!--
		The mandarin, drawn inline. It's a component rather than an .svg file
		because the gradients need ids, and ids must be unique on the page: the
		logo is drawn twice (top bar and sidebar), and when the first copy is
		hidden the second can't use its gradients, so the fruit vanished.
		useId() gives every copy its own. Same artwork as assets/mark.svg.
	-->
	<svg
		xmlns="http://www.w3.org/2000/svg"
		viewBox="0 0 256 256"
		width="256"
		height="256"
		role="img"
	>
		<title>{{ APP_NAME }}</title>
		<defs>
			<radialGradient
				:id="`${id}-peel`"
				cx="38%"
				cy="34%"
				r="75%"
			>
				<stop
					offset="0"
					stop-color="#ffc56b"
				/>
				<stop
					offset=".45"
					stop-color="#ff9a1f"
				/>
				<stop
					offset="1"
					stop-color="#e8680c"
				/>
			</radialGradient>
			<linearGradient
				:id="`${id}-leaf`"
				x1="0"
				y1="1"
				x2="1"
				y2="0"
			>
				<stop
					offset="0"
					stop-color="#2e8b3a"
				/>
				<stop
					offset="1"
					stop-color="#5cbf4a"
				/>
			</linearGradient>
			<linearGradient
				:id="`${id}-leaf2`"
				x1="1"
				y1="1"
				x2="0"
				y2="0"
			>
				<stop
					offset="0"
					stop-color="#2a7a33"
				/>
				<stop
					offset="1"
					stop-color="#4caf45"
				/>
			</linearGradient>
		</defs>
		<ellipse
			cx="128"
			cy="150"
			rx="104"
			ry="92"
			:fill="`url(#${id}-peel)`"
		/>
		<ellipse
			cx="94"
			cy="112"
			rx="30"
			ry="18"
			fill="#fff"
			opacity=".28"
			transform="rotate(-28 94 112)"
		/>
		<circle
			cx="128"
			cy="62"
			r="7"
			fill="#c95a08"
			opacity=".55"
		/>
		<path
			d="M128 63c1-9 2-15 6-21"
			fill="none"
			stroke="#6b4423"
			stroke-width="7"
			stroke-linecap="round"
		/>
		<path
			d="M133 44c16-26 52-34 80-24-12 28-48 42-80 24z"
			:fill="`url(#${id}-leaf)`"
		/>
		<path
			d="M137 43c22-12 46-18 72-21"
			fill="none"
			stroke="#c8f0b0"
			stroke-width="3"
			stroke-linecap="round"
			opacity=".7"
		/>
		<path
			d="M128 50c-14-20-40-26-62-18 10 22 36 30 62 18z"
			:fill="`url(#${id}-leaf2)`"
		/>
		<path
			d="M124 48c-16-8-34-12-54-14"
			fill="none"
			stroke="#c8f0b0"
			stroke-width="2.5"
			stroke-linecap="round"
			opacity=".6"
		/>
	</svg>
</template>

<script setup lang="ts">
import {useId} from 'vue'

import {APP_NAME} from '../branding'

const id = useId()
</script>
