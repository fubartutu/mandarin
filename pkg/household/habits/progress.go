// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package habits

import (
	"time"
)

// This file works out how a habit is going from its history: the status of
// each day (or week), "done X of the last Y", and the streak. It only does
// date arithmetic on plain values, so it is tested without a database.
//
// Days are calendar dates with no time zone: a time.Time at midnight UTC.
// Adding 24 hours to one always gives the next date, which isn't true of
// local times on daylight-saving days.

const (
	// recentDays and recentWeeks are the window of "done X of the last Y".
	recentDays  = 30
	recentWeeks = 12
	// A missed day is forgiven once per 7 days (a missed week once per 4
	// weeks), as long as the period before it was done. Both are in days.
	graceDays     = 7
	graceWeekDays = 4 * 7
)

// Period statuses, as shown on the habit's grid.
const (
	StatusDone    = "done"    // reached the goal
	StatusPartial = "partial" // some, but under the goal (counts as missed)
	StatusMissed  = "missed"  // due and not done
	StatusSkipped = "skipped" // skipped on purpose: neither done nor missed
	StatusPaused  = "paused"  // the habit was paused
	StatusRest    = "rest"    // not one of the habit's days
	StatusPending = "pending" // today (or this week), not done yet
	StatusBefore  = "before"  // before the habit existed
	StatusCount   = "count"   // a bad habit's day: only the count matters
)

// Period is one day (or, for weekly habits, one week) of a habit's grid.
type Period struct {
	Start  string `json:"start" format:"date" doc:"The day, or the Monday a week starts on, as YYYY-MM-DD."`
	Status string `json:"status" enum:"done,partial,missed,skipped,paused,rest,pending,before,count" doc:"done, partial (under the goal), missed, skipped, paused, rest (not one of the habit's days), pending (today or this week, not done yet), before (the habit didn't exist yet), or count (a bad habit's day)."`
	Count  int    `json:"count" doc:"How many times it was ticked (or, for a bad habit, happened) in this period."`
}

// Progress is how a habit is going, as of today.
type Progress struct {
	Today      string `json:"today" format:"date" doc:"The day this was worked out for, in the caller's time zone."`
	TodayCount int    `json:"today_count" doc:"Ticks today (this week for weekly habits)."`
	Goal       int    `json:"goal" doc:"The habit's goal for today (this week for weekly habits)."`
	DueToday   bool   `json:"due_today" doc:"Whether today is one of the habit's days and it isn't paused or skipped. Weekly habits are due every day until their goal is reached."`
	DoneToday  bool   `json:"done_today" doc:"Whether today's (this week's) goal is reached."`
	Skipped    bool   `json:"skipped" doc:"Whether today (this week) is skipped."`
	// TodayEntries is filled in from the database, not worked out here.
	TodayEntries []string `json:"today_entries" doc:"The uids of today's ticks (this week's for weekly habits), oldest first. Delete the last one to take a tick off."`
	Done         int      `json:"done" doc:"Days (weeks) done in the recent window. Not used for bad habits."`
	Due          int      `json:"due" doc:"Days (weeks) that were due in the recent window: the last 30 days or 12 weeks, leaving out skipped, paused and rest days, and today unless it's done. Not used for bad habits."`
	Streak       int      `json:"streak" doc:"Days (weeks) done in a row, counting only due ones. One missed day per 7 (one week per 4) is forgiven when the one before it was done. 0 for bad habits."`
	BestStreak   int      `json:"best_streak" doc:"The longest streak so far, counted the same way."`
	GraceUsed    bool     `json:"grace_used" doc:"Whether the current streak is only alive because a recent miss was forgiven."`
	ThisWeek     int      `json:"this_week" doc:"Ticks (or, for a bad habit, times it happened) this week, Monday to today."`
	LastWeek     int      `json:"last_week" doc:"The same for all of last week."`
}

// history is what has been recorded for one habit.
type history struct {
	counts map[string]int  // day → ticks that day
	skips  map[string]bool // skipped day; for weekly habits the week's Monday
	pauses []pauseSpan
}

// pauseSpan is a stretch the habit was paused, both ends included. An open
// pause (still paused) has until == "".
type pauseSpan struct {
	from, until string
}

func newHistory() *history {
	return &history{counts: map[string]int{}, skips: map[string]bool{}}
}

func (h *history) paused(day string) bool {
	for _, p := range h.pauses {
		if p.from <= day && (p.until == "" || day <= p.until) {
			return true
		}
	}
	return false
}

// pausedDuring reports whether any day from first to last was paused.
func (h *history) pausedDuring(first, last string) bool {
	for _, p := range h.pauses {
		if p.from <= last && (p.until == "" || first <= p.until) {
			return true
		}
	}
	return false
}

func civil(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func parseCivil(day string) (time.Time, error) {
	return time.Parse(dayFormat, day)
}

func addDays(d time.Time, n int) time.Time {
	return d.AddDate(0, 0, n)
}

// weekStart is the Monday of d's week. Weeks run Monday to Sunday for the
// whole household, so a shared weekly habit counts the same for everyone.
func weekStart(d time.Time) time.Time {
	// time.Weekday has Sunday = 0; this makes Monday 0 and Sunday 6.
	return addDays(d, -((int(d.Weekday()) + 6) % 7))
}

// scheduledOn reports whether a daily habit is due on d's weekday.
// weekdays has one "1" or "0" per day, Monday first.
func scheduledOn(weekdays string, d time.Time) bool {
	i := (int(d.Weekday()) + 6) % 7
	return i < len(weekdays) && weekdays[i] == '1'
}

// period is one step of the walk over a habit's history.
type period struct {
	start  time.Time
	status string
	count  int
}

// counts reports whether a status takes part in streaks and "X of Y".
func counts(status string) bool {
	return status == StatusDone || status == StatusPartial || status == StatusMissed
}

// periods returns the habit's days (weeks) from the week or day of start up
// to today, with their status. Bad habits get StatusCount on every day.
func periods(h *Habit, hist *history, start, today time.Time) []period {
	out := []period{}
	if h.Kind == KindWeekly {
		for w := weekStart(start); !w.After(today); w = addDays(w, 7) {
			out = append(out, weekPeriod(h, hist, w, start, today))
		}
		return out
	}
	for d := start; !d.After(today); d = addDays(d, 1) {
		out = append(out, dayPeriod(h, hist, d, today))
	}
	return out
}

func dayPeriod(h *Habit, hist *history, d, today time.Time) period {
	day := d.Format(dayFormat)
	p := period{start: d, count: hist.counts[day]}
	goal := max(h.Goal, 1)
	switch {
	case h.Kind == KindBad:
		p.status = StatusCount
	case hist.paused(day):
		p.status = StatusPaused
	case hist.skips[day]:
		p.status = StatusSkipped
	case !scheduledOn(h.Weekdays, d):
		// Ticks on a rest day show on the grid but don't change the numbers.
		p.status = StatusRest
	case p.count >= goal:
		p.status = StatusDone
	case d.Equal(today):
		p.status = StatusPending
	case p.count > 0:
		p.status = StatusPartial
	default:
		p.status = StatusMissed
	}
	return p
}

func weekPeriod(h *Habit, hist *history, w, start, today time.Time) period {
	p := period{start: w}
	last := addDays(w, 6)
	for d := w; !d.After(last); d = addDays(d, 1) {
		p.count += hist.counts[d.Format(dayFormat)]
	}
	thisWeek := !today.Before(w) && !today.After(last)
	switch {
	case hist.pausedDuring(w.Format(dayFormat), last.Format(dayFormat)):
		// A pause of any part of the week covers the whole week: there was
		// less time to reach the goal.
		p.status = StatusPaused
	case hist.skips[w.Format(dayFormat)]:
		p.status = StatusSkipped
	case p.count >= max(h.Goal, 1):
		p.status = StatusDone
	case thisWeek:
		p.status = StatusPending
	case start.After(w):
		// The habit was made partway through this week: it isn't held to it.
		p.status = StatusBefore
	case p.count > 0:
		p.status = StatusPartial
	default:
		p.status = StatusMissed
	}
	return p
}

// streaks walks the periods oldest first and returns the current streak,
// the best one and the start of the current streak's latest forgiven miss
// (zero if none).
//
// A miss is forgiven when the streak is already going, the last period
// that counted was done (so two misses in a row end it), and no other miss
// was forgiven in the last graceWindow days. Skipped, paused, rest and
// pending periods are passed over without breaking anything.
func streaks(ps []period, graceWindow int) (current, best int, lastForgiven time.Time) {
	run := 0
	prevDone := false
	for _, p := range ps {
		if !counts(p.status) {
			continue
		}
		if p.status == StatusDone {
			run++
			prevDone = true
			best = max(best, run)
			continue
		}
		withinGrace := !lastForgiven.IsZero() && p.start.Before(addDays(lastForgiven, graceWindow))
		if run > 0 && prevDone && !withinGrace {
			lastForgiven = p.start
			prevDone = false
			continue
		}
		run = 0
		lastForgiven = time.Time{}
		prevDone = false
	}
	return run, best, lastForgiven
}

// computeProgress works out a habit's progress as of today. start is the
// first day the habit counts from.
func computeProgress(h *Habit, hist *history, start, today time.Time) *Progress {
	if start.After(today) {
		start = today
	}
	pr := &Progress{Today: today.Format(dayFormat), Goal: max(h.Goal, 1)}

	monday := weekStart(today)
	for d := monday; !d.After(today); d = addDays(d, 1) {
		pr.ThisWeek += hist.counts[d.Format(dayFormat)]
	}
	for d := addDays(monday, -7); d.Before(monday); d = addDays(d, 1) {
		pr.LastWeek += hist.counts[d.Format(dayFormat)]
	}

	if h.Kind == KindBad {
		pr.TodayCount = hist.counts[pr.Today]
		pr.Goal = 0
		return pr
	}

	ps := periods(h, hist, start, today)
	current := ps[len(ps)-1]

	pr.TodayCount = current.count
	pr.DoneToday = current.status == StatusDone
	pr.Skipped = current.status == StatusSkipped
	pr.DueToday = current.status == StatusPending

	window, grace := recentDays, graceDays
	if h.Kind == KindWeekly {
		window, grace = recentWeeks, graceWeekDays
	}
	for _, p := range ps[max(len(ps)-window, 0):] {
		if counts(p.status) {
			pr.Due++
			if p.status == StatusDone {
				pr.Done++
			}
		}
	}
	var lastForgiven time.Time
	pr.Streak, pr.BestStreak, lastForgiven = streaks(ps, grace)
	// The grace is "used" while another miss now would end the streak.
	pr.GraceUsed = pr.Streak > 0 && !lastForgiven.IsZero() && today.Before(addDays(lastForgiven, grace))
	return pr
}

// gridPeriods returns the periods from first to last (both included, clipped
// to today) for the habit's grid. Periods before the habit's start are
// StatusBefore, with any ticks recorded on them.
func gridPeriods(h *Habit, hist *history, start, today, first, last time.Time) []*Period {
	if last.After(today) {
		last = today
	}
	if start.After(today) {
		start = today
	}
	// periods begins at start's day, or for weekly habits at its Monday.
	step, firstPeriod, firstTracked := 1, first, start
	if h.Kind == KindWeekly {
		step, firstPeriod, firstTracked = 7, weekStart(first), weekStart(start)
	}
	out := []*Period{}
	add := func(p period) {
		if !p.start.After(last) && !addDays(p.start, step-1).Before(first) {
			out = append(out, &Period{Start: p.start.Format(dayFormat), Status: p.status, Count: p.count})
		}
	}
	for d := firstPeriod; d.Before(firstTracked) && !d.After(last); d = addDays(d, step) {
		p := period{start: d, status: StatusBefore}
		for i := range step {
			p.count += hist.counts[addDays(d, i).Format(dayFormat)]
		}
		add(p)
	}
	for _, p := range periods(h, hist, start, today) {
		add(p)
	}
	return out
}
