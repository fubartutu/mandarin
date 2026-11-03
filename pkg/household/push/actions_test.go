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

package push

import (
	"strings"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionToken(t *testing.T) {
	now := time.Now()
	due := now.Add(time.Hour)
	token := NewActionToken(1, 12, due, now)

	c, err := parseActionToken(token, now.Add(47*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, actionClaims{User: 1, Task: 12, Due: due.Unix(), Expires: now.Add(actionTokenTTL).Unix()}, *c)

	_, err = parseActionToken(token, now.Add(49*time.Hour))
	assert.Equal(t, ErrInvalidActionToken{Expired: true}, err, "expired after 48 hours")

	// Another user or task in the payload breaks the signature.
	payload, sig, _ := strings.Cut(token, ".")
	forged, _, _ := strings.Cut(NewActionToken(2, 12, due, now), ".")
	for _, bad := range []string{"", "nodot", payload + ".", "." + sig, forged + "." + sig, token + "x"} {
		_, err = parseActionToken(bad, now)
		assert.Equal(t, ErrInvalidActionToken{}, err, bad)
	}

	assert.Equal(t, int64(0), dueUnix(time.Time{}), "no due date")
}

func TestTaskActionsOnMessages(t *testing.T) {
	now := time.Now()
	due := now.Add(time.Hour)
	msg, _ := messageFor(&models.ReminderDueNotification{Task: &models.Task{ID: 12, Title: "Bins", DueDate: due}})
	addTaskActions(msg, 1, now)
	assert.Equal(t, []Action{{"done", "Done"}, {"snooze", "Snooze 1h"}}, msg.Actions)
	c, err := parseActionToken(msg.ActionToken, now)
	require.NoError(t, err)
	assert.Equal(t, int64(12), c.Task)
	assert.Equal(t, due.Unix(), c.Due)

	msg, _ = messageFor(&models.UndoneTaskOverdueNotification{Task: &models.Task{ID: 7, Title: "Tax", DueDate: due}})
	addTaskActions(msg, 1, now)
	assert.Len(t, msg.Actions, 2, "one overdue task gets the buttons")

	msg, _ = messageFor(&models.UndoneTasksOverdueNotification{Assigned: map[int64]*models.Task{
		1: {ID: 1, Title: "Bins", DueDate: due},
		2: {ID: 2, Title: "Tax", DueDate: due},
	}})
	addTaskActions(msg, 1, now)
	assert.Empty(t, msg.Actions, "a summary of several tasks doesn't")
	assert.Empty(t, msg.ActionToken)
}

func doAction(t *testing.T, action string, userID, taskID int64, due time.Time, now time.Time) (*ActionResult, error) {
	t.Helper()
	s := db.NewSession()
	defer s.Close()
	r, err := DoAction(s, &ActionInput{Token: NewActionToken(userID, taskID, due, now), Action: action}, now)
	if err != nil {
		_ = s.Rollback()
		return nil, err
	}
	require.NoError(t, s.Commit())
	return r, nil
}

func getTask(t *testing.T, id int64) models.Task {
	t.Helper()
	s := db.NewSession()
	defer s.Close()
	task, err := models.GetTaskByIDSimple(s, id)
	require.NoError(t, err)
	return task
}

func TestActionDone(t *testing.T) {
	setup(t)
	now := time.Now()

	// Task 1 has no due date.
	r, err := doAction(t, ActionDone, 1, 1, time.Time{}, now)
	require.NoError(t, err)
	assert.Equal(t, "done", r.Result)
	assert.True(t, getTask(t, 1).Done)
	r, err = doAction(t, ActionDone, 1, 1, time.Time{}, now)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", r.Result, "tapping an old notification again does nothing")

	// Task 28 repeats every hour: Done moves it on and leaves it open.
	before := getTask(t, 28)
	r, err = doAction(t, ActionDone, 1, 28, before.DueDate, now)
	require.NoError(t, err)
	assert.Equal(t, "done", r.Result)
	after := getTask(t, 28)
	assert.False(t, after.Done)
	assert.True(t, after.DueDate.After(before.DueDate), "next occurrence")
	r, err = doAction(t, ActionDone, 1, 28, before.DueDate, now)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", r.Result, "an old notification can't skip a second occurrence")
	assert.True(t, getTask(t, 28).DueDate.Equal(after.DueDate))

	// Task 14 is in someone else's project.
	_, err = doAction(t, ActionDone, 1, 14, time.Time{}, now)
	require.ErrorAs(t, err, &models.ErrGenericForbidden{})
	assert.False(t, getTask(t, 14).Done)

	_, err = doAction(t, ActionDone, 1, 99999, time.Time{}, now)
	assert.True(t, models.IsErrTaskDoesNotExist(err), err)
	_, err = doAction(t, ActionDone, 99999, 1, time.Time{}, now)
	require.ErrorAs(t, err, &ErrInvalidActionToken{}, "no such user")
}

func snoozes(t *testing.T) []*pushSnooze {
	t.Helper()
	s := db.NewSession()
	defer s.Close()
	all := []*pushSnooze{}
	require.NoError(t, s.OrderBy("id").Find(&all))
	return all
}

func TestActionSnooze(t *testing.T) {
	setup(t)
	now := time.Date(2026, 11, 3, 9, 30, 20, 0, time.UTC)

	r, err := doAction(t, ActionSnooze, 1, 1, time.Time{}, now)
	require.NoError(t, err)
	assert.Equal(t, "snoozed", r.Result)
	require.NotNil(t, r.SnoozedUntil)
	assert.True(t, r.SnoozedUntil.Equal(time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)), r.SnoozedUntil)
	assert.False(t, getTask(t, 1).Done, "the task itself is untouched")

	_, err = doAction(t, ActionSnooze, 1, 1, time.Time{}, now.Add(10*time.Minute))
	require.NoError(t, err)
	all := snoozes(t)
	require.Len(t, all, 1, "snoozing again moves the snooze")
	assert.True(t, all[0].At.Equal(time.Date(2026, 11, 3, 10, 40, 0, 0, time.UTC)), all[0].At)
	assert.Equal(t, int64(1), all[0].UserID)

	_, err = doAction(t, ActionSnooze, 2, 1, time.Time{}, now)
	require.ErrorAs(t, err, &models.ErrGenericForbidden{}, "user 2 can't see task 1")
}

func TestSnoozedReminders(t *testing.T) {
	service := fakeService(t)
	s := setup(t)
	addDevice(t, s, user1, "https://fcm.googleapis.com/phone")
	now := time.Now()

	add := func(taskID int64, at time.Time) {
		s := db.NewSession()
		defer s.Close()
		require.NoError(t, snooze(s, 1, taskID, at))
		require.NoError(t, s.Commit())
	}
	add(1, now.Add(-time.Minute)) // due
	add(2, now.Add(-time.Minute)) // due, but task 2 is done
	add(3, now.Add(time.Minute))  // not yet

	sendSnoozedReminders(now)
	assert.Len(t, service.requests, 1, "only the open task's reminder")
	left := snoozes(t)
	require.Len(t, left, 1)
	assert.Equal(t, int64(3), left[0].TaskID)

	sendSnoozedReminders(now)
	assert.Len(t, service.requests, 1, "sent once")
}

func TestSnoozedMessage(t *testing.T) {
	setup(t)
	s := db.NewSession()
	require.NoError(t, snooze(s, 1, 1, time.Now()))
	require.NoError(t, s.Commit())
	s.Close()

	all := snoozes(t)
	require.Len(t, all, 1)
	msg, err := snoozedMessage(all[0])
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, "task #1", msg.Title)
	assert.Equal(t, "/tasks/1", msg.URL)
	assert.Len(t, msg.Actions, 2, "the buttons again")
	assert.NotEmpty(t, msg.ActionToken)
	assert.Empty(t, snoozes(t))
}

func TestSnoozeOnTaskPage(t *testing.T) {
	setup(t)
	s := setup0(t)
	state, err := GetSnooze(s, user1, 1)
	require.NoError(t, err)
	assert.Nil(t, state.SnoozedUntil)

	at := time.Date(2026, 11, 3, 10, 40, 0, 0, time.UTC)
	require.NoError(t, snooze(s, 1, 1, at))
	state, err = GetSnooze(s, user1, 1)
	require.NoError(t, err)
	require.NotNil(t, state.SnoozedUntil)
	assert.True(t, state.SnoozedUntil.Equal(at))

	_, err = GetSnooze(s, user2, 1)
	require.ErrorAs(t, err, &models.ErrGenericForbidden{}, "user 2 can't see task 1")
	require.ErrorAs(t, CancelSnooze(s, user2, 1), &models.ErrGenericForbidden{})

	require.NoError(t, CancelSnooze(s, user1, 1))
	state, err = GetSnooze(s, user1, 1)
	require.NoError(t, err)
	assert.Nil(t, state.SnoozedUntil)
	require.NoError(t, CancelSnooze(s, user1, 1), "nothing to cancel")
}
