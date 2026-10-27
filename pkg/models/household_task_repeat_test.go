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

package models

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadRepeatFixtures loads Vikunja's fixtures and empties task_repeats,
// which has no fixture file in Vikunja's list (setup_tests.go), so rows
// from an earlier test would otherwise stay.
func loadRepeatFixtures(t *testing.T) {
	t.Helper()
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	_, err := s.Where("1 = 1").Delete(&TaskRepeat{})
	require.NoError(t, err)
	require.NoError(t, s.Commit())
}

func TestTaskRepeatEnd(t *testing.T) {
	u := &user.User{ID: 1}

	// Fixture task 28 repeats hourly and belongs to user 1.
	setEnd := func(t *testing.T, re *TaskRepeat) {
		s := db.NewSession()
		defer s.Close()
		re.TaskID = 28
		require.NoError(t, SetTaskRepeat(s, re))
		require.NoError(t, s.Commit())
	}
	complete := func(t *testing.T, task *Task, fields []string) {
		s := db.NewSession()
		defer s.Close()
		task.ID = 28
		task.Done = true
		require.NoError(t, task.updateSingleTask(s, u, fields))
		require.NoError(t, s.Commit())
	}
	read := func(t *testing.T) (*Task, *TaskRepeat) {
		s := db.NewSession()
		defer s.Close()
		task := &Task{ID: 28}
		require.NoError(t, task.ReadOne(s, u))
		re, err := GetTaskRepeat(s, 28)
		require.NoError(t, err)
		return task, re
	}

	t.Run("a count above one reopens the task and counts down", func(t *testing.T) {
		loadRepeatFixtures(t)
		setEnd(t, &TaskRepeat{RepeatsLeft: 3})
		complete(t, &Task{RepeatAfter: 3600}, nil)

		task, re := read(t)
		assert.False(t, task.Done)
		assert.Equal(t, int64(3600), task.RepeatAfter)
		assert.Equal(t, 2, re.RepeatsLeft)
	})
	t.Run("the last occurrence stays done and stops repeating", func(t *testing.T) {
		loadRepeatFixtures(t)
		setEnd(t, &TaskRepeat{RepeatsLeft: 1})
		complete(t, &Task{RepeatAfter: 3600}, nil)

		task, re := read(t)
		assert.True(t, task.Done)
		assert.Equal(t, int64(0), task.RepeatAfter)
		assert.True(t, task.DueDate.Before(time.Now()), "the due date is not moved on")
		assert.True(t, re.IsEmpty(), "the end is removed with the repeat")
	})
	t.Run("an end date before the next occurrence ends the series", func(t *testing.T) {
		loadRepeatFixtures(t)
		until := time.Now() // the next occurrence is always after now
		setEnd(t, &TaskRepeat{Until: &until})
		complete(t, &Task{RepeatAfter: 3600}, nil)

		task, _ := read(t)
		assert.True(t, task.Done)
		assert.Equal(t, int64(0), task.RepeatAfter)
	})
	t.Run("an end date after the next occurrence keeps repeating", func(t *testing.T) {
		loadRepeatFixtures(t)
		until := time.Now().Add(48 * time.Hour)
		setEnd(t, &TaskRepeat{Until: &until})
		complete(t, &Task{RepeatAfter: 3600}, nil)

		task, re := read(t)
		assert.False(t, task.Done)
		require.NotNil(t, re.Until)
		assert.WithinDuration(t, until, *re.Until, time.Second)
	})
	t.Run("ends the series when only done is sent", func(t *testing.T) {
		loadRepeatFixtures(t)
		setEnd(t, &TaskRepeat{RepeatsLeft: 1})
		complete(t, &Task{}, []string{"done"})

		task, _ := read(t)
		assert.True(t, task.Done)
		assert.Equal(t, int64(0), task.RepeatAfter)
	})
	t.Run("removing the repeat removes the end", func(t *testing.T) {
		loadRepeatFixtures(t)
		setEnd(t, &TaskRepeat{RepeatsLeft: 3})
		s := db.NewSession()
		defer s.Close()
		task := &Task{ID: 28, RepeatAfter: 0}
		require.NoError(t, task.updateSingleTask(s, u, []string{"repeat_after"}))
		require.NoError(t, s.Commit())
		db.AssertMissing(t, "task_repeats", map[string]any{"task_id": 28})
	})
	t.Run("an empty end is deleted, a bad count refused", func(t *testing.T) {
		loadRepeatFixtures(t)
		setEnd(t, &TaskRepeat{RepeatsLeft: 4})
		setEnd(t, &TaskRepeat{})
		db.AssertMissing(t, "task_repeats", map[string]any{"task_id": 28})

		s := db.NewSession()
		defer s.Close()
		err := SetTaskRepeat(s, &TaskRepeat{TaskID: 28, RepeatsLeft: -1})
		assert.ErrorAs(t, err, &ErrInvalidTaskRepeat{})
	})
}

func TestNextRuleDate(t *testing.T) {
	tz := config.GetTimeZone()
	at := func(y int, m time.Month, d, h int) time.Time { return time.Date(y, m, d, h, 0, 0, 0, tz) }
	// Monday 26 October 2026, 17:00.
	monday := at(2026, time.October, 26, 17)

	tests := []struct {
		name     string
		rule     string
		ref, now time.Time
		want     time.Time
	}{
		{"next weekday in the set", "FREQ=WEEKLY;BYDAY=MO,TH", monday, monday.Add(time.Hour), at(2026, time.October, 29, 17)},
		{"done early counts from the due date", "FREQ=WEEKLY;BYDAY=MO,TH", monday, monday.Add(-48 * time.Hour), at(2026, time.October, 29, 17)},
		{"every other week", "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO", monday, monday.Add(time.Hour), at(2026, time.November, 9, 17)},
		{"weekdays skip the weekend", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", at(2026, time.October, 30, 17), at(2026, time.October, 30, 18), at(2026, time.November, 2, 17)},
		{"2nd Tuesday", "FREQ=MONTHLY;BYDAY=2TU", at(2026, time.October, 13, 9), at(2026, time.October, 13, 10), at(2026, time.November, 10, 9)},
		{"last Friday", "FREQ=MONTHLY;BYDAY=-1FR", at(2026, time.October, 30, 9), at(2026, time.October, 30, 10), at(2026, time.November, 27, 9)},
		{"last day of the month", "FREQ=MONTHLY;BYMONTHDAY=-1", at(2026, time.October, 31, 9), at(2026, time.October, 31, 10), at(2026, time.November, 30, 9)},
		{"every 2 months keeps the day", "FREQ=MONTHLY;INTERVAL=2", at(2026, time.October, 15, 9), at(2026, time.October, 15, 10), at(2026, time.December, 15, 9)},
		{"the 31st skips short months", "FREQ=MONTHLY;BYMONTHDAY=31", at(2026, time.October, 31, 9), at(2026, time.October, 31, 10), at(2026, time.December, 31, 9)},
		{"yearly", "FREQ=YEARLY", at(2026, time.November, 5, 9), at(2026, time.November, 5, 10), at(2027, time.November, 5, 9)},
		{"29 February waits for a leap year", "FREQ=YEARLY", at(2028, time.February, 29, 9), at(2028, time.February, 29, 10), at(2032, time.February, 29, 9)},
		{"weeks overdue lands after now", "FREQ=WEEKLY;BYDAY=TU", at(2026, time.September, 1, 17), at(2026, time.October, 27, 10), at(2026, time.October, 27, 17)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nextRuleDate(tt.rule, tt.ref, tt.now)
			require.NoError(t, err)
			assert.True(t, tt.want.Equal(got), "want %s, got %s", tt.want, got)
		})
	}
}

func TestTaskRepeatRule(t *testing.T) {
	u := &user.User{ID: 1}

	setRepeat := func(t *testing.T, taskID int64, re *TaskRepeat) {
		s := db.NewSession()
		defer s.Close()
		re.TaskID = taskID
		require.NoError(t, SetTaskRepeat(s, re))
		require.NoError(t, s.Commit())
	}
	complete := func(t *testing.T) {
		s := db.NewSession()
		defer s.Close()
		task := &Task{ID: 28, Done: true}
		require.NoError(t, task.updateSingleTask(s, u, []string{"done"}))
		require.NoError(t, s.Commit())
	}
	read := func(t *testing.T, id int64) *Task {
		s := db.NewSession()
		defer s.Close()
		task := &Task{ID: id}
		require.NoError(t, task.ReadOne(s, u))
		return task
	}

	t.Run("the rule picks the next date; the interval stays", func(t *testing.T) {
		loadRepeatFixtures(t)
		// Fixture task 28 repeats hourly, due 2 December 2018 22:25:24.
		setRepeat(t, 28, &TaskRepeat{Rule: "FREQ=WEEKLY;BYDAY=MO,TH"})
		before := read(t, 28)
		complete(t)

		task := read(t, 28)
		assert.False(t, task.Done)
		assert.Equal(t, int64(3600), task.RepeatAfter, "the stored interval is put back")
		due := task.DueDate.In(config.GetTimeZone())
		assert.Contains(t, []time.Weekday{time.Monday, time.Thursday}, due.Weekday())
		assert.True(t, due.After(time.Now()))
		assert.True(t, due.Before(time.Now().Add(5*24*time.Hour)), "the next Monday or Thursday, not a later one")
		wantClock := before.DueDate.In(config.GetTimeZone())
		assert.Equal(t, wantClock.Hour(), due.Hour(), "keeps the time of day")
		assert.Equal(t, wantClock.Minute(), due.Minute())
		assert.True(t, task.StartDate.After(before.StartDate), "the start date moves too")
	})
	t.Run("the rule's last occurrence by count stays done", func(t *testing.T) {
		loadRepeatFixtures(t)
		setRepeat(t, 28, &TaskRepeat{Rule: "FREQ=MONTHLY;BYDAY=2TU", RepeatsLeft: 1})
		complete(t)

		task := read(t, 28)
		assert.True(t, task.Done)
		assert.Equal(t, int64(0), task.RepeatAfter)
		db.AssertMissing(t, "task_repeats", map[string]any{"task_id": 28})
	})
	t.Run("an end date before the rule's next date ends the series", func(t *testing.T) {
		loadRepeatFixtures(t)
		until := time.Now()
		setRepeat(t, 28, &TaskRepeat{Rule: "FREQ=YEARLY", Until: &until})
		complete(t)
		assert.True(t, read(t, 28).Done)
	})
	t.Run("a count with a rule counts down", func(t *testing.T) {
		loadRepeatFixtures(t)
		setRepeat(t, 28, &TaskRepeat{Rule: "FREQ=WEEKLY;BYDAY=FR", RepeatsLeft: 3})
		complete(t)

		s := db.NewSession()
		defer s.Close()
		re, err := GetTaskRepeat(s, 28)
		require.NoError(t, err)
		assert.Equal(t, 2, re.RepeatsLeft)
		assert.Equal(t, "FREQ=WEEKLY;BYDAY=FR", re.Rule)
	})
	t.Run("a rule makes a task that didn't repeat repeat", func(t *testing.T) {
		loadRepeatFixtures(t)
		// Fixture task 1 doesn't repeat.
		setRepeat(t, 1, &TaskRepeat{Rule: "rrule:freq=monthly;interval=2"})

		task := read(t, 1)
		assert.Equal(t, int64(60*24*3600), task.RepeatAfter, "two 30-day months stand in")
		s := db.NewSession()
		defer s.Close()
		re, err := GetTaskRepeat(s, 1)
		require.NoError(t, err)
		assert.Equal(t, "FREQ=MONTHLY;INTERVAL=2", re.Rule, "stored in its canonical form")
	})
	t.Run("rules outside the subset are refused", func(t *testing.T) {
		loadRepeatFixtures(t)
		s := db.NewSession()
		defer s.Close()
		for _, rule := range []string{"FREQ=DAILY", "FREQ=WEEKLY;BYHOUR=9", "FREQ=WEEKLY;COUNT=3", "FREQ=MONTHLY;INTERVAL=500", "nonsense"} {
			err := SetTaskRepeat(s, &TaskRepeat{TaskID: 28, Rule: rule})
			assert.ErrorAs(t, err, &ErrInvalidTaskRepeatRule{}, rule)
		}
	})
}

func TestSkipTaskOccurrence(t *testing.T) {
	u := &user.User{ID: 1}

	setRepeat := func(t *testing.T, re *TaskRepeat) {
		s := db.NewSession()
		defer s.Close()
		re.TaskID = 28
		require.NoError(t, SetTaskRepeat(s, re))
		require.NoError(t, s.Commit())
	}
	skip := func(t *testing.T, taskID int64) (*Task, error) {
		s := db.NewSession()
		defer s.Close()
		task, err := SkipTaskOccurrence(s, u, taskID)
		if err != nil {
			return nil, err
		}
		require.NoError(t, s.Commit())
		return task, nil
	}

	t.Run("moves to the next occurrence without completing it", func(t *testing.T) {
		loadRepeatFixtures(t)
		// Fixture task 28 repeats hourly, due 2 December 2018.
		task, err := skip(t, 28)
		require.NoError(t, err)
		assert.False(t, task.Done)
		assert.True(t, task.DoneAt.IsZero(), "a skip isn't a completion")
		assert.True(t, task.DueDate.After(time.Now()))
		assert.True(t, task.DueDate.Before(time.Now().Add(time.Hour+time.Minute)), "due %s, now %s", task.DueDate, time.Now())
		assert.Equal(t, int64(3600), task.RepeatAfter)
	})
	t.Run("follows the rule and keeps the count", func(t *testing.T) {
		loadRepeatFixtures(t)
		setRepeat(t, &TaskRepeat{Rule: "FREQ=WEEKLY;BYDAY=MO,TH", RepeatsLeft: 3})
		task, err := skip(t, 28)
		require.NoError(t, err)
		due := task.DueDate.In(config.GetTimeZone())
		assert.Contains(t, []time.Weekday{time.Monday, time.Thursday}, due.Weekday())
		assert.Equal(t, int64(3600), task.RepeatAfter, "the stored interval stays")

		s := db.NewSession()
		defer s.Close()
		re, err := GetTaskRepeat(s, 28)
		require.NoError(t, err)
		assert.Equal(t, 3, re.RepeatsLeft, "skipping doesn't count down")
	})
	t.Run("past the end date the series is over", func(t *testing.T) {
		loadRepeatFixtures(t)
		until := time.Now()
		setRepeat(t, &TaskRepeat{Until: &until})
		task, err := skip(t, 28)
		require.NoError(t, err)
		assert.True(t, task.Done)
		assert.Equal(t, int64(0), task.RepeatAfter)
	})
	t.Run("a task that doesn't repeat can't be skipped", func(t *testing.T) {
		loadRepeatFixtures(t)
		_, err := skip(t, 1)
		assert.ErrorAs(t, err, &ErrTaskDoesNotRepeat{})
	})
	t.Run("next is worked out on read", func(t *testing.T) {
		loadRepeatFixtures(t)
		setRepeat(t, &TaskRepeat{Rule: "FREQ=WEEKLY;BYDAY=FR"})
		s := db.NewSession()
		defer s.Close()
		re, err := GetTaskRepeat(s, 28)
		require.NoError(t, err)
		require.NoError(t, re.FillNext(s))
		require.NotNil(t, re.Next)
		assert.Equal(t, time.Friday, re.Next.In(config.GetTimeZone()).Weekday())

		until := time.Now()
		re.Until = &until
		require.NoError(t, re.FillNext(s))
		assert.Nil(t, re.Next, "no next once the end comes first")
	})
}
