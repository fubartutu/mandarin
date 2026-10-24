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
						<span
							v-if="entry.recipeId"
							class="entry-main"
						>
							<RouterLink
								class="entry-recipe"
								:to="{name: 'recipe', params: {recipeId: entry.recipeId}}"
							>
								{{ entryLabel(entry) }}
							</RouterLink>
							<form
								v-if="editingServings === entry.id"
								class="servings-form"
								@submit.prevent="saveServings(entry)"
							>
								<input
									v-model.number="servingsDraft"
									v-focus
									class="input is-small servings-input"
									type="number"
									step="1"
									min="1"
									:aria-label="$t('household.menu.servingsLabel', {recipe: entryLabel(entry)})"
									@keydown.esc="stopEditingServings"
								>
								<button
									type="submit"
									class="button is-small is-primary"
									:disabled="busy || !validServings"
								>
									{{ $t('household.menu.saveServings') }}
								</button>
								<button
									type="button"
									class="button is-small"
									@click="stopEditingServings"
								>
									{{ $t('household.menu.cancel') }}
								</button>
							</form>
							<span
								v-else-if="entry.servings > 0"
								class="entry-servings"
							>· {{ $t('household.menu.servings', {count: entry.servings}, entry.servings) }}</span>
						</span>
						<span
							v-else
							class="entry-note"
						>{{ entry.note }}</span>
						<Dropdown
							class="entry-options"
							trigger-icon="ellipsis-h"
							:trigger-label="$t('household.menu.options', {entry: entryLabel(entry)})"
						>
							<template #default="{close}">
								<DropdownItem
									v-for="action in entryActions(entry)"
									:key="action.key"
									:icon="action.icon"
									:class="{'has-text-danger': action.danger}"
									:icon-class="{'has-text-danger': action.danger}"
									:disabled="busy"
									@click="close(); action.run()"
								>
									{{ action.label }}
								</DropdownItem>
							</template>
						</Dropdown>
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
					@submit.prevent="submitDraft(day)"
				>
					<div class="add-row">
						<input
							v-model="draft"
							v-focus
							class="input is-small"
							type="text"
							maxlength="250"
							role="combobox"
							aria-autocomplete="list"
							:aria-expanded="matches.length > 0"
							:aria-controls="`menu-matches-${day}`"
							:aria-activedescendant="highlighted >= 0 ? `menu-match-${day}-${highlighted}` : undefined"
							:placeholder="$t('household.menu.notePlaceholder')"
							:aria-label="$t('household.menu.noteLabel', {day: dayLabel(day)})"
							@keydown.esc="stopAdding"
							@keydown.down.prevent="moveHighlight(1)"
							@keydown.up.prevent="moveHighlight(-1)"
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
					</div>
					<ul
						v-if="matches.length > 0"
						:id="`menu-matches-${day}`"
						class="recipe-matches"
						role="listbox"
						:aria-label="$t('household.menu.recipeMatches')"
					>
						<!-- mousedown.prevent keeps the focus in the field while tapping. -->
						<li
							v-for="(recipe, i) in matches"
							:id="`menu-match-${day}-${i}`"
							:key="recipe.id"
							class="recipe-match"
							:class="{'is-highlighted': i === highlighted}"
							role="option"
							:aria-selected="i === highlighted"
							@mousedown.prevent
							@click="pickRecipe(day, recipe.id)"
						>
							{{ recipe.title }}
						</li>
					</ul>
				</form>
			</li>
		</ol>

		<PickDayModal
			v-if="moving"
			:title="$t('household.menu.moveTitle', {entry: entryLabel(moving)})"
			:confirm-label="$t('household.menu.move')"
			:day="moving.day"
			:busy="updateEntry.isPending.value"
			@pick="move"
			@close="moving = null"
		/>

		<AddToListModal
			v-if="addingToList"
			:recipe-id="addingToList.recipeId"
			:recipe-title="entryLabel(addingToList)"
			:servings="addingToList.servings"
			@close="addingToList = null"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed, nextTick, ref, watch, type ComponentPublicInstance} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'
import {useNow} from '@vueuse/core'
import {useRouter} from 'vue-router'

import type {IconProp} from '@fortawesome/fontawesome-svg-core'

import Dropdown from '@/components/misc/Dropdown.vue'
import DropdownItem from '@/components/misc/DropdownItem.vue'
import {useTitle} from '@/composables/useTitle'
import {success} from '@/message'

import {recipesQuery} from '../recipes/queries'
import AddToListModal from './components/AddToListModal.vue'
import PickDayModal from './components/PickDayModal.vue'
import {cookRoute, menuWeekRoute, plannedDayLabel} from './planning'
import {matchRecipes} from './recipeMatches'
import {
	menuRangeQuery,
	useAddEntryMutation,
	useDeleteEntryMutation,
	useUpdateEntryMutation,
	type MenuEntryResponse,
	type NewEntry,
} from './queries'
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

// ---- Adding notes and recipes ----

const adding = ref('')
const draft = ref('')

// The recipe list is only fetched once "+" is open; it's filtered on the device.
const recipes = useQuery(computed(() => ({...recipesQuery(), enabled: adding.value !== ''})))
const matches = computed(() => matchRecipes(recipes.data.value ?? [], draft.value))
// The match picked with the arrow keys, or -1 for none (Enter then adds a note).
const highlighted = ref(-1)
watch(draft, () => {
	highlighted.value = -1
})

function moveHighlight(step: number) {
	const count = matches.value.length
	if (count === 0) return
	// Wraps around, and passes through -1 (no match picked) on the way.
	highlighted.value = ((highlighted.value + 1 + step + count + 1) % (count + 1)) - 1
}

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

const addEntry = useAddEntryMutation()
const updateEntry = useUpdateEntryMutation()
const deleteEntry = useDeleteEntryMutation()
// One write at a time: the server's SQLite allows a single writer.
const busy = computed(() => addEntry.isPending.value || updateEntry.isPending.value || deleteEntry.isPending.value)

// Only dinner is shown for now, so everything goes to dinner.
function add(entry: NewEntry) {
	if (busy.value) return
	// On failure the mutation shows a toast and the typed text stays.
	addEntry.mutate(entry, {onSuccess: stopAdding})
}

function pickRecipe(day: string, recipeId: number) {
	add({day, meal: 'dinner', recipeId})
}

// Enter or "Add": the highlighted recipe, else the text as a note.
function submitDraft(day: string) {
	const picked = matches.value[highlighted.value]
	if (picked) {
		pickRecipe(day, picked.id)
		return
	}
	const note = draft.value.trim()
	if (note) add({day, meal: 'dinner', note})
}

// ---- An entry's menu ----

function entryLabel(entry: MenuEntryResponse) {
	if (!entry.recipeId) return entry.note
	return entry.recipeTitle || t('household.menu.missingRecipe')
}

interface EntryAction {
	key: string
	label: string
	icon: IconProp
	danger?: boolean
	run: () => void
}

/** The "⋯" menu of an entry. Later: cook. */
function entryActions(entry: MenuEntryResponse): EntryAction[] {
	const actions: EntryAction[] = []
	if (entry.recipeId) {
		actions.push({key: 'cook', label: t('household.cooking.open'), icon: 'utensils', run: () => router.push(cookRoute(entry))})
		actions.push({key: 'servings', label: t('household.menu.changeServings'), icon: 'users', run: () => startEditingServings(entry)})
		actions.push({key: 'add-to-list', label: t('household.menu.addToList'), icon: 'basket-shopping', run: () => {
			addingToList.value = entry
		}})
	}
	actions.push({key: 'move', label: t('household.menu.moveTo'), icon: 'calendar-alt', run: () => {
		moving.value = entry
	}})
	actions.push({key: 'delete',label: t('household.menu.delete'), icon: 'trash-alt', danger: true, run: () => remove(entry.id)})
	return actions
}

function remove(id: number) {
	if (busy.value) return
	deleteEntry.mutate(id)
}

// ---- Changing servings ----

const editingServings = ref(0) // the entry id, 0 for none
// v-model.number gives '' while the field is empty.
const servingsDraft = ref<number | ''>('')
const validServings = computed(() => typeof servingsDraft.value === 'number' && servingsDraft.value >= 1)

function startEditingServings(entry: MenuEntryResponse) {
	editingServings.value = entry.id
	servingsDraft.value = entry.servings > 0 ? entry.servings : ''
}

function stopEditingServings() {
	editingServings.value = 0
	servingsDraft.value = ''
}

function saveServings(entry: MenuEntryResponse) {
	if (busy.value || !validServings.value || typeof servingsDraft.value !== 'number') return
	if (servingsDraft.value === entry.servings) {
		stopEditingServings()
		return
	}
	updateEntry.mutate({id: entry.id, changes: {servings: servingsDraft.value}}, {onSuccess: stopEditingServings})
}

// ---- Adding a recipe's ingredients to a shopping list ----

// The entry whose "Add to list" dialog is open. The dialog adds through the
// shopping sync engine (as the recipe page does), not through the menu's writes.
const addingToList = ref<MenuEntryResponse | null>(null)

// ---- Moving to another day ----

const router = useRouter()
// The entry whose "Move to…" dialog is open.
const moving = ref<MenuEntryResponse | null>(null)

function move(day: string) {
	const entry = moving.value
	if (!entry || busy.value) return
	if (day === entry.day) {
		moving.value = null
		return
	}
	// On failure the mutation shows the error toast and the dialog stays open.
	updateEntry.mutate({id: entry.id, changes: {day}}, {
		onSuccess: () => {
			moving.value = null
			// Moved within the week shown: the list itself shows where it went.
			if (days.value.includes(day)) return
			success(
				{message: t('household.menu.moved', {day: plannedDayLabel(day, locale.value)})},
				[{title: t('household.menu.openMenu'), callback: () => router.push(menuWeekRoute(day, new Date()))}],
			)
		},
	})
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

.entry-note,
.entry-main {
	overflow-wrap: anywhere;
}

.entry-main {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .4rem;
}

.entry-servings {
	color: var(--grey-500);
	font-size: .875rem;
}

.servings-form {
	display: flex;
	gap: .4rem;
}

.servings-input {
	inline-size: 4.5rem;
}

.entry-options {
	flex-shrink: 0;
}

.nothing {
	margin-block-start: .25rem;
	color: var(--grey-500);
	font-style: italic;
}

.add-form {
	margin-block-start: .5rem;
}

.add-row {
	display: flex;
	gap: .4rem;

	.input {
		flex: 1;
	}
}

.recipe-matches {
	margin: .25rem 0 0;
	padding: 0;
	border: 1px solid var(--grey-200);
	border-radius: $radius;
	list-style: none;
}

.recipe-match {
	padding: .4rem .75rem;
	cursor: pointer;

	&:hover,
	&.is-highlighted {
		background: var(--grey-100);
	}
}
</style>
