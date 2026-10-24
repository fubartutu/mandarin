<template>
	<div class="menu-week">
		<header class="menu-header">
			<h1 class="title">
				{{ $t('household.menu.title') }}
			</h1>
			<nav class="week-nav">
				<RouterLink
					class="button is-small"
					:to="weekRoute(shiftWeek(monday, -1))"
					:title="$t('household.menu.previousWeek')"
					:aria-label="$t('household.menu.previousWeek')"
				>
					<Icon icon="chevron-left" />
				</RouterLink>
				<span class="week-label">{{ weekLabel }}</span>
				<RouterLink
					class="button is-small"
					:to="weekRoute(shiftWeek(monday, 1))"
					:title="$t('household.menu.nextWeek')"
					:aria-label="$t('household.menu.nextWeek')"
				>
					<Icon icon="chevron-right" />
				</RouterLink>
				<RouterLink
					v-if="monday !== mondayOf(now)"
					class="button is-small"
					:to="weekRoute(mondayOf(now))"
				>
					{{ $t('household.menu.thisWeek') }}
				</RouterLink>
			</nav>
		</header>

		<p
			v-if="entries.isPending.value"
			class="has-text-centered has-text-grey"
		>
			{{ $t('household.menu.loading') }}
		</p>
		<p
			v-else-if="entries.isError.value && !entries.data.value"
			class="has-text-centered has-text-grey is-italic"
		>
			{{ $t('household.menu.notLoaded') }}
		</p>

		<ol
			v-else
			class="days"
		>
			<li
				v-for="day in days"
				:key="day"
				:ref="el => setDayRef(day, el)"
				class="day-card"
				:class="`is-${dayStatus(day, today)}`"
				:aria-current="day === today ? 'date' : undefined"
			>
				<div class="day-head">
					<h2 class="day-name">
						{{ dayLabel(day) }}
						<span
							v-if="day === today"
							class="today-badge"
						>{{ $t('household.menu.today') }}</span>
					</h2>
					<button
						v-if="adding !== day"
						type="button"
						class="button is-small is-text add-button"
						:title="$t('household.menu.add', {day: dayLabel(day)})"
						:aria-label="$t('household.menu.add', {day: dayLabel(day)})"
						@click="startAdding(day)"
					>
						<Icon icon="plus" />
					</button>
				</div>

				<ul
					v-if="(dinners.get(day) ?? []).length > 0"
					class="entries"
				>
					<li
						v-for="entry in dinners.get(day)"
						:key="entry.id"
						class="entry"
					>
						<span class="entry-note">{{ entry.note }}</span>
						<button
							type="button"
							class="button is-small is-text delete-button"
							:title="$t('household.menu.delete', {note: entry.note})"
							:aria-label="$t('household.menu.delete', {note: entry.note})"
							:disabled="busy"
							@click="remove(entry.id)"
						>
							<Icon icon="trash-alt" />
						</button>
					</li>
				</ul>
				<p
					v-else-if="adding !== day"
					class="nothing"
				>
					{{ $t('household.menu.nothingPlanned') }}
				</p>

				<form
					v-if="adding === day"
					class="add-form"
					@submit.prevent="submitNote(day)"
				>
					<input
						v-model="draft"
						v-focus
						class="input is-small"
						type="text"
						maxlength="250"
						:placeholder="$t('household.menu.notePlaceholder')"
						:aria-label="$t('household.menu.noteLabel', {day: dayLabel(day)})"
						@keydown.esc="stopAdding"
					>
					<button
						type="submit"
						class="button is-small is-primary"
						:disabled="busy || draft.trim() === ''"
					>
						{{ $t('household.menu.save') }}
					</button>
					<button
						type="button"
						class="button is-small"
						@click="stopAdding"
					>
						{{ $t('household.menu.cancel') }}
					</button>
				</form>
			</li>
		</ol>
	</div>
</template>

<script setup lang="ts">
import {computed, nextTick, ref, watch, type ComponentPublicInstance} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'
import {useNow} from '@vueuse/core'

import {useTitle} from '@/composables/useTitle'

import {menuRangeQuery, useAddNoteMutation, useDeleteEntryMutation} from './queries'
import {dayKey, dayStatus, dinnersByDay, fromKey, mondayOf, shiftWeek, weekDays} from './week'

const props = defineProps<{
	/** Any day of the week to show, YYYY-MM-DD; empty for this week. */
	week: string
}>()

const {t, locale} = useI18n({useScope: 'global'})
useTitle(() => t('household.menu.title'))

// Re-checked every minute, so "today" moves on at midnight with the page open.
const now = useNow({interval: 60 * 1000})
const today = computed(() => dayKey(now.value))

const monday = computed(() => mondayOf(props.week ? fromKey(props.week) : now.value))
const days = computed(() => weekDays(monday.value))

const entries = useQuery(computed(() => menuRangeQuery(days.value[0], days.value[6])))
const dinners = computed(() => dinnersByDay(entries.data.value ?? [], days.value))

function weekRoute(weekMonday: string) {
	return weekMonday === mondayOf(now.value)
		? {name: 'menu'}
		: {name: 'menu', params: {week: weekMonday}}
}

function dayLabel(day: string) {
	return fromKey(day).toLocaleDateString(locale.value, {weekday: 'long', day: 'numeric', month: 'short'})
}

const weekLabel = computed(() => {
	const options: Intl.DateTimeFormatOptions = {day: 'numeric', month: 'short', year: 'numeric'}
	// formatRange writes "19–25 Oct 2026" or "28 Dec 2026 – 3 Jan 2027" as the locale prefers.
	return new Intl.DateTimeFormat(locale.value, options).formatRange(fromKey(days.value[0]), fromKey(days.value[6]))
})

// ---- Today scrolled into view ----

const dayRefs = new Map<string, HTMLElement>()
function setDayRef(day: string, el: Element | ComponentPublicInstance | null) {
	if (el instanceof HTMLElement) dayRefs.set(day, el)
	else dayRefs.delete(day)
}

// Once per week shown, when its days are on screen.
let scrolledFor = ''
watch(() => [monday.value, entries.data.value !== undefined] as const, async ([shown, loaded]) => {
	if (!loaded || scrolledFor === shown) return
	scrolledFor = shown
	await nextTick()
	dayRefs.get(today.value)?.scrollIntoView({block: 'center'})
}, {immediate: true})

// ---- Adding and deleting notes ----

const adding = ref('')
const draft = ref('')

// A local directive: in <script setup>, a `vFoo` constant is usable as
// `v-foo` in the template. The note field is focused each time it appears
// (the autofocus attribute only works once per page load).
const vFocus = {mounted: (el: HTMLElement) => el.focus()}

function startAdding(day: string) {
	adding.value = day
	draft.value = ''
}

function stopAdding() {
	adding.value = ''
	draft.value = ''
}

const addNote = useAddNoteMutation()
const deleteEntry = useDeleteEntryMutation()
// One write at a time: the server's SQLite allows a single writer.
const busy = computed(() => addNote.isPending.value || deleteEntry.isPending.value)

// Only dinner is shown for now, so notes go to dinner.
function submitNote(day: string) {
	const note = draft.value.trim()
	if (!note || busy.value) return
	// On failure the mutation shows a toast and the typed note stays.
	addNote.mutate({day, meal: 'dinner', note}, {onSuccess: stopAdding})
}

function remove(id: number) {
	if (busy.value) return
	deleteEntry.mutate(id)
}
</script>

<style lang="scss" scoped>
.menu-week {
	max-inline-size: 40rem;
	margin-inline: auto;
}

.menu-header {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: space-between;
	gap: .75rem;
	margin-block-end: 1rem;

	.title {
		margin: 0;
	}
}

.week-nav {
	display: flex;
	align-items: center;
	gap: .4rem;
}

.week-label {
	min-inline-size: 9rem;
	font-weight: 600;
	text-align: center;
}

.days {
	display: flex;
	flex-direction: column;
	gap: .75rem;
	margin: 0;
	padding: 0;
	list-style: none;
}

.day-card {
	padding: .75rem 1rem;
	border: 2px solid transparent;
	border-radius: $radius;
	background: var(--white);
	box-shadow: var(--shadow-sm);

	&.is-today {
		border-color: var(--primary);
	}

	&.is-past {
		opacity: .6;
	}
}

.day-head {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: .5rem;
}

.day-name {
	margin: 0;
	font-size: 1rem;
	font-weight: 600;
}

.today-badge {
	margin-inline-start: .5rem;
	padding: .1rem .5rem;
	border-radius: 999px;
	background: var(--primary);
	color: var(--white);
	font-size: .75rem;
	vertical-align: middle;
}

.entries {
	margin: .4rem 0 0;
	padding: 0;
	list-style: none;
}

.entry {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: .5rem;
	padding-block: .2rem;
}

.entry-note {
	overflow-wrap: anywhere;
}

.nothing {
	margin-block-start: .25rem;
	color: var(--grey-500);
	font-style: italic;
}

.add-form {
	display: flex;
	gap: .4rem;
	margin-block-start: .5rem;

	.input {
		flex: 1;
	}
}
</style>
