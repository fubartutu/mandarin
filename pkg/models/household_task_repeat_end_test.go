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

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRepeatEnd(t *testing.T) {
	u := &user.User{ID: 1}

	// Fixture task 28 repeats hourly and belongs to user 1.
	setEnd := func(t *testing.T, re *TaskRepeatEnd) {
		s := db.NewSession()
		defer s.Close()
		re.TaskID = 28
		require.NoError(t, SetTaskRepeatEnd(s, re))
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
	read := func(t *testing.T) (*Task, *TaskRepeatEnd) {
		s := db.NewSession()
		defer s.Close()
		task := &Task{ID: 28}
		require.NoError(t, task.ReadOne(s, u))
		re, err := GetTaskRepeatEnd(s, 28)
		require.NoError(t, err)
		return task, re
	}

	t.Run("a count above one reopens the task and counts down", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		setEnd(t, &TaskRepeatEnd{RepeatsLeft: 3})
		complete(t, &Task{RepeatAfter: 3600}, nil)

		task, re := read(t)
		assert.False(t, task.Done)
		assert.Equal(t, int64(3600), task.RepeatAfter)
		assert.Equal(t, 2, re.RepeatsLeft)
	})
	t.Run("the last occurrence stays done and stops repeating", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		setEnd(t, &TaskRepeatEnd{RepeatsLeft: 1})
		complete(t, &Task{RepeatAfter: 3600}, nil)

		task, re := read(t)
		assert.True(t, task.Done)
		assert.Equal(t, int64(0), task.RepeatAfter)
		assert.True(t, task.DueDate.Before(time.Now()), "the due date is not moved on")
		assert.True(t, re.IsEmpty(), "the end is removed with the repeat")
	})
	t.Run("an end date before the next occurrence ends the series", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		until := time.Now() // the next occurrence is always after now
		setEnd(t, &TaskRepeatEnd{Until: &until})
		complete(t, &Task{RepeatAfter: 3600}, nil)

		task, _ := read(t)
		assert.True(t, task.Done)
		assert.Equal(t, int64(0), task.RepeatAfter)
	})
	t.Run("an end date after the next occurrence keeps repeating", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		until := time.Now().Add(48 * time.Hour)
		setEnd(t, &TaskRepeatEnd{Until: &until})
		complete(t, &Task{RepeatAfter: 3600}, nil)

		task, re := read(t)
		assert.False(t, task.Done)
		require.NotNil(t, re.Until)
		assert.WithinDuration(t, until, *re.Until, time.Second)
	})
	t.Run("ends the series when only done is sent", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		setEnd(t, &TaskRepeatEnd{RepeatsLeft: 1})
		complete(t, &Task{}, []string{"done"})

		task, _ := read(t)
		assert.True(t, task.Done)
		assert.Equal(t, int64(0), task.RepeatAfter)
	})
	t.Run("removing the repeat removes the end", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		setEnd(t, &TaskRepeatEnd{RepeatsLeft: 3})
		s := db.NewSession()
		defer s.Close()
		task := &Task{ID: 28, RepeatAfter: 0}
		require.NoError(t, task.updateSingleTask(s, u, []string{"repeat_after"}))
		require.NoError(t, s.Commit())
		db.AssertMissing(t, "task_repeat_ends", map[string]any{"task_id": 28})
	})
	t.Run("an empty end is deleted, a bad count refused", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		setEnd(t, &TaskRepeatEnd{RepeatsLeft: 4})
		setEnd(t, &TaskRepeatEnd{})
		db.AssertMissing(t, "task_repeat_ends", map[string]any{"task_id": 28})

		s := db.NewSession()
		defer s.Close()
		err := SetTaskRepeatEnd(s, &TaskRepeatEnd{TaskID: 28, RepeatsLeft: -1})
		assert.ErrorAs(t, err, &ErrInvalidTaskRepeatEnd{})
	})
}
