<template>
	<!-- Mandarin: a Kanban card's subtitle, urgency stripe and hover comments. One FORK insert under the title in KanbanCard.vue. -->
	<p
		v-if="subtitle"
		ref="subtitleRef"
		class="kanban-card__subtitle"
		:title="subtitle"
	>
		{{ subtitle }}
	</p>
	<span
		v-else
		ref="subtitleRef"
		class="kanban-card__anchor"
		aria-hidden="true"
	/>

	<Teleport :to="teleportTarget">
		<CustomTransition name="fade">
			<div
				v-if="showComments && comments.length"
				ref="popupRef"
				class="kanban-comments"
				role="tooltip"
			>
				<p class="kanban-comments__heading">
					{{ $t('household.tasks.kanban.recentComments', {count: task.comment_count ?? comments.length}, task.comment_count ?? comments.length) }}
				</p>
				<div
					v-for="comment in comments"
					:key="comment.id"
					class="kanban-comments__comment"
				>
					<div class="kanban-comments__meta">
						<strong>{{ getDisplayName(comment.author) }}</strong>
						<span>{{ formatDateSince(comment.created) }}</span>
					</div>
					<p>{{ commentText(comment.comment) }}</p>
				</div>
			</div>
		</CustomTransition>
	</Teleport>
</template>

<script setup lang="ts">
import {computed, nextTick, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import {useMediaQuery} from '@vueuse/core'
import {useQuery} from '@tanstack/vue-query'
import {computePosition, flip, offset, shift} from '@floating-ui/dom'

import type {Task} from '@/client/generated'
import {commentsQuery} from '@/client/queries/comments'
import CustomTransition from '@/components/misc/CustomTransition.vue'
import {getTopLayerContainer} from '@/helpers/getTopLayerContainer'
import {formatDateSince} from '@/helpers/time/formatDate'
import {useGlobalNow} from '@/composables/useGlobalNow'
import {getDisplayName} from '@/helpers/user'
import {commentText, firstLine} from './cardText'
import {urgencyOf} from './urgency'

const props = defineProps<{
	task: Task
}>()

const HOVER_DELAY = 1000 // like Vikunja's TaskGlanceTooltip
const SHOWN = 3 // the newest comments

const subtitle = computed(() => firstLine(props.task.description))

// The stripe colour (household/kanban.scss styles the card by it). The shared
// clock ticks every minute, so "due today" turns into "overdue" on its own.
const {now} = useGlobalNow()
const urgency = computed(() => urgencyOf(props.task, now.value))

// Taps on phones fake a mouseenter; only real pointers get the popup.
const canHover = useMediaQuery('(hover: hover) and (pointer: fine)')

const subtitleRef = ref<HTMLElement | null>(null)
const popupRef = ref<HTMLElement | null>(null)
const showComments = ref(false)
const teleportTarget = ref<HTMLElement | string>('body')
let card: HTMLElement | null = null
let timer: ReturnType<typeof setTimeout> | null = null

// Fetched only once the popup is wanted; Vikunja's comment query, newest first.
const wanted = ref(false)
const commentsPage = useQuery(computed(() => ({
	...commentsQuery(props.task.id ?? 0, 'desc', 1),
	enabled: wanted.value && (props.task.id ?? 0) > 0,
})))
const comments = computed(() => (commentsPage.data.value?.items ?? []).slice(0, SHOWN))

function show() {
	if (!canHover.value || !(props.task.comment_count ?? 0)) return
	cancel()
	timer = setTimeout(async () => {
		timer = null
		wanted.value = true
		teleportTarget.value = getTopLayerContainer(card)
		showComments.value = true
		await place()
	}, HOVER_DELAY)
}

function cancel() {
	if (timer) clearTimeout(timer)
	timer = null
}

function hide() {
	cancel()
	showComments.value = false
}

async function place() {
	await nextTick()
	if (!card || !popupRef.value) return
	const {x, y} = await computePosition(card, popupRef.value, {
		strategy: 'absolute',
		placement: 'right-start',
		middleware: [offset(8), flip({fallbackPlacements: ['left-start', 'bottom', 'top'], padding: 8}), shift({padding: 8})],
	})
	popupRef.value.style.left = `${x}px`
	popupRef.value.style.top = `${y}px`
}

// The comments arrive after the popup opens; place it again once they're in.
watch(comments, () => {
	if (showComments.value) place()
})

function markUrgency() {
	if (!card) return
	if (urgency.value) card.dataset.householdUrgency = urgency.value
	else delete card.dataset.householdUrgency
}
watch(urgency, markUrgency)

// The card itself is Vikunja's (KanbanCard.vue): mark it and listen on it rather than wrap it.
onMounted(() => {
	card = subtitleRef.value?.closest('.task') ?? null
	markUrgency()
	card?.addEventListener('mouseenter', show)
	card?.addEventListener('mouseleave', hide)
	// Picking a card up to drag it closes the popup.
	card?.addEventListener('pointerdown', hide)
})

onBeforeUnmount(() => {
	hide()
	if (card) delete card.dataset.householdUrgency
	card?.removeEventListener('mouseenter', show)
	card?.removeEventListener('mouseleave', hide)
	card?.removeEventListener('pointerdown', hide)
})
</script>

<style lang="scss" scoped>
.kanban-card__subtitle {
	margin: .1rem 0 0;
	font-size: .85rem;
	color: var(--grey-500);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

// Light text on a coloured card: Vikunja sets has-light-text on the card.
:global(.task.has-light-text) .kanban-card__subtitle {
	color: var(--grey-200);
}

.kanban-card__anchor {
	display: none;
}

.kanban-comments {
	position: absolute;
	inset-block-start: 0;
	inset-inline-start: 0;
	z-index: 9999;
	inline-size: 300px;
	padding: .75rem;
	background: var(--white);
	border: 1px solid var(--grey-200);
	border-radius: $radius;
	box-shadow: var(--shadow-lg);
	pointer-events: none;
	font-size: .875rem;
}

.kanban-comments__heading {
	margin: 0 0 .5rem;
	font-weight: 600;
	color: var(--grey-600);
}

.kanban-comments__comment + .kanban-comments__comment {
	margin-block-start: .5rem;
	padding-block-start: .5rem;
	border-block-start: 1px solid var(--grey-100);
}

.kanban-comments__meta {
	display: flex;
	justify-content: space-between;
	gap: .5rem;
	color: var(--grey-500);
	font-size: .8rem;

	strong {
		color: var(--text);
	}
}

.kanban-comments__comment p {
	margin: .2rem 0 0;
	overflow-wrap: anywhere;
}
</style>
