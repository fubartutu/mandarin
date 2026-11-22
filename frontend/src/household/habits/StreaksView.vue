<template>
	<div class="streaks-page">
		<header class="streaks-header">
			<h1 class="title">
				{{ $t('household.habits.streaks.title') }}
			</h1>
			<RouterLink
				:to="{name: 'habits'}"
				class="button is-secondary"
			>
				{{ $t('household.habits.streaks.back') }}
			</RouterLink>
		</header>

		<p
			v-if="!sync.hasData.value && sync.online.value"
			class="has-text-centered has-text-grey"
		>
			{{ $t('household.home.loading') }}
		</p>
		<p
			v-else-if="habits.length === 0"
			class="has-text-centered has-text-grey"
		>
			{{ $t('household.habits.streaks.none') }}
		</p>

		<template v-else>
			<!-- Each habit's current streak as a bar out of its best. -->
			<section class="card streak-section">
				<h2 class="section-title">
					{{ $t('household.habits.streaks.nowTitle') }}
				</h2>
				<p class="section-help">
					{{ $t('household.habits.streaks.nowHelp') }}
				</p>
				<ul class="gauges">
					<li
						v-for="habit in byStreak"
						:key="habit.id"
						class="gauge"
						:style="colorStyle(habit.color)"
					>
						<span class="gauge-title">{{ habit.title }}</span>
						<span
							class="gauge-track"
							role="img"
							:aria-label="gaugeLabel(habit)"
						>
							<span
								class="gauge-fill"
								:class="{'is-best': isBest(habit), 'is-grace': habit.progress.grace_used}"
								:style="{inlineSize: `${gaugePercent(habit)}%`}"
							/>
						</span>
						<span class="gauge-numbers">
							<Icon
								v-if="habit.progress.streak > 0"
								icon="fire"
								class="flame"
							/>
							<strong>{{ habit.progress.streak }}</strong>
							<span class="has-text-grey"> / {{ habit.progress.best_streak }} {{ unitLabel(habit) }}</span>
							<span
								v-if="isBest(habit)"
								class="best-badge"
							>{{ $t('household.habits.streaks.best') }}</span>
						</span>
					</li>
				</ul>
			</section>

			<!-- Every streak in the last 26 weeks, one lane per habit. -->
			<section class="card streak-section">
				<h2 class="section-title">
					{{ $t('household.habits.streaks.timelineTitle') }}
				</h2>
				<p class="section-help">
					{{ $t('household.habits.streaks.timelineHelp') }}
				</p>
				<p
					v-if="!sync.online.value"
					class="has-text-grey is-italic"
				>
					{{ $t('household.habits.streaks.offline') }}
				</p>
				<div
					v-else
					class="timeline"
				>
					<div class="axis lane-row">
						<span class="lane-title" />
						<span class="lane">
							<span
								v-for="month in months"
								:key="month"
								class="month"
								:style="{insetInlineStart: `${position(month)}%`}"
							>{{ monthName(month) }}</span>
						</span>
					</div>
					<div
						v-for="row in lanes"
						:key="row.habit.id"
						class="lane-row"
						:style="colorStyle(row.habit.color)"
					>
						<span class="lane-title">{{ row.habit.title }}</span>
						<span
							class="lane"
							role="img"
							:aria-label="laneLabel(row)"
						>
							<span
								v-for="month in months"
								:key="month"
								class="gridline"
								:style="{insetInlineStart: `${position(month)}%`}"
							/>
							<span
								v-if="row.loading"
								class="lane-loading"
							/>
							<template
								v-for="run in row.runs"
								:key="run.start"
							>
								<span
									class="run"
									:class="{'is-ongoing': run.ongoing, 'is-clipped': run.span.clipped, 'is-best': run.best}"
									:style="{insetInlineStart: `${run.span.left}%`, inlineSize: `${run.span.width}%`}"
									:title="runTitle(row.habit, run)"
								/>
								<span
									v-for="miss in run.forgivenAt"
									:key="miss"
									class="forgiven"
									:style="{insetInlineStart: `${miss}%`}"
								/>
							</template>
						</span>
					</div>
					<ul class="legend">
						<li><span class="key run" /> {{ $t('household.habits.streaks.legendRun') }}</li>
						<li><span class="key run is-ongoing" /> {{ $t('household.habits.streaks.legendOngoing') }}</li>
						<li><span class="key forgiven" /> {{ $t('household.habits.streaks.legendForgiven') }}</li>
					</ul>
				</div>
			</section>
		</template>
	</div>
</template>

<script setup lang="ts">
import {computed, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQueries} from '@tanstack/vue-query'

import {useTitle} from '@/composables/useTitle'

import {addDays, dayKey, daysBetween, fromKey} from '../menu/week'
import {colorStyle} from './colors'
import {mondayOf, streakUnit} from './grid'
import {habitHistoryQuery, type HabitResponse} from './queries'
import {graceDays, monthStarts, periodEnd, streakRuns, timelineSpan, type StreakRun} from './streaks'
import {useHabitsSync} from './sync/useHabitsSync'
import {habitSections} from './today'

const {t, locale} = useI18n({useScope: 'global'})
useTitle(() => t('household.habits.streaks.title'))

const sync = useHabitsSync()

// Bad habits have no streaks. The rest in the Habits page's order.
const habits = computed(() => habitSections(sync.habits.value, sync.groups.value).flatMap(s => s.habits))
const today = computed(() => habits.value[0]?.progress.today || dayKey(new Date()))

// The timeline shows 26 whole weeks; the history goes back 56 weeks (the
// server allows 400 days), so runs and forgiven misses that began before
// the timeline come out the same as the server's.
const SHOWN_WEEKS = 26
const FETCHED_WEEKS = 56
const shownFrom = computed(() => addDays(mondayOf(today.value), -7 * (SHOWN_WEEKS - 1)))
const fetchedFrom = computed(() => addDays(mondayOf(today.value), -7 * (FETCHED_WEEKS - 1)))

// One history request per habit, in parallel (reads, so SQLite's one
// writer doesn't matter). Cached with the habit grid's other ranges.
const histories = useQueries({
	queries: computed(() => habits.value.map(h => habitHistoryQuery(h.id, fetchedFrom.value, today.value))),
})

// The histories come from the server (online only); after the queue has
// sent ticks, fetch them again so the timeline shows them.
watch(sync.lastSyncedAt, () => {
	for (const q of histories.value) void q.refetch()
})

interface DrawnRun extends StreakRun {
	span: NonNullable<ReturnType<typeof timelineSpan>>
	forgivenAt: number[]
	best: boolean
}

const lanes = computed(() => habits.value.map((habit, i) => {
	const query = histories.value[i]
	const runs = streakRuns(query?.data?.periods ?? [], graceDays(habit.kind))
	const longest = Math.max(0, ...runs.map(r => r.length))
	const drawn: DrawnRun[] = runs.flatMap(run => {
		const span = timelineSpan(shownFrom.value, today.value, run.start, periodEnd(habit.kind, run.end))
		if (!span) return []
		return [{
			...run,
			span,
			forgivenAt: run.forgiven.filter(d => d >= shownFrom.value).map(d => position(d, habit.kind === 'weekly' ? 3.5 : .5)),
			best: run.length === longest && longest >= habit.progress.best_streak,
		}]
	})
	return {habit, runs: drawn, loading: query?.isPending ?? true}
}))

/** Where a day sits on the timeline, in percent; `offset` days to the right. */
function position(day: string, offset = 0) {
	return (daysBetween(shownFrom.value, day) + offset) / (daysBetween(shownFrom.value, today.value) + 1) * 100
}

const months = computed(() => monthStarts(shownFrom.value, today.value))

function monthName(day: string) {
	return fromKey(day).toLocaleDateString(locale.value, {month: 'short'})
}

// ---- Now: current streak out of the best ----

const byStreak = computed(() => [...habits.value].sort((a, b) =>
	b.progress.streak - a.progress.streak || b.progress.best_streak - a.progress.best_streak))

function gaugePercent(h: HabitResponse) {
	const best = h.progress.best_streak
	return best > 0 ? Math.round(h.progress.streak / best * 100) : 0
}

/** Matching the best (and not zero): this streak is the record. */
function isBest(h: HabitResponse) {
	return h.progress.streak > 0 && h.progress.streak >= h.progress.best_streak
}

function unitLabel(h: HabitResponse) {
	return t(`household.habits.streaks.unit.${streakUnit(h.kind)}`, h.progress.best_streak)
}

function gaugeLabel(h: HabitResponse) {
	return t('household.habits.streaks.gaugeLabel', {title: h.title, count: h.progress.streak, best: h.progress.best_streak, unit: unitLabel(h)})
}

// ---- Labels for the timeline ----

function shortDate(day: string) {
	return fromKey(day).toLocaleDateString(locale.value, {day: 'numeric', month: 'short'})
}

function runTitle(h: HabitResponse, run: StreakRun) {
	const unit = streakUnit(h.kind)
	const length = t(`household.habits.streak.${unit}`, {count: run.length}, run.length)
	const end = run.ongoing ? t('household.habits.streaks.now') : shortDate(periodEnd(h.kind, run.end))
	return `${length}: ${shortDate(run.start)} – ${end}`
}

function laneLabel(row: {habit: HabitResponse, runs: DrawnRun[]}) {
	if (row.runs.length === 0) return t('household.habits.streaks.laneEmpty', {title: row.habit.title})
	return `${row.habit.title}: ${row.runs.map(r => runTitle(row.habit, r)).join('; ')}`
}
</script>

<style lang="scss" scoped>
.streaks-page {
	max-inline-size: 50rem;
	margin-inline: auto;
}

.streaks-header {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: space-between;
	gap: .5rem;
	margin-block-end: 1rem;

	.title {
		margin: 0;
	}
}

.streak-section {
	margin-block-end: 1rem;
	padding: 1rem;
}

.section-title {
	margin-block-end: .25rem;
	font-size: 1.1rem;
	font-weight: 700;
}

.section-help {
	margin-block-end: .75rem;
	color: var(--grey-600);
	font-size: .9rem;
}

// ---- Gauges ----

.gauges {
	margin: 0;
	padding: 0;
	list-style: none;
}

.gauge {
	display: grid;
	grid-template-columns: minmax(6rem, 10rem) 1fr auto;
	align-items: center;
	gap: .75rem;
	padding-block: .35rem;
}

.gauge-title,
.lane-title {
	overflow: hidden;
	font-weight: 600;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.gauge-track {
	position: relative;
	display: block;
	overflow: hidden;
	block-size: .75rem;
	border-radius: 999px;
	background: var(--grey-200);
}

.gauge-fill {
	display: block;
	block-size: 100%;
	border-radius: 999px;
	background: var(--habit-color);
	transition: inline-size .4s ease;

	// Alive thanks to a forgiven miss: striped, one more and it ends.
	&.is-grace {
		background: repeating-linear-gradient(45deg, var(--habit-color) 0 6px, color-mix(in srgb, var(--habit-color) 55%, transparent) 6px 12px);
	}
}

.gauge-numbers {
	font-size: .9rem;
	white-space: nowrap;
}

.flame {
	color: var(--habit-color);
}

.best-badge {
	margin-inline-start: .35rem;
	padding: .05rem .4rem;
	border-radius: 999px;
	background: color-mix(in srgb, var(--habit-color) 18%, transparent);
	color: var(--text);
	font-size: .75rem;
	font-weight: 600;
}

// ---- Timeline ----

.lane-row {
	display: grid;
	grid-template-columns: minmax(5rem, 9rem) 1fr;
	align-items: center;
	gap: .75rem;
	padding-block: .3rem;
}

.lane {
	position: relative;
	display: block;
	block-size: 1rem;
	border-radius: 3px;
	background: var(--grey-100);
}

.axis .lane {
	block-size: 1.1rem;
	background: transparent;
}

.month {
	position: absolute;
	color: var(--grey-600);
	font-size: .75rem;
	white-space: nowrap;
}

.gridline {
	position: absolute;
	inset-block: 0;
	inline-size: 1px;
	background: var(--grey-300);
}

.lane-loading {
	position: absolute;
	inset: 0;
	border-radius: 3px;
	background: linear-gradient(90deg, transparent, var(--grey-200), transparent);
	animation: shimmer 1.2s infinite;
}

@keyframes shimmer {
	from { transform: translateX(-100%); }
	to { transform: translateX(100%); }
}

.run {
	position: absolute;
	inset-block: 2px;
	min-inline-size: 3px;
	border-radius: 999px;
	background: color-mix(in srgb, var(--habit-color) 55%, transparent);

	// The streak still going: full colour, with a bright end at today.
	&.is-ongoing {
		background: var(--habit-color);
		box-shadow: 2px 0 0 0 color-mix(in srgb, var(--habit-color) 40%, transparent);
	}

	// The longest one: a ring, so it stands out among faded ones.
	&.is-best {
		outline: 2px solid var(--habit-color);
		outline-offset: 1px;
	}

	// Began before the timeline: no rounded start.
	&.is-clipped {
		border-start-start-radius: 0;
		border-end-start-radius: 0;
	}
}

// A forgiven miss: a notch in the run.
.forgiven {
	position: absolute;
	inset-block: 0;
	inline-size: 3px;
	margin-inline-start: -1px;
	border-radius: 1px;
	background: var(--grey-100);
	box-shadow: 0 0 0 1px var(--grey-400);
}

.legend {
	display: flex;
	flex-wrap: wrap;
	gap: 1rem;
	margin: .75rem 0 0;
	padding: 0;
	color: var(--grey-600);
	font-size: .8rem;
	list-style: none;

	--habit-color: var(--primary);

	li {
		display: flex;
		align-items: center;
		gap: .35rem;
	}

	.key {
		position: relative;
		inset: auto;
		display: inline-block;
		inline-size: 1.5rem;
		block-size: .6rem;
	}

	.key.forgiven {
		inline-size: 3px;
		block-size: .9rem;
	}
}

// Narrow phones: names above the bars.
@media (width <= 32rem) {
	.gauge {
		grid-template-columns: 1fr auto;

		.gauge-track {
			grid-column: 1 / -1;
			grid-row: 2;
		}
	}

	.lane-row {
		grid-template-columns: 1fr;
		gap: .2rem;
	}

	.axis .lane-title {
		display: none;
	}
}
</style>
