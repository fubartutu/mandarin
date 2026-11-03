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

// Reminders: a habit can have reminder times (HH:MM). At each one, the push
// package (pkg/household/push/habits.go) asks DueReminders who should be
// reminded, and the reminder's Done and Skip buttons come back through
// ReminderTick and ReminderSkip.

import (
	"fmt"
	"slices"
	"time"

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// Reminder is a habit one person should be reminded of now.
type Reminder struct {
	UserID int64
	// Habit is a copy with that person's progress filled in.
	Habit *Habit
}

// DueReminders returns the reminders for the minute `now` is in, for the
// given people: each habit they can see (their own, and every household
// routine) that has a reminder time equal to their local time and is still
// due today, so not done, skipped, paused or on a rest day. Bad habits and
// archived habits never remind.
func DueReminders(s *xorm.Session, users []*user.User, now time.Time) ([]*Reminder, error) {
	if len(users) == 0 {
		return nil, nil
	}
	hs := []*Habit{}
	if err := s.Where("archived = ? AND kind <> ?", false, KindBad).Find(&hs); err != nil {
		return nil, fmt.Errorf("list habits for reminders: %w", err)
	}
	hs = slices.DeleteFunc(hs, func(h *Habit) bool { return len(h.ReminderTimes) == 0 })
	if len(hs) == 0 {
		return nil, nil
	}

	out := []*Reminder{}
	for _, u := range users {
		local := now.In(location(u))
		clock := local.Format("15:04")
		mine := []*Habit{}
		for _, h := range hs {
			if (h.OwnerID == u.ID || h.Shared) && slices.Contains(h.ReminderTimes, clock) {
				// A copy: a household routine's progress differs per person
				// (their own day), so each gets their own.
				c := *h
				mine = append(mine, &c)
			}
		}
		if err := addProgressOn(s, u, civil(local), mine...); err != nil {
			return nil, err
		}
		for _, h := range mine {
			if h.Progress.DueToday {
				out = append(out, &Reminder{UserID: u.ID, Habit: h})
			}
		}
	}
	return out, nil
}

// reminderHabit is the habit with the caller's progress on `day`, for a
// reminder's button.
func reminderHabit(s *xorm.Session, a web.Auth, habitID int64, day string) (*Habit, error) {
	u, h, err := habitFor(s, a, habitID)
	if err != nil {
		return nil, err
	}
	d, err := checkDay(day, u)
	if err != nil {
		return nil, err
	}
	return h, addProgressOn(s, u, d, h)
}

// ReminderTick is a reminder's Done button: one tick on the reminder's day,
// with the uid the reminder was sent with, so a second tap (or a retry)
// changes nothing. When the day's goal was already reached, nothing is added
// either. Reports whether it ticked.
func ReminderTick(s *xorm.Session, a web.Auth, habitID int64, uid, day string) (bool, error) {
	h, err := reminderHabit(s, a, habitID, day)
	if err != nil {
		return false, err
	}
	if h.Progress.DoneToday || slices.Contains(h.Progress.TodayEntries, uid) {
		return false, nil
	}
	if _, err := AddEntry(s, a, habitID, uid, day); err != nil {
		return false, err
	}
	return true, nil
}

// ReminderSkip is a reminder's Skip button: skips the reminder's day (week),
// unless it's done or skipped already. Reports whether it skipped.
func ReminderSkip(s *xorm.Session, a web.Auth, habitID int64, day string) (bool, error) {
	h, err := reminderHabit(s, a, habitID, day)
	if err != nil {
		return false, err
	}
	if h.Progress.DoneToday || h.Progress.Skipped {
		return false, nil
	}
	if _, err := SetSkip(s, a, habitID, day, ""); err != nil {
		return false, err
	}
	return true, nil
}
