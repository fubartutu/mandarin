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
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

// This file has what's recorded against a habit: ticks, skips and pauses.
// Ticks and skips are actions on a habit rather than resources of their
// own, so they are plain functions the v2 handlers (and later the offline
// sync engine) call, each checking the caller may see the habit first.

const (
	maxReasonLength = 250
	// earliestDay stops a typo like 0026-11-03 creating a 2000-year history.
	earliestDay = "2000-01-01"
)

// HabitEntry is one tick of a habit on one day (for a bad habit: one time
// it happened). A daily_count habit gets one row per tick.
type HabitEntry struct {
	ID      int64     `xorm:"bigint autoincr not null unique pk" json:"-"`
	UID     string    `xorm:"'uid' varchar(36) not null unique" json:"uid" doc:"The tick's id across devices: a UUID made by the app that ticked. Sending the same uid again changes nothing."`
	HabitID int64     `xorm:"bigint not null index" json:"habit_id" readOnly:"true" doc:"The habit ticked."`
	Day     string    `xorm:"varchar(10) not null index" json:"day" format:"date" doc:"The day it counts for, YYYY-MM-DD. Today or earlier."`
	UserID  int64     `xorm:"bigint not null" json:"user_id" readOnly:"true" doc:"Who ticked, e.g. who put the bins out for a household routine."`
	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When the tick reached the server."`
}

// TableName sets the database table name.
func (*HabitEntry) TableName() string {
	return "habit_entries"
}

// HabitSkip marks a day (for a weekly habit, a week) as skipped on purpose:
// neither done nor missed.
type HabitSkip struct {
	ID      int64     `xorm:"bigint autoincr not null unique pk" json:"-"`
	HabitID int64     `xorm:"bigint not null unique(habit_day)" json:"habit_id" readOnly:"true" doc:"The habit skipped."`
	Day     string    `xorm:"varchar(10) not null unique(habit_day)" json:"day" format:"date" readOnly:"true" doc:"The day skipped; for a weekly habit the Monday of the week."`
	Reason  string    `xorm:"varchar(250) not null default ''" json:"reason" maxLength:"250" doc:"Why, optional: ill, away, rest day."`
	UserID  int64     `xorm:"bigint not null" json:"user_id" readOnly:"true" doc:"Who skipped it."`
	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When it was skipped."`
}

// TableName sets the database table name.
func (*HabitSkip) TableName() string {
	return "habit_skips"
}

// HabitPause is a stretch of days a habit was paused, both ends included.
// Until is empty while it's still paused. Past pauses are kept so their
// days stay "paused" in the history rather than turning into misses.
type HabitPause struct {
	ID      int64     `xorm:"bigint autoincr not null unique pk"`
	HabitID int64     `xorm:"bigint not null index"`
	From    string    `xorm:"'from_day' varchar(10) not null"`
	Until   string    `xorm:"'until_day' varchar(10) not null default ''"`
	Created time.Time `xorm:"created not null"`
}

// TableName sets the database table name.
func (*HabitPause) TableName() string {
	return "habit_pauses"
}

// History is a habit's grid between two days, with the ticks and skips.
type History struct {
	From    string        `json:"from" format:"date" doc:"The first day asked for."`
	To      string        `json:"to" format:"date" doc:"The last day asked for (no later than today)."`
	Periods []*Period     `json:"periods" doc:"One per day (per week, Monday first, for weekly habits) from the first to the last day, oldest first."`
	Entries []*HabitEntry `json:"entries" doc:"The ticks in the range, oldest first."`
	Skips   []*HabitSkip  `json:"skips" doc:"The skips in the range."`
}

// maxHistoryDays caps a history request at a bit over a year.
const maxHistoryDays = 400

// habitFor checks the caller is a member who can see the habit, and returns
// the caller and the habit.
func habitFor(s *xorm.Session, a web.Auth, habitID int64) (*user.User, *Habit, error) {
	if err := household.RequireMember(a); err != nil {
		return nil, nil, err
	}
	u, err := user.GetFromAuth(a)
	if err != nil {
		return nil, nil, err
	}
	h, err := getHabit(s, habitID, u.ID)
	if err != nil {
		return nil, nil, err
	}
	return u, h, nil
}

// checkDay parses a day the caller sent: a real date from 2000 to today.
func checkDay(day string, u *user.User) (time.Time, error) {
	d, err := parseCivil(day)
	if err != nil || day < earliestDay {
		return time.Time{}, ErrInvalidHabit{Reason: "The day must be a date written YYYY-MM-DD."}
	}
	if d.After(today(u)) {
		return time.Time{}, ErrInvalidHabit{Reason: "A day in the future can't be ticked or skipped yet."}
	}
	return d, nil
}

// AddEntry ticks a habit once on a day and returns the habit with its new
// progress. A uid that was already sent for this habit changes nothing, so
// a phone can safely send a tick again after a lost answer.
func AddEntry(s *xorm.Session, a web.Auth, habitID int64, uid, day string) (*Habit, error) {
	u, h, err := habitFor(s, a, habitID)
	if err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(uid); err != nil {
		return nil, ErrInvalidHabit{Reason: "The uid must be a UUID."}
	}
	if _, err := checkDay(day, u); err != nil {
		return nil, err
	}
	existing := &HabitEntry{}
	has, err := s.Where("uid = ?", uid).Get(existing)
	if err != nil {
		return nil, fmt.Errorf("get habit entry %s: %w", uid, err)
	}
	switch {
	case has && existing.HabitID != h.ID:
		return nil, ErrInvalidHabit{Reason: "This uid belongs to another habit's tick."}
	case !has:
		e := &HabitEntry{UID: uid, HabitID: h.ID, Day: day, UserID: u.ID}
		if _, err := s.Insert(e); err != nil {
			return nil, fmt.Errorf("insert habit entry: %w", err)
		}
	}
	return h, addProgress(s, u, h)
}

// RemoveEntry takes a tick off. A uid that's already gone (deleted on
// another phone) changes nothing.
func RemoveEntry(s *xorm.Session, a web.Auth, habitID int64, uid string) (*Habit, error) {
	u, h, err := habitFor(s, a, habitID)
	if err != nil {
		return nil, err
	}
	if _, err := s.Where("uid = ? AND habit_id = ?", uid, h.ID).Delete(&HabitEntry{}); err != nil {
		return nil, fmt.Errorf("delete habit entry %s: %w", uid, err)
	}
	return h, addProgress(s, u, h)
}

// SetSkip skips a day (for weekly habits, the week of that day) with an
// optional reason. Skipping again only changes the reason.
func SetSkip(s *xorm.Session, a web.Auth, habitID int64, day, reason string) (*Habit, error) {
	u, h, err := habitFor(s, a, habitID)
	if err != nil {
		return nil, err
	}
	d, err := checkDay(day, u)
	if err != nil {
		return nil, err
	}
	if h.Kind == KindBad {
		return nil, ErrInvalidHabit{Reason: "A bad habit can't be skipped."}
	}
	if h.Kind == KindWeekly {
		day = weekStart(d).Format(dayFormat)
	}
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > maxReasonLength {
		return nil, ErrInvalidHabit{Reason: "The reason can be at most 250 characters."}
	}
	existing := &HabitSkip{}
	has, err := s.Where("habit_id = ? AND day = ?", h.ID, day).Get(existing)
	if err != nil {
		return nil, fmt.Errorf("get habit skip: %w", err)
	}
	switch {
	case !has:
		skip := &HabitSkip{HabitID: h.ID, Day: day, Reason: reason, UserID: u.ID}
		if _, err := s.Insert(skip); err != nil {
			return nil, fmt.Errorf("insert habit skip: %w", err)
		}
	case existing.Reason != reason:
		existing.Reason = reason
		if _, err := s.ID(existing.ID).Cols("reason").Update(existing); err != nil {
			return nil, fmt.Errorf("update habit skip: %w", err)
		}
	}
	return h, addProgress(s, u, h)
}

// RemoveSkip un-skips a day (for weekly habits, the week of that day).
func RemoveSkip(s *xorm.Session, a web.Auth, habitID int64, day string) (*Habit, error) {
	u, h, err := habitFor(s, a, habitID)
	if err != nil {
		return nil, err
	}
	d, err := parseCivil(day)
	if err != nil {
		return nil, ErrInvalidHabit{Reason: "The day must be a date written YYYY-MM-DD."}
	}
	if h.Kind == KindWeekly {
		day = weekStart(d).Format(dayFormat)
	}
	if _, err := s.Where("habit_id = ? AND day = ?", h.ID, day).Delete(&HabitSkip{}); err != nil {
		return nil, fmt.Errorf("delete habit skip: %w", err)
	}
	return h, addProgress(s, u, h)
}

// GetHistory returns a habit's grid, ticks and skips from one day to
// another (clipped to today).
func GetHistory(s *xorm.Session, a web.Auth, habitID int64, from, to string) (*History, error) {
	u, h, err := habitFor(s, a, habitID)
	if err != nil {
		return nil, err
	}
	first, err1 := parseCivil(from)
	last, err2 := parseCivil(to)
	if err1 != nil || err2 != nil || from < earliestDay {
		return nil, ErrInvalidHabit{Reason: "from and to must be dates written YYYY-MM-DD."}
	}
	if last.Before(first) || last.Sub(first) > maxHistoryDays*24*time.Hour {
		return nil, ErrInvalidHabit{Reason: "The range must run forwards and cover at most 400 days."}
	}
	day := today(u)
	if last.After(day) {
		last = day
		to = day.Format(dayFormat)
	}

	hists, err := loadHistories(s, h.ID)
	if err != nil {
		return nil, err
	}
	hist := hists[h.ID]
	out := &History{From: from, To: to, Entries: []*HabitEntry{}, Skips: []*HabitSkip{}}
	if first.After(last) {
		out.Periods = []*Period{}
		return out, nil
	}
	out.Periods = gridPeriods(h, hist, startDay(h, hist, location(u)), day, first, last)

	// A weekly habit's skips sit on Mondays, which can be before from.
	skipFrom := from
	if h.Kind == KindWeekly {
		skipFrom = weekStart(first).Format(dayFormat)
	}
	if err := s.Where("habit_id = ? AND day >= ? AND day <= ?", h.ID, from, to).
		OrderBy("day ASC, id ASC").Find(&out.Entries); err != nil {
		return nil, fmt.Errorf("list habit entries: %w", err)
	}
	if err := s.Where("habit_id = ? AND day >= ? AND day <= ?", h.ID, skipFrom, to).
		OrderBy("day ASC").Find(&out.Skips); err != nil {
		return nil, fmt.Errorf("list habit skips: %w", err)
	}
	return out, nil
}

// loadHistories reads the ticks per day, skips and pauses of the habits.
// Every id gets a history, empty if nothing was recorded.
func loadHistories(s *xorm.Session, ids ...int64) (map[int64]*history, error) {
	out := make(map[int64]*history, len(ids))
	for _, id := range ids {
		out[id] = newHistory()
	}

	type dayCount struct {
		HabitID int64
		Day     string
		N       int
	}
	rows := []*dayCount{}
	if err := s.Table("habit_entries").Select("habit_id, day, COUNT(*) AS n").
		In("habit_id", ids).GroupBy("habit_id, day").Find(&rows); err != nil {
		return nil, fmt.Errorf("count habit entries: %w", err)
	}
	for _, r := range rows {
		out[r.HabitID].counts[r.Day] = r.N
	}

	skips := []*HabitSkip{}
	if err := s.In("habit_id", ids).Find(&skips); err != nil {
		return nil, fmt.Errorf("list habit skips: %w", err)
	}
	for _, sk := range skips {
		out[sk.HabitID].skips[sk.Day] = true
	}

	pauses := []*HabitPause{}
	if err := s.In("habit_id", ids).Find(&pauses); err != nil {
		return nil, fmt.Errorf("list habit pauses: %w", err)
	}
	for _, p := range pauses {
		out[p.HabitID].pauses = append(out[p.HabitID].pauses, pauseSpan{from: p.From, until: p.Until})
	}
	return out, nil
}

// currentEntries returns the uids of the habit's ticks today (this week for
// weekly habits), oldest first.
func currentEntries(s *xorm.Session, h *Habit, day time.Time) ([]string, error) {
	first := day
	if h.Kind == KindWeekly {
		first = weekStart(day)
	}
	entries := []*HabitEntry{}
	if err := s.Where("habit_id = ? AND day >= ? AND day <= ?", h.ID, first.Format(dayFormat), day.Format(dayFormat)).
		OrderBy("day ASC, id ASC").Cols("uid").Find(&entries); err != nil {
		return nil, fmt.Errorf("list today's habit entries: %w", err)
	}
	uids := make([]string, len(entries))
	for i, e := range entries {
		uids[i] = e.UID
	}
	return uids, nil
}

func openPause(s *xorm.Session, habitID int64) (*HabitPause, bool, error) {
	p := &HabitPause{}
	has, err := s.Where("habit_id = ? AND until_day = ''", habitID).Get(p)
	if err != nil {
		return nil, false, fmt.Errorf("get open pause of habit %d: %w", habitID, err)
	}
	return p, has, nil
}

func isPaused(s *xorm.Session, habitID int64) (bool, error) {
	_, has, err := openPause(s, habitID)
	return has, err
}

// setPaused pauses a habit from today, or carries on from today: the open
// pause then ends yesterday. A pause started and ended on the same day is
// removed, as if it never happened.
func setPaused(s *xorm.Session, h *Habit, paused bool, day time.Time) error {
	p, has, err := openPause(s, h.ID)
	if err != nil {
		return err
	}
	switch {
	case paused && !has:
		np := &HabitPause{HabitID: h.ID, From: day.Format(dayFormat)}
		if _, err := s.Insert(np); err != nil {
			return fmt.Errorf("insert habit pause: %w", err)
		}
	case !paused && has:
		until := addDays(day, -1).Format(dayFormat)
		if until < p.From {
			if _, err := s.ID(p.ID).Delete(&HabitPause{}); err != nil {
				return fmt.Errorf("delete habit pause: %w", err)
			}
			return nil
		}
		p.Until = until
		if _, err := s.ID(p.ID).Cols("until_day").Update(p); err != nil {
			return fmt.Errorf("end habit pause: %w", err)
		}
	}
	return nil
}
