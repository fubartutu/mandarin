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

// Snooze: the reminder comes back in an hour, for the person who snoozed it
// only. It's kept here rather than as a task reminder, so it doesn't clutter
// the task (or CalDAV) and doesn't wake everyone else on the task.

import (
	"context"
	"time"

	"code.vikunja.io/api/pkg/cron"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"xorm.io/xorm"
)

// pushSnooze is a snoozed reminder: user UserID gets task TaskID's reminder
// again at At. One per user and task: snoozing again moves it.
type pushSnooze struct {
	ID     int64     `xorm:"bigint autoincr not null unique pk"`
	UserID int64     `xorm:"bigint not null unique(user_task)"`
	TaskID int64     `xorm:"bigint not null unique(user_task)"`
	At     time.Time `xorm:"datetime not null index"`
}

func (*pushSnooze) TableName() string {
	return "push_snoozes"
}

// snooze saves (or moves) the user's snooze for the task.
func snooze(s *xorm.Session, userID, taskID int64, at time.Time) error {
	existing := &pushSnooze{}
	has, err := s.Where("user_id = ? AND task_id = ?", userID, taskID).Get(existing)
	if err != nil {
		return err
	}
	if !has {
		_, err = s.Insert(&pushSnooze{UserID: userID, TaskID: taskID, At: at})
		return err
	}
	if existing.At.Equal(at) {
		return nil // SQLite: don't write a row that hasn't changed
	}
	_, err = s.ID(existing.ID).Cols("at").Update(&pushSnooze{At: at})
	return err
}

func scheduleSnoozes() {
	if err := cron.Schedule("* * * * *", func() { sendSnoozedReminders(time.Now()) }); err != nil {
		log.Errorf("Could not schedule snoozed reminders: %s", err)
	}
}

// sendSnoozedReminders sends every snooze that's due, then forgets it.
func sendSnoozedReminders(now time.Time) {
	s := db.NewSession()
	due := []*pushSnooze{}
	err := s.Where("at <= ?", now).OrderBy("at").Find(&due)
	_ = s.Commit()
	s.Close()
	if err != nil {
		log.Errorf("Snoozed reminders: %s", err)
		return
	}
	for _, sn := range due {
		msg, err := snoozedMessage(sn)
		if err != nil {
			log.Errorf("Snoozed reminder of task %d for user %d: %s", sn.TaskID, sn.UserID, err)
			continue // the row stays; next minute tries again
		}
		if msg == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		if _, err := Send(ctx, sn.UserID, msg); err != nil {
			log.Errorf("Snoozed reminder to user %d failed: %s", sn.UserID, err)
		}
		cancel()
	}
}

// snoozedMessage removes the snooze and returns the reminder to send, or nil
// when there's nothing to send any more: the task is done or gone, the user
// can't see it, or turned reminders off.
func snoozedMessage(sn *pushSnooze) (*Message, error) {
	// Removed first, in its own commit: a failed send isn't retried every
	// minute, as Vikunja's own reminders aren't.
	del := db.NewSession()
	_, err := del.ID(sn.ID).Delete(&pushSnooze{})
	if err == nil {
		err = del.Commit()
	} else {
		_ = del.Rollback()
	}
	del.Close()
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()
	u, err := user.GetUserByIDForAuth(s, sn.UserID)
	if err != nil {
		return nil, nil //nolint:nilerr // the user is gone or disabled: nothing to send
	}
	task, err := models.GetTaskByIDSimple(s, sn.TaskID)
	if models.IsErrTaskDoesNotExist(err) || err == nil && task.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if can, _, err := task.CanRead(s, u); err != nil || !can {
		return nil, err
	}
	prefs, err := preferencesFor(s, u.ID)
	if err != nil {
		return nil, err
	}
	if !prefs.Reminders {
		return nil, nil
	}
	project, err := models.GetProjectSimpleByID(s, task.ProjectID)
	if err != nil {
		project = nil // the title is a nicety
	}
	msg := reminderMessage(&task, project, u)
	addTaskActions(msg, u.ID, time.Now())
	return msg, s.Commit()
}
