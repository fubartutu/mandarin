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

// Mandarin: a repeating task's own settings beyond Vikunja's interval, like
// Todoist: a calendar rule ("every Mon and Thu", "every 2nd Tuesday", "every
// 2 months") and an end (a last date or a number of times). A new file in
// Vikunja's package (not pkg/household) because it hooks into Task.Update and
// uses its unexported date helpers; Vikunja's Task struct and task routes stay
// untouched.

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/web"

	"github.com/teambition/rrule-go"
	"xorm.io/xorm"
)

// MaxTaskRepeatsLeft caps the count: more than this is "forever" in practice.
const MaxTaskRepeatsLeft = 1000

// TaskRepeat holds a repeating task's household repeat settings, one row per
// task. Rule, when set, decides the next date instead of Vikunja's interval.
// Until and RepeatsLeft say when it stops: either or both may be set, and
// whichever comes first ends the series. No row means none of these.
type TaskRepeat struct {
	TaskID int64 `xorm:"bigint not null unique pk" json:"task_id" readOnly:"true" doc:"The task these settings belong to."`
	// Rule is an iCalendar RRULE without DTSTART, e.g. "FREQ=WEEKLY;BYDAY=MO,TH".
	// The task's due date is the series' start.
	Rule string `xorm:"varchar(250) not null default ''" json:"rule" required:"true" maxLength:"250" doc:"An iCalendar RRULE deciding the next date when the task is done, e.g. FREQ=WEEKLY;BYDAY=MO,TH or FREQ=MONTHLY;BYDAY=2TU. Only FREQ (WEEKLY, MONTHLY, YEARLY), INTERVAL, BYDAY, BYMONTHDAY and BYSETPOS. Empty means Vikunja's repeat_after decides. Setting one on a task that doesn't repeat makes it repeat."`
	// Until is the last moment an occurrence may fall on. The frontend sends
	// the end of the chosen day, so an occurrence that evening still counts.
	Until *time.Time `xorm:"datetime null 'repeat_until'" json:"until" doc:"The latest an occurrence may be due. Completing the task when its next due date would be later ends the series. Null means no end date."`
	// RepeatsLeft counts the occurrences still to do, the open one included:
	// completing a task with 1 left ends the series. 0 means no limit.
	RepeatsLeft int `xorm:"int not null default 0" json:"repeats_left" minimum:"0" maximum:"1000" doc:"How many occurrences are left, the current one included. Completing the task with 1 left ends the series; 0 means no limit."`
}

// TableName holds the table name
func (*TaskRepeat) TableName() string {
	return "task_repeats"
}

// IsEmpty is true when there's nothing to store: no rule and no end.
func (re *TaskRepeat) IsEmpty() bool {
	return re.Rule == "" && re.Until == nil && re.RepeatsLeft == 0
}

// GetTaskRepeat returns the task's repeat settings, or empty ones (not an
// error) when it has none.
func GetTaskRepeat(s *xorm.Session, taskID int64) (*TaskRepeat, error) {
	re := &TaskRepeat{TaskID: taskID}
	if _, err := s.Where("task_id = ?", taskID).Get(re); err != nil {
		return nil, err
	}
	return re, nil
}

// SetTaskRepeat stores re, replacing what the task had. Empty settings delete
// the row. A rule on a task that doesn't repeat yet gives it a stand-in
// interval (see ruleStandIn). Permission checks are the caller's.
func SetTaskRepeat(s *xorm.Session, re *TaskRepeat) error {
	if re.RepeatsLeft < 0 || re.RepeatsLeft > MaxTaskRepeatsLeft {
		return ErrInvalidTaskRepeat{RepeatsLeft: re.RepeatsLeft}
	}
	re.Rule = strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(re.Rule)), "RRULE:"))
	if re.Rule != "" {
		if _, err := parseRepeatRule(re.Rule); err != nil {
			return err
		}
		if err := giveRuleStandIn(s, re); err != nil {
			return err
		}
	}
	if re.IsEmpty() {
		_, err := s.Where("task_id = ?", re.TaskID).Delete(&TaskRepeat{})
		return err
	}
	has, err := s.Where("task_id = ?", re.TaskID).Exist(&TaskRepeat{})
	if err != nil {
		return err
	}
	if !has {
		_, err = s.Insert(re)
		return err
	}
	// AllCols, or xorm would skip an empty Rule, a nil Until and a zero RepeatsLeft.
	_, err = s.Where("task_id = ?", re.TaskID).AllCols().Update(re)
	return err
}

// giveRuleStandIn makes a task that doesn't repeat yet repeat, so Vikunja
// (and CalDAV, the task list's repeat icon, other clients) treats it as
// repeating. A task that already repeats keeps its interval: the frontend
// saves the stand-in itself, and the rule decides the dates either way.
func giveRuleStandIn(s *xorm.Session, re *TaskRepeat) error {
	task := &Task{}
	has, err := s.Where("id = ?", re.TaskID).Get(task)
	if err != nil {
		return err
	}
	if !has {
		return ErrTaskDoesNotExist{ID: re.TaskID}
	}
	if task.isRepeating() {
		return nil
	}
	task.RepeatAfter = ruleStandIn(re.Rule)
	task.RepeatMode = TaskRepeatModeDefault
	_, err = s.ID(task.ID).Cols("repeat_after", "repeat_mode").Update(task)
	return err
}

// ruleStandIn is the repeat_after a task with this rule shows to code that
// only knows intervals: a week, a month (30 days) or a year (365 days), times
// the rule's interval.
func ruleStandIn(rule string) int64 {
	opt, err := parseRepeatRule(rule)
	if err != nil {
		return 0
	}
	interval := int64(max(opt.Interval, 1))
	day := int64(24 * 3600)
	switch opt.Freq {
	case rrule.WEEKLY:
		return interval * 7 * day
	case rrule.MONTHLY:
		return interval * 30 * day
	case rrule.YEARLY:
		return interval * 365 * day
	case rrule.SECONDLY, rrule.MINUTELY, rrule.HOURLY, rrule.DAILY:
	}
	return 0
}

var allowedRuleParts = []string{"FREQ", "INTERVAL", "BYDAY", "BYMONTHDAY", "BYSETPOS"}

// parseRepeatRule checks a rule against the subset Mandarin supports and
// parses it. Plain day and hour intervals are Vikunja's repeat_after.
func parseRepeatRule(rule string) (*rrule.ROption, error) {
	for _, part := range strings.Split(rule, ";") {
		key, _, _ := strings.Cut(part, "=")
		if !slices.Contains(allowedRuleParts, key) {
			return nil, ErrInvalidTaskRepeatRule{Rule: rule}
		}
	}
	opt, err := rrule.StrToROption(rule)
	if err != nil {
		return nil, ErrInvalidTaskRepeatRule{Rule: rule}
	}
	if opt.Freq != rrule.WEEKLY && opt.Freq != rrule.MONTHLY && opt.Freq != rrule.YEARLY {
		return nil, ErrInvalidTaskRepeatRule{Rule: rule}
	}
	if opt.Interval > 100 {
		return nil, ErrInvalidTaskRepeatRule{Rule: rule}
	}
	return opt, nil
}

// nextRuleDate is the rule's first occurrence after now (or after ref when
// the task is done early), counted from ref in the server's time zone so
// weekdays and times of day are local. Zero when the rule has no more dates.
func nextRuleDate(rule string, ref, now time.Time) (time.Time, error) {
	opt, err := parseRepeatRule(rule)
	if err != nil {
		return time.Time{}, err
	}
	opt.Dtstart = ref.In(config.GetTimeZone())
	r, err := rrule.NewRRule(*opt)
	if err != nil {
		return time.Time{}, ErrInvalidTaskRepeatRule{Rule: rule}
	}
	after := now
	if ref.After(now) {
		after = ref
	}
	return r.After(after, false), nil
}

// repeatRef is the date a task's repeat counts from: the due date, else the
// start or end date, else now for a task without dates.
func repeatRef(task *Task, now time.Time) time.Time {
	for _, d := range []time.Time{task.DueDate, task.StartDate, task.EndDate} {
		if !d.IsZero() {
			return d
		}
	}
	return now
}

// applyTaskRepeat runs in Task.Update before a repeating task is rescheduled.
//
//   - Removing a task's repeat removes its settings too.
//   - Completing a task with a rule: the rule picks the next date, and the
//     task's interval is set (for this update only) to exactly the gap to it,
//     so Vikunja's own updateDone moves the due date there and the start/end
//     dates and reminders with it. The returned function puts the stored
//     interval back and must run right after updateDone.
//   - Completing the series' last occurrence (by date or count) clears the
//     repeat so the task stays done; otherwise the count goes down by one.
func applyTaskRepeat(s *xorm.Session, ot, t *Task, colsToUpdate *[]string) (restore func(), err error) {
	restore = func() {}
	if ot.isRepeating() && !t.isRepeating() {
		return restore, deleteTaskRepeat(s, ot.ID)
	}
	if ot.Done || !t.Done || !ot.isRepeating() {
		return restore, nil
	}
	re, err := GetTaskRepeat(s, ot.ID)
	if err != nil || re.IsEmpty() {
		return restore, err
	}

	now := time.Now()
	ref := repeatRef(ot, now)
	next := nextRepeatDate(ot, now)
	if re.Rule != "" {
		next, err = nextRuleDate(re.Rule, ref, now)
		if err != nil {
			return restore, err
		}
	}

	last := next.IsZero() || re.RepeatsLeft == 1 || (re.Until != nil && next.After(*re.Until))
	if last {
		ot.RepeatAfter, t.RepeatAfter = 0, 0
		ot.RepeatMode, t.RepeatMode = TaskRepeatModeDefault, TaskRepeatModeDefault
		// With an explicit field list the repeat columns may not be in it.
		for _, col := range []string{"repeat_after", "repeat_mode"} {
			if !slices.Contains(*colsToUpdate, col) {
				*colsToUpdate = append(*colsToUpdate, col)
			}
		}
		return restore, deleteTaskRepeat(s, ot.ID)
	}

	if re.RepeatsLeft > 0 {
		re.RepeatsLeft--
		if _, err = s.Where("task_id = ?", ot.ID).Cols("repeats_left").Update(re); err != nil {
			return restore, err
		}
	}

	if re.Rule != "" {
		// addRepeatIntervalToTime adds the interval until the date is after
		// now; next is after now and one gap from ref, so it lands on next.
		storedAfter, storedMode := ot.RepeatAfter, ot.RepeatMode
		requestAfter, requestMode := t.RepeatAfter, t.RepeatMode
		gap := int64(next.Sub(ref) / time.Second)
		ot.RepeatAfter, t.RepeatAfter = gap, gap
		ot.RepeatMode, t.RepeatMode = TaskRepeatModeDefault, TaskRepeatModeDefault
		restore = func() {
			ot.RepeatAfter, ot.RepeatMode = storedAfter, storedMode
			t.RepeatAfter, t.RepeatMode = requestAfter, requestMode
		}
	}
	return restore, nil
}

// nextRepeatDate is when the next occurrence would be due under Vikunja's
// interval, worked out the same way updateDone moves the dates.
func nextRepeatDate(task *Task, now time.Time) time.Time {
	ref := repeatRef(task, now)
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

// deleteTaskRepeat removes a task's repeat settings.
func deleteTaskRepeat(s *xorm.Session, taskID int64) error {
	_, err := s.Where("task_id = ?", taskID).Delete(&TaskRepeat{})
	return err
}

// ErrInvalidTaskRepeat is returned for a count out of range.
type ErrInvalidTaskRepeat struct {
	RepeatsLeft int
}

func (err ErrInvalidTaskRepeat) Error() string {
	return fmt.Sprintf("Invalid task repeat end [RepeatsLeft: %d]", err.RepeatsLeft)
}

// ErrCodeInvalidTaskRepeat holds the unique world-error code of this error.
// Mandarin's task error codes are 904xx (menu has 903xx).
const ErrCodeInvalidTaskRepeat = 90401

// HTTPError holds the http error description
func (err ErrInvalidTaskRepeat) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusBadRequest,
		Code:     ErrCodeInvalidTaskRepeat,
		Message:  fmt.Sprintf("The number of repeats must be between 0 and %d.", MaxTaskRepeatsLeft),
	}
}

// ErrInvalidTaskRepeatRule is returned for a rule Mandarin can't use.
type ErrInvalidTaskRepeatRule struct {
	Rule string
}

func (err ErrInvalidTaskRepeatRule) Error() string {
	return fmt.Sprintf("Invalid task repeat rule [Rule: %s]", err.Rule)
}

// ErrCodeInvalidTaskRepeatRule holds the unique world-error code of this error.
const ErrCodeInvalidTaskRepeatRule = 90402

// HTTPError holds the http error description
func (err ErrInvalidTaskRepeatRule) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusBadRequest,
		Code:     ErrCodeInvalidTaskRepeatRule,
		Message:  "The repeat rule must be an RRULE with FREQ WEEKLY, MONTHLY or YEARLY and only INTERVAL (at most 100), BYDAY, BYMONTHDAY or BYSETPOS.",
	}
}
