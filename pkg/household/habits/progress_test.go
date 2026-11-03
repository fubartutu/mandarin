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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 2026-11-02 is a Monday; the tests use the weeks around it.
func day(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := parseCivil(s)
	require.NoError(t, err)
	return d
}

// ticks builds a history with one tick on each day given.
func ticks(days ...string) *history {
	h := newHistory()
	for _, d := range days {
		h.counts[d]++
	}
	return h
}

func daily(weekdays string) *Habit {
	return &Habit{Kind: KindDaily, Weekdays: weekdays, Goal: 1}
}

func TestWeekStart(t *testing.T) {
	assert.Equal(t, "2026-11-02", weekStart(day(t, "2026-11-02")).Format(dayFormat)) // Monday
	assert.Equal(t, "2026-11-02", weekStart(day(t, "2026-11-08")).Format(dayFormat)) // Sunday
	assert.Equal(t, "2026-10-26", weekStart(day(t, "2026-11-01")).Format(dayFormat)) // the Sunday before
}

func TestProgressDailyEveryDay(t *testing.T) {
	h := daily(everyDay)
	start := day(t, "2026-10-26")

	t.Run("today not done yet doesn't break the streak", func(t *testing.T) {
		hist := ticks("2026-10-30", "2026-10-31", "2026-11-01", "2026-11-02")
		pr := computeProgress(h, hist, start, day(t, "2026-11-03"))
		assert.Equal(t, 4, pr.Streak)
		assert.True(t, pr.DueToday)
		assert.False(t, pr.DoneToday)
		// 26–29 Oct missed, 30 Oct–2 Nov done, today (pending) left out.
		assert.Equal(t, 4, pr.Done)
		assert.Equal(t, 8, pr.Due)
	})

	t.Run("today done counts", func(t *testing.T) {
		hist := ticks("2026-11-01", "2026-11-02", "2026-11-03")
		pr := computeProgress(h, hist, day(t, "2026-11-01"), day(t, "2026-11-03"))
		assert.Equal(t, 3, pr.Streak)
		assert.Equal(t, 3, pr.BestStreak)
		assert.True(t, pr.DoneToday)
		assert.False(t, pr.DueToday)
		assert.Equal(t, 3, pr.Done)
		assert.Equal(t, 3, pr.Due)
	})

	t.Run("one miss after a done day is forgiven", func(t *testing.T) {
		// Done 26–29, missed 30, done 31 Oct–2 Nov.
		hist := ticks("2026-10-26", "2026-10-27", "2026-10-28", "2026-10-29", "2026-10-31", "2026-11-01", "2026-11-02")
		pr := computeProgress(h, hist, start, day(t, "2026-11-02"))
		assert.Equal(t, 7, pr.Streak)
		assert.True(t, pr.GraceUsed)
	})

	t.Run("two misses in a row end it", func(t *testing.T) {
		hist := ticks("2026-10-26", "2026-10-27", "2026-10-30", "2026-10-31")
		pr := computeProgress(h, hist, start, day(t, "2026-10-31"))
		assert.Equal(t, 2, pr.Streak)
		assert.Equal(t, 2, pr.BestStreak)
		assert.False(t, pr.GraceUsed)
	})

	t.Run("a second miss within 7 days ends it", func(t *testing.T) {
		// Missed 28 Oct (forgiven) and 31 Oct.
		hist := ticks("2026-10-26", "2026-10-27", "2026-10-29", "2026-10-30", "2026-11-01", "2026-11-02")
		pr := computeProgress(h, hist, start, day(t, "2026-11-02"))
		assert.Equal(t, 2, pr.Streak)
		assert.Equal(t, 4, pr.BestStreak)
	})

	t.Run("a miss 7 days after the last forgiven one is forgiven again", func(t *testing.T) {
		hist := newHistory()
		for d := day(t, "2026-10-01"); !d.After(day(t, "2026-10-20")); d = addDays(d, 1) {
			if s := d.Format(dayFormat); s != "2026-10-05" && s != "2026-10-12" {
				hist.counts[s] = 1
			}
		}
		pr := computeProgress(h, hist, day(t, "2026-10-01"), day(t, "2026-10-20"))
		assert.Equal(t, 18, pr.Streak)
		assert.False(t, pr.GraceUsed, "the last forgiven miss is 8 days ago")
	})

	t.Run("a miss can't start a streak", func(t *testing.T) {
		pr := computeProgress(h, ticks("2026-11-02"), day(t, "2026-11-01"), day(t, "2026-11-02"))
		assert.Equal(t, 1, pr.Streak)
	})

	t.Run("yesterday missed: streak kept on grace", func(t *testing.T) {
		hist := ticks("2026-10-30", "2026-10-31")
		pr := computeProgress(h, hist, day(t, "2026-10-30"), day(t, "2026-11-02"))
		assert.Equal(t, 2, pr.Streak)
		assert.True(t, pr.GraceUsed)
		assert.True(t, pr.DueToday)
	})
}

func TestProgressScheduledDays(t *testing.T) {
	// Mon/Wed/Fri. Friday 30 Oct, then Monday 2 Nov: the weekend is rest.
	h := daily("1010100")
	hist := ticks("2026-10-26", "2026-10-28", "2026-10-30", "2026-11-02")
	pr := computeProgress(h, hist, day(t, "2026-10-26"), day(t, "2026-11-03"))
	assert.Equal(t, 4, pr.Streak)
	assert.Equal(t, 4, pr.Due)
	assert.False(t, pr.DueToday, "Tuesday is a rest day")
	assert.False(t, pr.GraceUsed)
}

func TestProgressSkipAndPause(t *testing.T) {
	h := daily(everyDay)

	t.Run("skips are neither done nor missed", func(t *testing.T) {
		hist := ticks("2026-10-26", "2026-10-29", "2026-10-30")
		hist.skips["2026-10-27"] = true
		hist.skips["2026-10-28"] = true
		pr := computeProgress(h, hist, day(t, "2026-10-26"), day(t, "2026-10-30"))
		assert.Equal(t, 3, pr.Streak)
		assert.Equal(t, 3, pr.Due)
		assert.False(t, pr.GraceUsed)
	})

	t.Run("skipped today", func(t *testing.T) {
		hist := ticks("2026-11-01")
		hist.skips["2026-11-02"] = true
		pr := computeProgress(h, hist, day(t, "2026-11-01"), day(t, "2026-11-02"))
		assert.True(t, pr.Skipped)
		assert.False(t, pr.DueToday)
		assert.Equal(t, 1, pr.Streak)
	})

	t.Run("paused stretch", func(t *testing.T) {
		hist := ticks("2026-10-20", "2026-10-21", "2026-11-01")
		hist.pauses = []pauseSpan{{from: "2026-10-22", until: "2026-10-31"}}
		pr := computeProgress(h, hist, day(t, "2026-10-20"), day(t, "2026-11-01"))
		assert.Equal(t, 3, pr.Streak)
		assert.Equal(t, 3, pr.Due)
	})

	t.Run("still paused", func(t *testing.T) {
		hist := ticks("2026-10-31")
		hist.pauses = []pauseSpan{{from: "2026-11-01"}}
		pr := computeProgress(h, hist, day(t, "2026-10-31"), day(t, "2026-11-03"))
		assert.False(t, pr.DueToday)
		assert.Equal(t, 1, pr.Streak)
		assert.True(t, hist.paused("2026-11-03"))
	})
}

func TestProgressDailyCount(t *testing.T) {
	h := &Habit{Kind: KindDailyCount, Weekdays: everyDay, Goal: 3}
	hist := newHistory()
	hist.counts["2026-10-31"] = 3
	hist.counts["2026-11-01"] = 4
	hist.counts["2026-11-02"] = 2 // under the goal
	hist.counts["2026-11-03"] = 1

	pr := computeProgress(h, hist, day(t, "2026-10-31"), day(t, "2026-11-03"))
	assert.Equal(t, 1, pr.TodayCount)
	assert.Equal(t, 3, pr.Goal)
	assert.True(t, pr.DueToday)
	assert.Equal(t, 2, pr.Streak, "a partial day is a forgiven miss")
	assert.True(t, pr.GraceUsed)
	assert.Equal(t, 2, pr.Done)
	assert.Equal(t, 3, pr.Due)
	assert.Equal(t, 3, pr.ThisWeek, "ticks are counted, not days")
	assert.Equal(t, 7, pr.LastWeek)
}

func TestProgressWeekly(t *testing.T) {
	h := &Habit{Kind: KindWeekly, Weekdays: everyDay, Goal: 2}

	t.Run("weeks meeting the goal in a row", func(t *testing.T) {
		hist := ticks(
			"2026-10-12", "2026-10-14", // week of 12 Oct: 2
			"2026-10-19", "2026-10-25", // week of 19 Oct: 2
			"2026-10-27", "2026-10-28", "2026-10-29", // week of 26 Oct: 3
			"2026-11-03", // this week: 1 so far
		)
		pr := computeProgress(h, hist, day(t, "2026-10-12"), day(t, "2026-11-04"))
		assert.Equal(t, 3, pr.Streak)
		assert.Equal(t, 1, pr.TodayCount)
		assert.True(t, pr.DueToday, "due every day until the week's goal is reached")
		assert.Equal(t, 3, pr.Done)
		assert.Equal(t, 3, pr.Due)
		assert.Equal(t, 1, pr.ThisWeek)
		assert.Equal(t, 3, pr.LastWeek)
		assert.Len(t, periods(h, hist, day(t, "2026-10-12"), day(t, "2026-11-04")), 4)
	})

	t.Run("the week the habit was made isn't held against it", func(t *testing.T) {
		hist := ticks("2026-10-30", "2026-11-02", "2026-11-03")
		pr := computeProgress(h, hist, day(t, "2026-10-29"), day(t, "2026-11-03"))
		assert.Equal(t, 1, pr.Streak)
		assert.Equal(t, 1, pr.Due)
		assert.True(t, pr.DoneToday)
	})

	t.Run("a skipped week", func(t *testing.T) {
		hist := ticks("2026-10-12", "2026-10-13", "2026-10-26", "2026-10-27")
		hist.skips["2026-10-19"] = true
		pr := computeProgress(h, hist, day(t, "2026-10-12"), day(t, "2026-11-01"))
		assert.Equal(t, 2, pr.Streak)
	})

	t.Run("a missed week is forgiven once per 4 weeks", func(t *testing.T) {
		hist := ticks("2026-10-05", "2026-10-06", "2026-10-12", "2026-10-13", "2026-10-26", "2026-10-27")
		pr := computeProgress(h, hist, day(t, "2026-10-05"), day(t, "2026-11-01"))
		assert.Equal(t, 3, pr.Streak)
		assert.True(t, pr.GraceUsed)
	})
}

func TestProgressBadHabit(t *testing.T) {
	h := &Habit{Kind: KindBad, Weekdays: everyDay, Goal: 1}
	hist := newHistory()
	hist.counts["2026-10-27"] = 2
	hist.counts["2026-10-30"] = 3
	hist.counts["2026-11-02"] = 1
	hist.counts["2026-11-03"] = 2

	pr := computeProgress(h, hist, day(t, "2026-10-27"), day(t, "2026-11-03"))
	assert.Equal(t, 2, pr.TodayCount)
	assert.Equal(t, 3, pr.ThisWeek)
	assert.Equal(t, 5, pr.LastWeek)
	assert.Zero(t, pr.Streak)
	assert.Zero(t, pr.Due)
	assert.Zero(t, pr.Goal)
	assert.False(t, pr.DueToday)
}

func TestGridPeriods(t *testing.T) {
	t.Run("days before the start", func(t *testing.T) {
		h := daily("1111100") // weekdays only
		hist := ticks("2026-10-30", "2026-11-02")
		ps := gridPeriods(h, hist, day(t, "2026-10-30"), day(t, "2026-11-03"), day(t, "2026-10-28"), day(t, "2026-11-10"))
		got := []string{}
		for _, p := range ps {
			got = append(got, p.Start+" "+p.Status)
		}
		assert.Equal(t, []string{
			"2026-10-28 before",
			"2026-10-29 before",
			"2026-10-30 done",
			"2026-10-31 rest",
			"2026-11-01 rest",
			"2026-11-02 done",
			"2026-11-03 pending",
		}, got)
	})

	t.Run("weeks", func(t *testing.T) {
		h := &Habit{Kind: KindWeekly, Weekdays: everyDay, Goal: 1}
		hist := ticks("2026-10-20", "2026-11-03")
		ps := gridPeriods(h, hist, day(t, "2026-10-21"), day(t, "2026-11-04"), day(t, "2026-10-14"), day(t, "2026-11-04"))
		got := []string{}
		for _, p := range ps {
			got = append(got, p.Start+" "+p.Status)
		}
		// The week of 19 Oct is the start week and was done; the week of
		// 26 Oct was missed; this week is done.
		assert.Equal(t, []string{
			"2026-10-12 before",
			"2026-10-19 done",
			"2026-10-26 missed",
			"2026-11-02 done",
		}, got)
	})
}
