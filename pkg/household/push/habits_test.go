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
	"encoding/json"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household/habits"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// habitSetup gives users 1 and 2 a time zone and a device each, and empties
// the habit tables.
func habitSetup(t *testing.T) time.Time {
	t.Helper()
	s := setup(t)
	for _, table := range []string{"habits", "habit_groups", "habit_entries", "habit_skips", "habit_pauses"} {
		_, err := s.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
	_, err := s.Exec("UPDATE users SET timezone = 'Europe/London' WHERE id IN (1, 2)")
	require.NoError(t, err)
	require.NoError(t, s.Commit())
	addDevice(t, s, user1, "https://push.example.com/1")
	addDevice(t, s, user2, "https://push.example.com/2")
	loc, err := time.LoadLocation("Europe/London")
	require.NoError(t, err)
	// The real current minute: ticks can't be on a day after the real today.
	return time.Now().In(loc).Truncate(time.Minute).Add(20 * time.Second)
}

func makeReminderHabit(t *testing.T, owner *user.User, h *habits.Habit) *habits.Habit {
	t.Helper()
	s := db.NewSession()
	defer s.Close()
	require.NoError(t, h.Create(s, owner))
	require.NoError(t, s.Commit())
	return h
}

// remindedAt lists "user:habit title" for the reminders due at now.
func remindedAt(t *testing.T, now time.Time) []string {
	t.Helper()
	rs, err := dueHabitReminders(now)
	require.NoError(t, err)
	out := []string{}
	for _, r := range rs {
		out = append(out, map[int64]string{1: "1:", 2: "2:"}[r.UserID]+r.Habit.Title)
	}
	return out
}

func TestHabitReminders(t *testing.T) {
	now := habitSetup(t)
	clock := now.Format("15:04")
	later := now.Add(time.Minute).Format("15:04")

	makeReminderHabit(t, user1, &habits.Habit{Title: "Stretch", Kind: habits.KindDaily, ReminderTimes: []string{clock}})
	kitchen := makeReminderHabit(t, user1, &habits.Habit{Title: "Kitchen", Kind: habits.KindDaily, Shared: true, ReminderTimes: []string{later, clock}})
	makeReminderHabit(t, user2, &habits.Habit{Title: "Read", Kind: habits.KindWeekly, Goal: 3, ReminderTimes: []string{later}})
	makeReminderHabit(t, user1, &habits.Habit{Title: "No time", Kind: habits.KindDaily})

	assert.ElementsMatch(t, []string{"1:Stretch", "1:Kitchen", "2:Kitchen"}, remindedAt(t, now),
		"own habits and household routines, at the person's local time")
	assert.ElementsMatch(t, []string{"1:Kitchen", "2:Kitchen", "2:Read"}, remindedAt(t, now.Add(time.Minute)))

	// Done by anyone: a household routine stops reminding everyone.
	s := db.NewSession()
	_, err := habits.AddEntry(s, user2, kitchen.ID, "6f1c3b9e-0f7c-4a51-9a52-3c0d1f2b4a10", now.Format("2006-01-02"))
	require.NoError(t, err)
	require.NoError(t, s.Commit())
	s.Close()
	assert.Equal(t, []string{"1:Stretch"}, remindedAt(t, now))

	// Habit reminders off for user 1.
	s = db.NewSession()
	_, err = SavePreferences(s, user1, &Preferences{Reminders: true, Overdue: true, QuietHours: true, HabitReminders: false})
	require.NoError(t, err)
	require.NoError(t, s.Commit())
	s.Close()
	assert.Empty(t, remindedAt(t, now))
}

func TestHabitReminderMessage(t *testing.T) {
	now := time.Now()
	r := &habits.Reminder{UserID: 1, Habit: &habits.Habit{ID: 5, Title: "Water", Kind: habits.KindDailyCount,
		Progress: &habits.Progress{Today: "2026-11-04", TodayCount: 3, Goal: 8, Streak: 4}}}
	msg := habitMessage(r, now)
	assert.Equal(t, "Water", msg.Title)
	assert.Equal(t, "3 of 8 today so far. Keep your 4-day streak going.", msg.Body)
	assert.Equal(t, []Action{{"done", "+1"}, {"skip", "Skip today"}}, msg.Actions)
	c, err := parseActionToken(msg.ActionToken, now)
	require.NoError(t, err)
	assert.Equal(t, int64(5), c.Habit)
	assert.Equal(t, "2026-11-04", c.Day)
	assert.NotEmpty(t, c.UID)
	assert.Zero(t, c.Task)

	// The token stays small: push services cap the payload at about 4 KB.
	raw, err := json.Marshal(msg)
	require.NoError(t, err)
	assert.Less(t, len(raw), 1000)
}

func habitAction(t *testing.T, action string, userID, habitID int64, day, uid string) (*ActionResult, error) {
	t.Helper()
	s := db.NewSession()
	defer s.Close()
	now := time.Now()
	r, err := DoAction(s, &ActionInput{Token: NewHabitActionToken(userID, habitID, day, uid, now), Action: action}, now)
	if err != nil {
		_ = s.Rollback()
		return nil, err
	}
	require.NoError(t, s.Commit())
	return r, nil
}

func TestHabitActions(t *testing.T) {
	now := habitSetup(t)
	day := now.Format("2006-01-02")
	water := makeReminderHabit(t, user1, &habits.Habit{Title: "Water", Kind: habits.KindDailyCount, Goal: 2, Shared: true})
	mine := makeReminderHabit(t, user1, &habits.Habit{Title: "Stretch", Kind: habits.KindDaily})

	const uid1 = "0b0f2a4e-5d55-4c1e-8a7e-2b8d3c4f5a61"
	const uid2 = "0b0f2a4e-5d55-4c1e-8a7e-2b8d3c4f5a62"
	r, err := habitAction(t, ActionDone, 2, water.ID, day, uid1)
	require.NoError(t, err)
	assert.Equal(t, "done", r.Result, "anyone can tick a household routine")
	r, err = habitAction(t, ActionDone, 2, water.ID, day, uid1)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", r.Result, "the same notification ticks once")
	r, err = habitAction(t, ActionDone, 1, water.ID, day, uid2)
	require.NoError(t, err)
	assert.Equal(t, "done", r.Result, "another notification adds the second glass")
	r, err = habitAction(t, ActionDone, 1, water.ID, day, "0b0f2a4e-5d55-4c1e-8a7e-2b8d3c4f5a63")
	require.NoError(t, err)
	assert.Equal(t, "unchanged", r.Result, "the goal is reached")
	r, err = habitAction(t, ActionSkip, 1, water.ID, day, uid1)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", r.Result, "a done day isn't skipped")

	r, err = habitAction(t, ActionSkip, 1, mine.ID, day, uid1)
	require.NoError(t, err)
	assert.Equal(t, "skipped", r.Result)
	r, err = habitAction(t, ActionSkip, 1, mine.ID, day, uid1)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", r.Result)

	_, err = habitAction(t, ActionDone, 2, mine.ID, day, uid1)
	require.ErrorAs(t, err, &habits.ErrHabitDoesNotExist{}, "someone else's personal habit")
	_, err = habitAction(t, ActionSnooze, 1, mine.ID, day, uid1)
	require.ErrorAs(t, err, &ErrInvalidActionToken{}, "habits have no Snooze")
	_, err = doAction(t, ActionSkip, 1, 1, time.Time{}, time.Now())
	require.ErrorAs(t, err, &ErrInvalidActionToken{}, "tasks have no Skip")
}
