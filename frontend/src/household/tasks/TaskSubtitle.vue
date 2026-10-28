<template>
	<!-- Mandarin: a task row's subtitle, the description's first line (FORK in SingleTaskInProject.vue). -->
	<span
		v-if="line"
		class="task-subtitle"
		:title="line"
	>{{ line }}</span>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import {firstLine} from './kanban/cardText'

const props = defineProps<{
	description?: string | null
}>()

const line = computed(() => firstLine(props.description))
</script>

<style lang="scss" scoped>
// Its own line under the title, labels and due date, cut off with an ellipsis.
.task-subtitle {
	display: block;
	margin-block-start: .1rem;
	font-size: .85rem;
	color: var(--grey-500);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}
</style>
