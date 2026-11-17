<template>
	<!-- Mandarin: "Break down" on the task page (FORK in TaskDetailView.vue). Opens a panel where
		the AI (or you) suggests steps or small chunks, which you edit before adding them. -->
	<XButton
		variant="secondary"
		icon="sitemap"
		@click="open"
	>
		{{ $t('household.tasks.breakdown.button') }}
	</XButton>

	<Modal
		v-if="shown"
		:aria-label="$t('household.tasks.breakdown.title')"
		@close="close"
	>
		<Card
			:title="$t('household.tasks.breakdown.title')"
			:shadow="false"
			:show-close="true"
			class="has-text-start breakdown"
			@close="close"
		>
			<p class="breakdown__task">
				{{ task.title }}
			</p>

			<div
				class="breakdown__modes"
				role="radiogroup"
				:aria-label="$t('household.tasks.breakdown.modeLabel')"
			>
				<XButton
					v-for="m in MODES"
					:key="m"
					role="radio"
					:aria-checked="mode === m"
					:variant="mode === m ? 'primary' : 'secondary'"
					:shadow="false"
					@click="mode = m"
				>
					{{ $t(`household.tasks.breakdown.modes.${m}`) }}
				</XButton>
			</div>
			<p class="help breakdown__hint">
				{{ $t(`household.tasks.breakdown.hints.${mode}`) }}
			</p>

			<XButton
				variant="secondary"
				icon="lightbulb"
				:shadow="false"
				:loading="suggest.isPending.value"
				class="breakdown__suggest"
				@click="askAi"
			>
				{{ items.length ? $t('household.tasks.breakdown.suggestAgain') : $t('household.tasks.breakdown.suggest') }}
			</XButton>
			<p
				v-if="suggest.isPending.value"
				class="help"
			>
				{{ $t('household.tasks.breakdown.thinking') }}
			</p>
			<p
				v-if="aiError"
				class="help is-danger"
				role="alert"
			>
				{{ aiError }} {{ $t('household.tasks.breakdown.typeInstead') }}
			</p>

			<ol
				v-if="items.length"
				class="breakdown__items"
			>
				<li
					v-for="(item, index) in items"
					:key="item.key"
					class="breakdown__item"
				>
					<input
						:ref="el => setInput(item.key, el)"
						v-model="item.title"
						class="input"
						:aria-label="$t('household.tasks.breakdown.itemLabel', {n: index + 1})"
						maxlength="250"
						@keydown.enter.prevent="addAfter(index)"
					>
					<span
						v-if="item.minutes > 0"
						class="breakdown__minutes"
					>{{ $t('household.tasks.breakdown.minutes', {n: item.minutes}) }}</span>
					<BaseButton
						class="breakdown__icon"
						:disabled="index === 0"
						:aria-label="$t('household.tasks.breakdown.moveUp')"
						@click="move(index, -1)"
					>
						<Icon icon="chevron-up" />
					</BaseButton>
					<BaseButton
						class="breakdown__icon"
						:disabled="index === items.length - 1"
						:aria-label="$t('household.tasks.breakdown.moveDown')"
						@click="move(index, 1)"
					>
						<Icon icon="chevron-down" />
					</BaseButton>
					<BaseButton
						class="breakdown__icon"
						:aria-label="$t('household.tasks.breakdown.remove', {title: item.title})"
						@click="remove(index)"
					>
						<Icon icon="trash-alt" />
					</BaseButton>
				</li>
			</ol>

			<template v-if="items.length">
				<BaseButton
					v-if="items.length < MAX_BREAKDOWN_ITEMS"
					class="breakdown__add"
					@click="addAfter(items.length - 1)"
				>
					<Icon icon="plus" /> {{ $t('household.tasks.breakdown.addOne') }}
				</BaseButton>
				<p
					v-if="total > 0"
					class="help"
				>
					{{ $t('household.tasks.breakdown.total', {time: totalLabel}) }}
				</p>
			</template>
			<div
				v-else
				class="breakdown__typed"
			>
				<label
					class="label"
					:for="typedId"
				>{{ $t('household.tasks.breakdown.typeLabel') }}</label>
				<textarea
					:id="typedId"
					v-model="typed"
					class="textarea"
					rows="4"
					:placeholder="$t(`household.tasks.breakdown.placeholders.${mode}`)"
				/>
				<XButton
					variant="secondary"
					:shadow="false"
					:disabled="!typed.trim()"
					@click="useTyped"
				>
					{{ $t('household.tasks.breakdown.useTyped') }}
				</XButton>
			</div>

			<template #footer>
				<XButton
					variant="secondary"
					:shadow="false"
					@click="close"
				>
					{{ $t('household.tasks.breakdown.cancel') }}
				</XButton>
				<XButton
					variant="primary"
					:shadow="false"
					:disabled="kept.length === 0 || kept.length > MAX_BREAKDOWN_ITEMS"
					:loading="apply.isPending.value"
					@click="save"
				>
					{{ $t(`household.tasks.breakdown.add.${mode}`, {n: kept.length}, kept.length) }}
				</XButton>
			</template>
		</Card>
	</Modal>
</template>

<script setup lang="ts">
import {computed, nextTick, ref, useId, watch, type ComponentPublicInstance} from 'vue'
import {useI18n} from 'vue-i18n'

import BaseButton from '@/components/base/BaseButton.vue'
import type {Task} from '@/client/generated'
import {getErrorText, success} from '@/message'
import {
	MAX_BREAKDOWN_ITEMS, defaultMode, itemText, moveItem, newItem, parseTyped, totalMinutes,
	type BreakdownItem, type BreakdownMode,
} from './breakdown'
import {useApplyBreakdownMutation, useSuggestBreakdownMutation} from './queries'

const props = defineProps<{
	task: Task
}>()

const emit = defineEmits<{
	/** The task as the server left it (new subtasks or description): TaskDetailView takes it. */
	done: [task: Task]
}>()

const MODES: BreakdownMode[] = ['steps', 'chunks']

const {t} = useI18n({useScope: 'global'})
const typedId = useId()

const shown = ref(false)
const mode = ref<BreakdownMode>('steps')
const items = ref<BreakdownItem[]>([])
const typed = ref('')
const aiError = ref('')

const suggest = useSuggestBreakdownMutation()
const apply = useApplyBreakdownMutation()

// Switching kind keeps what's typed, but an AI answer for the other kind no longer fits.
watch(mode, () => {
	aiError.value = ''
})

function open() {
	mode.value = defaultMode(props.task)
	items.value = []
	typed.value = ''
	aiError.value = ''
	shown.value = true
}

function close() {
	shown.value = false
}

async function askAi() {
	aiError.value = ''
	try {
		const suggested = await suggest.mutateAsync({taskId: props.task.id!, mode: mode.value})
		items.value = suggested.slice(0, MAX_BREAKDOWN_ITEMS)
	} catch (e) {
		aiError.value = getErrorText(e)
	}
}

function useTyped() {
	items.value = parseTyped(typed.value).slice(0, MAX_BREAKDOWN_ITEMS)
	typed.value = ''
}

// Inputs by item key, so a new item can be focused once it's drawn.
const inputs = new Map<number, HTMLInputElement>()
function setInput(key: number, el: Element | ComponentPublicInstance | null) {
	if (el instanceof HTMLInputElement) inputs.set(key, el)
	else inputs.delete(key)
}

async function addAfter(index: number) {
	if (items.value.length >= MAX_BREAKDOWN_ITEMS) return
	const item = newItem()
	items.value.splice(index + 1, 0, item)
	await nextTick()
	inputs.get(item.key)?.focus()
}

function move(index: number, delta: number) {
	items.value = moveItem(items.value, index, delta)
}

function remove(index: number) {
	items.value.splice(index, 1)
}

const kept = computed(() => items.value.map(i => itemText(i, mode.value)).filter(text => text !== ''))

const total = computed(() => totalMinutes(items.value))
const totalLabel = computed(() => {
	const h = Math.floor(total.value / 60)
	const m = total.value % 60
	if (h === 0) return t('household.tasks.breakdown.minutes', {n: m})
	return m === 0
		? t('household.tasks.breakdown.hours', {n: h})
		: `${t('household.tasks.breakdown.hours', {n: h})} ${t('household.tasks.breakdown.minutes', {n: m})}`
})

async function save() {
	const count = kept.value.length
	const task = await apply.mutateAsync({taskId: props.task.id!, mode: mode.value, items: kept.value})
	emit('done', task)
	shown.value = false
	success({message: t(`household.tasks.breakdown.added.${mode.value}`, {n: count}, count)})
}
</script>

<style lang="scss" scoped>
.breakdown__task {
	font-weight: 600;
	margin-block-end: .75rem;
}

.breakdown__modes {
	display: flex;
	flex-wrap: wrap;
	gap: .5rem;
}

.breakdown__hint {
	margin-block: .25rem .75rem;
}

.breakdown__suggest {
	margin-block-end: .25rem;
}

.breakdown__items {
	list-style: decimal;
	padding-inline-start: 1.5rem;
	margin-block: .75rem .5rem;
}

.breakdown__item {
	display: flex;
	align-items: center;
	gap: .25rem;
	margin-block-end: .4rem;

	.input {
		flex: 1;
		min-inline-size: 0;
	}
}

.breakdown__minutes {
	color: var(--grey-500);
	font-size: .85rem;
	white-space: nowrap;
}

.breakdown__icon {
	padding: .4rem;
	color: var(--grey-500);

	&:disabled {
		opacity: .3;
	}
}

.breakdown__add {
	color: var(--primary);
	margin-block-end: .5rem;
}

.breakdown__typed {
	margin-block-start: .75rem;

	.textarea {
		margin-block-end: .5rem;
	}
}
</style>
