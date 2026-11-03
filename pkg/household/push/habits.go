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

// Habit reminders: every minute, the habits whose reminder time it is (in each
// person's own time zone) and that are still to do today are pushed to the
// people who see them, with Done and Skip buttons. The habits package decides
// who is due (habits.DueReminders); this file only sends.

import (
	"context"
	"fmt"
	"time"

	"code.vikunja.io/api/pkg/cron"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household/habits"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/user"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

func scheduleHabitReminders() {
	if err := cron.Schedule("* * * * *", func() { sendHabitReminders(time.Now()) }); err != nil {
		log.Errorf("Could not schedule habit reminders: %s", err)
	}
}

// habitReminderUsers are the people who could get a habit reminder: active,
// not bots, with at least one push device and habit reminders on.
func habitReminderUsers(s *xorm.Session) ([]*user.User, error) {
	ids := []int64{}
	if err := s.Table("push_devices").Distinct("user_id").Find(&ids); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	users := []*user.User{}
	if err := s.In("id", ids).Where("status = ?", user.StatusActive).Find(&users); err != nil {
		return nil, err
	}
	out := []*user.User{}
	for _, u := range users {
		if u.IsBot() {
			continue
		}
		p, err := preferencesFor(s, u.ID)
		if err != nil {
			return nil, err
		}
		if p.HabitReminders {
			out = append(out, u)
		}
	}
	return out, nil
}

// dueHabitReminders reads who to remind at now, in one read-only session.
func dueHabitReminders(now time.Time) ([]*habits.Reminder, error) {
	s := db.NewSession()
	defer s.Close()
	users, err := habitReminderUsers(s)
	if err != nil {
		return nil, err
	}
	reminders, err := habits.DueReminders(s, users, now)
	if err != nil {
		return nil, err
	}
	return reminders, s.Commit()
}

func sendHabitReminders(now time.Time) {
	reminders, err := dueHabitReminders(now)
	if err != nil {
		log.Errorf("Habit reminders: %s", err)
		return
	}
	for _, r := range reminders {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		if _, err := Send(ctx, r.UserID, habitMessage(r, now)); err != nil {
			log.Errorf("Habit reminder of habit %d to user %d failed: %s", r.Habit.ID, r.UserID, err)
		}
		cancel()
	}
}

// habitMessage is the push for one reminder, with its Done and Skip buttons.
func habitMessage(r *habits.Reminder, now time.Time) *Message {
	h, p := r.Habit, r.Habit.Progress
	body := "Still to do today."
	switch {
	case h.Kind == habits.KindWeekly:
		body = fmt.Sprintf("%d of %d this week so far.", p.TodayCount, p.Goal)
	case p.Goal > 1:
		body = fmt.Sprintf("%d of %d today so far.", p.TodayCount, p.Goal)
	}
	if p.Streak > 1 {
		unit := "day"
		if h.Kind == habits.KindWeekly {
			unit = "week"
		}
		body += fmt.Sprintf(" Keep your %d-%s streak going.", p.Streak, unit)
	}

	done, skip := "Done", "Skip today"
	if p.Goal > 1 {
		done = "+1"
	}
	if h.Kind == habits.KindWeekly {
		skip = "Skip this week"
	}
	msg := &Message{
		Title: h.Title,
		Body:  body,
		URL:   "/habits",
		Tag:   fmt.Sprintf("habit-%d", h.ID),
		// A reminder still waiting hours later is noise.
		ttl: 3 * time.Hour,
	}
	if token := NewHabitActionToken(r.UserID, h.ID, p.Today, uuid.NewString(), now); token != "" {
		msg.Actions = []Action{{Action: ActionDone, Title: done}, {Action: ActionSkip, Title: skip}}
		msg.ActionToken = token
	}
	return msg
}

// doHabitAction carries out a habit reminder's button for u.
func doHabitAction(s *xorm.Session, u *user.User, c *actionClaims, action string) (*ActionResult, error) {
	var changed bool
	var err error
	result := "done"
	switch action {
	case ActionDone:
		changed, err = habits.ReminderTick(s, u, c.Habit, c.UID, c.Day)
	case ActionSkip:
		changed, err = habits.ReminderSkip(s, u, c.Habit, c.Day)
		result = "skipped"
	default:
		return nil, ErrInvalidActionToken{} // habit reminders have no Snooze button
	}
	if err != nil {
		return nil, err
	}
	if !changed {
		result = "unchanged"
	}
	return &ActionResult{Result: result}, nil
}
