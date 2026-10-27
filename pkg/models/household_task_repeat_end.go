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

// Mandarin: an end for a repeating task, either a last date ("until 5 November")
// or a number of times ("10 times"), like Todoist. A new file in Vikunja's
// package (not pkg/household) because it hooks into Task.Update and uses its
// unexported date helpers; Vikunja's Task struct and task routes stay untouched.

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// MaxTaskRepeatsLeft caps the count: more than this is "forever" in practice.
const MaxTaskRepeatsLeft = 1000

// TaskRepeatEnd says when a repeating task stops repeating. Either or both
// may be set; whichever comes first ends the series. No row means no end.
type TaskRepeatEnd struct {
	TaskID int64 `xorm:"bigint not null unique pk" json:"task_id" readOnly:"true" doc:"The task this end belongs to."`
	// Until is the last moment an occurrence may fall on. The frontend sends
	// the end of the chosen day, so an occurrence that evening still counts.
	Until *time.Time `xorm:"datetime null 'repeat_until'" json:"until" doc:"The latest an occurrence may be due. Completing the task when its next due date would be later ends the series. Null means no end date."`
	// RepeatsLeft counts the occurrences still to do, the open one included:
	// completing a task with 1 left ends the series. 0 means no limit.
	RepeatsLeft int `xorm:"int not null default 0" json:"repeats_left" minimum:"0" maximum:"1000" doc:"How many occurrences are left, the current one included. Completing the task with 1 left ends the series; 0 means no limit."`
}

// TableName holds the table name
func (*TaskRepeatEnd) TableName() string {
	return "task_repeat_ends"
}

// IsEmpty is true when nothing would ever end the series.
func (re *TaskRepeatEnd) IsEmpty() bool {
	return re.Until == nil && re.RepeatsLeft == 0
}

// GetTaskRepeatEnd returns the task's repeat end, or an empty one (not an
// error) when it has none.
func GetTaskRepeatEnd(s *xorm.Session, taskID int64) (*TaskRepeatEnd, error) {
	re := &TaskRepeatEnd{TaskID: taskID}
	if _, err := s.Where("task_id = ?", taskID).Get(re); err != nil {
		return nil, err
	}
	return re, nil
}

// SetTaskRepeatEnd stores re, replacing what the task had. An empty one
// deletes the row. Permission checks are the caller's.
func SetTaskRepeatEnd(s *xorm.Session, re *TaskRepeatEnd) error {
	if re.RepeatsLeft < 0 || re.RepeatsLeft > MaxTaskRepeatsLeft {
		return ErrInvalidTaskRepeatEnd{RepeatsLeft: re.RepeatsLeft}
	}
	if re.IsEmpty() {
		_, err := s.Where("task_id = ?", re.TaskID).Delete(&TaskRepeatEnd{})
		return err
	}
	has, err := s.Where("task_id = ?", re.TaskID).Exist(&TaskRepeatEnd{})
	if err != nil {
		return err
	}
	if !has {
		_, err = s.Insert(re)
		return err
	}
	// AllCols, or xorm would skip a nil Until and a zero RepeatsLeft.
	_, err = s.Where("task_id = ?", re.TaskID).AllCols().Update(re)
	return err
}

// applyTaskRepeatEnd runs in Task.Update before a repeating task is
// rescheduled. When the occurrence being completed is the series' last, it
// clears the repeat (on both the stored task and the update) so the task is
// done for good: it lands in the done bucket and updateDone leaves its dates
// alone. Otherwise it counts one occurrence off. Removing a task's repeat
// removes its end too, so it doesn't come back with a later repeat.
func applyTaskRepeatEnd(s *xorm.Session, ot, t *Task, colsToUpdate *[]string) error {
	if ot.isRepeating() && !t.isRepeating() {
		return deleteTaskRepeatEnd(s, ot.ID)
	}
	if ot.Done || !t.Done || !ot.isRepeating() {
		return nil
	}
	re, err := GetTaskRepeatEnd(s, ot.ID)
	if err != nil || re.IsEmpty() {
		return err
	}

	last := re.RepeatsLeft == 1 ||
		(re.Until != nil && nextRepeatDate(ot, time.Now()).After(*re.Until))
	if !last {
		if re.RepeatsLeft == 0 {
			return nil
		}
		re.RepeatsLeft--
		_, err = s.Where("task_id = ?", ot.ID).Cols("repeats_left").Update(re)
		return err
	}

	ot.RepeatAfter, t.RepeatAfter = 0, 0
	ot.RepeatMode, t.RepeatMode = TaskRepeatModeDefault, TaskRepeatModeDefault
	// With an explicit field list the repeat columns may not be in it.
	for _, col := range []string{"repeat_after", "repeat_mode"} {
		if !slices.Contains(*colsToUpdate, col) {
			*colsToUpdate = append(*colsToUpdate, col)
		}
	}
	_, err = s.Where("task_id = ?", ot.ID).Delete(&TaskRepeatEnd{})
	return err
}

// nextRepeatDate is when the next occurrence would be due, worked out the
// same way updateDone moves the dates: from the due date, else the start or
// end date, else from now for a task without dates.
func nextRepeatDate(task *Task, now time.Time) time.Time {
	ref := now
	for _, d := range []time.Time{task.DueDate, task.StartDate, task.EndDate} {
		if !d.IsZero() {
			ref = d
			break
		}
	}
	interval := time.Duration(task.RepeatAfter) * time.Second
	switch task.RepeatMode {
	case TaskRepeatModeMonth:
		return addOneMonthToDate(ref)
	case TaskRepeatModeFromCurrentDate:
		return now.Add(interval)
	case TaskRepeatModeDefault:
		return addRepeatIntervalToTime(now, ref, interval)
	}
	return addRepeatIntervalToTime(now, ref, interval)
}

// deleteTaskRepeatEnd removes the end when a task is permanently deleted.
func deleteTaskRepeatEnd(s *xorm.Session, taskID int64) error {
	_, err := s.Where("task_id = ?", taskID).Delete(&TaskRepeatEnd{})
	return err
}

// ErrInvalidTaskRepeatEnd is returned for a count out of range.
type ErrInvalidTaskRepeatEnd struct {
	RepeatsLeft int
}

func (err ErrInvalidTaskRepeatEnd) Error() string {
	return fmt.Sprintf("Invalid task repeat end [RepeatsLeft: %d]", err.RepeatsLeft)
}

// ErrCodeInvalidTaskRepeatEnd holds the unique world-error code of this error.
// Mandarin's task error codes are 904xx (menu has 903xx).
const ErrCodeInvalidTaskRepeatEnd = 90401

// HTTPError holds the http error description
func (err ErrInvalidTaskRepeatEnd) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusBadRequest,
		Code:     ErrCodeInvalidTaskRepeatEnd,
		Message:  fmt.Sprintf("The number of repeats must be between 0 and %d.", MaxTaskRepeatsLeft),
	}
}
