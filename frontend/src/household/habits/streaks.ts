// The Streaks page's timeline: where each streak started and ended. Pure,
// tested in streaks.test.ts.
//
// The numbers (current and best streak) come from the server's progress;
// this only finds the runs to draw. It follows the same rules as streaks()
// in pkg/household/habits/progress.go, so keep the two in step: skipped,
// paused, rest and pending periods pass without breaking a run, and one
// miss is forgiven when the run is going, the period before it that counted
// was done, and no other miss was forgiven in the last 7 days (28 for weekly
// habits). The server walks from the habit's start and this from the start
// of the history it fetched, so ask for more history than you show.

import {addDays, daysBetween} from '../menu/week'
import type {HabitKind, PeriodResponse} from './queries'

export interface StreakRun {
	/** First and last done period (a week's Monday for weekly habits). */
	start: string
	end: string
	/** Done periods in the run. */
	length: number
	/** Missed periods inside the run that were forgiven. */
	forgiven: string[]
	/** Still going today. */
	ongoing: boolean
}

/** The days a missed period can be forgiven within, as on the server. */
export function graceDays(kind: HabitKind): number {
	return kind === 'weekly' ? 28 : 7
}

const counts = (status: string) => status === 'done' || status === 'partial' || status === 'missed'

/** The habit's streaks in `periods` (oldest first), oldest first. */
export function streakRuns(periods: PeriodResponse[], grace: number): StreakRun[] {
	const runs: StreakRun[] = []
	let run: StreakRun | null = null
	let prevDone = false
	let lastForgiven = ''

	for (const p of periods) {
		if (!counts(p.status)) continue
		if (p.status === 'done') {
			run ??= {start: p.start, end: p.start, length: 0, forgiven: [], ongoing: false}
			run.length++
			run.end = p.start
			prevDone = true
			continue
		}
		// YYYY-MM-DD strings sort like dates, so < compares days.
		const withinGrace = lastForgiven !== '' && p.start < addDays(lastForgiven, grace)
		if (run && prevDone && !withinGrace) {
			run.forgiven.push(p.start)
			lastForgiven = p.start
			prevDone = false
			continue
		}
		if (run) runs.push(closed(run))
		run = null
		lastForgiven = ''
		prevDone = false
	}
	if (run) runs.push({...run, ongoing: true})
	return runs
}

/** A run that ended: a forgiven miss after its last done period was the end, not part of it. */
function closed(run: StreakRun): StreakRun {
	return {...run, forgiven: run.forgiven.filter(day => day < run.end)}
}

/**
 * Where a span of days sits on a timeline from `from` to `to`, as
 * percentages for CSS (`left`, `width`). Clipped to the timeline; null
 * when it's entirely outside. `clipped` says the start was cut off.
 */
export function timelineSpan(from: string, to: string, start: string, end: string): {left: number, width: number, clipped: boolean} | null {
	const total = daysBetween(from, to) + 1
	const first = Math.max(daysBetween(from, start), 0)
	const last = Math.min(daysBetween(from, end), total - 1)
	if (last < 0 || first > total - 1) return null
	return {
		left: first / total * 100,
		width: (last - first + 1) / total * 100,
		clipped: start < from,
	}
}

/** The last day a period covers: the day itself, or a week's Sunday. */
export function periodEnd(kind: HabitKind, start: string): string {
	return kind === 'weekly' ? addDays(start, 6) : start
}

/** The first of each month on the timeline, for the axis labels. */
export function monthStarts(from: string, to: string): string[] {
	const out: string[] = []
	let day = `${from.slice(0, 7)}-01`
	if (day < from) day = nextMonth(day)
	for (; day <= to; day = nextMonth(day)) out.push(day)
	return out
}

function nextMonth(day: string): string {
	const [y, m] = day.split('-').map(Number)
	return m === 12 ? `${y + 1}-01-01` : `${y}-${String(m + 1).padStart(2, '0')}-01`
}
