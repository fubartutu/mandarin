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

// Package habits is Mandarin's habits and routines. A habit is personal
// (only its owner sees it) or a household routine (shared: everyone sees and
// ticks it). Ticks are separate rows, each with a uid made by the device that
// ticked, so ticks made offline on two phones never overwrite each other.
// The /api/v2 handlers are in pkg/routes/api/v2/household_habits.go.
package habits

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

func init() {
	db.RegisterTables(GetTables())
}

// GetTables returns all structs which are also a table. pkg/migration uses
// this to create them on a fresh install.
func GetTables() []any {
	return []any{
		&Habit{},
		&HabitGroup{},
		&HabitEntry{},
		&HabitSkip{},
		&HabitPause{},
	}
}

// Habit kinds. The kind can't change after a habit is made: its history
// would mean something else.
const (
	KindDaily      = "daily"       // once on each of its weekdays
	KindDailyCount = "daily_count" // a number of times on each of its weekdays
	KindWeekly     = "weekly"      // a number of times a week, any days
	KindBad        = "bad"         // logged when it happens; fewer is better
)

var kinds = []string{KindDaily, KindDailyCount, KindWeekly, KindBad}

const (
	dayFormat       = "2006-01-02"
	everyDay        = "1111111"
	maxTitleLength  = 250
	maxGoal         = 99
	maxReminders    = 8
	permissionAdmin = 2
)

var (
	weekdaysPattern = regexp.MustCompile(`^[01]{7}$`)
	colorPattern    = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
	timePattern     = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
)

// Habit is one habit or household routine.
type Habit struct {
	ID            int64    `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this habit."`
	Title         string   `xorm:"varchar(250) not null" json:"title" minLength:"1" maxLength:"250" doc:"What the habit is, e.g. Stretch or Bins out."`
	Kind          string   `xorm:"varchar(20) not null" json:"kind" enum:"daily,daily_count,weekly,bad" doc:"daily: once on each of its weekdays. daily_count: goal times on each of its weekdays. weekly: goal times a week, any days. bad: logged each time it happens, no goal or streak. Can't be changed after the habit is made."`
	Weekdays      string   `xorm:"varchar(7) not null default '1111111'" json:"weekdays" pattern:"^[01]{7}$" doc:"Which days a daily or daily_count habit is due: seven 1s and 0s, Monday first, e.g. 1010100 for Mon/Wed/Fri. At least one 1. Always 1111111 for weekly and bad habits."`
	Goal          int      `xorm:"int not null default 1" json:"goal" minimum:"0" maximum:"99" doc:"Ticks needed: per day for daily_count, per week for weekly (1–99). Always 1 for daily and bad habits."`
	Color         string   `xorm:"varchar(6) not null default ''" json:"color" doc:"A hex colour without #, e.g. e8590c, or empty for the default."`
	GroupID       int64    `xorm:"bigint not null default 0 index" json:"group_id" minimum:"0" doc:"The routine (habit group) it belongs to, or 0 for none. Must be a group the caller can see."`
	Position      int      `xorm:"int not null default 0" json:"position" doc:"Order in its group (or among ungrouped habits), lowest first."`
	Shared        bool     `xorm:"not null default false" json:"shared" doc:"true for a household routine: every household member sees and ticks it. false for a personal habit only its owner sees. Only the owner can make a household routine personal again."`
	ReminderTimes []string `xorm:"text null json" json:"reminder_times" maxItems:"8" doc:"Times of day (HH:MM, 24-hour, the caller's time zone) to remind if the habit isn't done or skipped yet. At most 8."`
	Archived      bool     `xorm:"not null default false" json:"archived" doc:"An archived habit is kept with its history but left out of lists unless asked for."`
	OwnerID       int64    `xorm:"bigint not null index" json:"owner_id" readOnly:"true" doc:"The user who made the habit."`

	// Paused isn't stored on the row: a pause is a HabitPause span, so past
	// pauses stay in the history.
	Paused bool `xorm:"-" json:"paused" doc:"Whether the habit is paused now (holidays, illness). Days while paused are neither done nor missed. Set to true to pause from today, false to carry on from today."`

	Progress *Progress `xorm:"-" json:"progress" readOnly:"true" doc:"How the habit is going as of today."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When the habit was made. Its history counts from that day (or its first tick, if earlier)."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When the habit was last changed. Ticks and skips don't change it."`

	// ShowArchived comes from the list's query string.
	ShowArchived bool `xorm:"-" json:"-"`
	// stored is the habit as it was before an update, loaded by CanUpdate.
	stored *Habit

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName sets the database table name.
func (*Habit) TableName() string {
	return "habits"
}

// visibleTo is the condition for the habits a user can see.
func visibleTo(userID int64) builder.Cond {
	return builder.Or(builder.Eq{"owner_id": userID}, builder.Eq{"shared": true})
}

// getHabit loads a habit the caller can see; others' personal habits are
// reported as not existing.
func getHabit(s *xorm.Session, id, userID int64) (*Habit, error) {
	h := &Habit{}
	exists, err := s.Where(builder.Eq{"id": id}.And(visibleTo(userID))).Get(h)
	if err != nil {
		return nil, fmt.Errorf("get habit %d: %w", id, err)
	}
	if !exists {
		return nil, ErrHabitDoesNotExist{HabitID: id}
	}
	return h, nil
}

// location is the caller's time zone: their own setting, else the server's.
func location(u *user.User) *time.Location {
	if u != nil && u.Timezone != "" {
		if l, err := time.LoadLocation(u.Timezone); err == nil {
			return l
		}
	}
	return config.GetTimeZone()
}

// today is the caller's calendar day.
func today(u *user.User) time.Time {
	return dayAt(time.Now(), u)
}

// dayAt is the calendar day it is for u at the moment now.
func dayAt(now time.Time, u *user.User) time.Time {
	return civil(now.In(location(u)))
}

// CanRead lets a member read their own habits and every household routine.
func (h *Habit) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if err := household.RequireMember(a); err != nil {
		return false, 0, err
	}
	stored, err := getHabit(s, h.ID, a.GetID())
	if err != nil {
		return false, 0, err
	}
	*h = *stored
	return true, permissionAdmin, nil
}

// CanCreate lets every household member make habits.
func (h *Habit) CanCreate(_ *xorm.Session, a web.Auth) (bool, error) {
	return household.IsMember(a), nil
}

// CanUpdate lets a member change their own habits and every household
// routine. It keeps the stored habit for Update to compare against.
func (h *Habit) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	stored, err := canWrite(s, h.ID, a)
	if err != nil {
		return false, err
	}
	h.stored = stored
	return true, nil
}

// CanDelete lets a member delete their own habits and every household routine.
func (h *Habit) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	_, err := canWrite(s, h.ID, a)
	return err == nil, err
}

func canWrite(s *xorm.Session, id int64, a web.Auth) (*Habit, error) {
	if err := household.RequireMember(a); err != nil {
		return nil, err
	}
	return getHabit(s, id, a.GetID())
}

// normalize checks and tidies the fields a caller sets. keptGroupID is the
// group the habit is already in: a household routine can sit in its owner's
// personal group, and others must still be able to change the routine.
func (h *Habit) normalize(s *xorm.Session, userID, keptGroupID int64) error {
	h.Title = strings.TrimSpace(h.Title)
	if n := utf8.RuneCountInString(h.Title); n < 1 || n > maxTitleLength {
		return ErrInvalidHabit{Reason: "The title must be 1–250 characters."}
	}
	if !slices.Contains(kinds, h.Kind) {
		return ErrInvalidHabit{Reason: "The kind must be daily, daily_count, weekly or bad."}
	}

	switch h.Kind {
	case KindDaily, KindDailyCount:
		if h.Weekdays == "" {
			h.Weekdays = everyDay
		}
		if !weekdaysPattern.MatchString(h.Weekdays) || !strings.Contains(h.Weekdays, "1") {
			return ErrInvalidHabit{Reason: "The weekdays must be seven 1s and 0s, Monday first, with at least one 1."}
		}
	default:
		h.Weekdays = everyDay
	}

	switch h.Kind {
	case KindDailyCount, KindWeekly:
		if h.Goal == 0 {
			h.Goal = 1
		}
		if h.Goal < 1 || h.Goal > maxGoal {
			return ErrInvalidHabit{Reason: "The goal must be 1–99."}
		}
	default:
		h.Goal = 1
	}

	h.Color = strings.TrimPrefix(strings.TrimSpace(h.Color), "#")
	if h.Color != "" && !colorPattern.MatchString(h.Color) {
		return ErrInvalidHabit{Reason: "The colour must be six hex digits, e.g. e8590c."}
	}
	h.Color = strings.ToLower(h.Color)

	times := []string{}
	for _, t := range h.ReminderTimes {
		t = strings.TrimSpace(t)
		if !timePattern.MatchString(t) {
			return ErrInvalidHabit{Reason: "Reminder times must be written HH:MM, e.g. 07:30."}
		}
		if !slices.Contains(times, t) {
			times = append(times, t)
		}
	}
	if len(times) > maxReminders {
		return ErrInvalidHabit{Reason: "A habit can have at most 8 reminder times."}
	}
	slices.Sort(times)
	h.ReminderTimes = times

	if h.GroupID != 0 && h.GroupID != keptGroupID {
		if _, err := getGroup(s, h.GroupID, userID); err != nil {
			return err
		}
	}
	return nil
}

// Create makes a habit owned by the caller, last in its group.
func (h *Habit) Create(s *xorm.Session, a web.Auth) error {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	if err := h.normalize(s, u.ID, 0); err != nil {
		return err
	}
	last := &Habit{}
	has, err := s.Where("group_id = ?", h.GroupID).Desc("position").Get(last)
	if err != nil {
		return fmt.Errorf("get last habit: %w", err)
	}
	h.Position = 0
	if has {
		h.Position = last.Position + 1
	}
	h.ID = 0
	h.OwnerID = u.ID
	h.Archived = false
	paused := h.Paused
	h.Paused = false
	if _, err := s.Insert(h); err != nil {
		return fmt.Errorf("insert habit: %w", err)
	}
	if paused {
		if err := setPaused(s, h, true, today(u)); err != nil {
			return err
		}
	}
	return addProgress(s, u, h)
}

// ReadOne fills in the habit's progress; CanRead loaded the habit.
func (h *Habit) ReadOne(s *xorm.Session, a web.Auth) error {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	return addProgress(s, u, h)
}

// ReadAll returns the caller's habits and the household routines, with
// progress, by group then position. Archived ones only with ShowArchived.
// Everything fits on one page.
func (h *Habit) ReadAll(s *xorm.Session, a web.Auth, _ string, _ int, _ int) (any, int, int64, error) {
	if err := household.RequireMember(a); err != nil {
		return nil, 0, 0, err
	}
	u, err := user.GetFromAuth(a)
	if err != nil {
		return nil, 0, 0, err
	}
	cond := visibleTo(u.ID)
	if !h.ShowArchived {
		cond = cond.And(builder.Eq{"archived": false})
	}
	all := []*Habit{}
	if err := s.Where(cond).OrderBy("group_id ASC, position ASC, id ASC").Find(&all); err != nil {
		return nil, 0, 0, fmt.Errorf("list habits: %w", err)
	}
	if err := addProgress(s, u, all...); err != nil {
		return nil, 0, 0, err
	}
	return all, len(all), int64(len(all)), nil
}

// Update replaces the habit's editable fields. The kind and owner stay.
func (h *Habit) Update(s *xorm.Session, a web.Auth) error {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	stored := h.stored
	if stored == nil {
		if stored, err = getHabit(s, h.ID, u.ID); err != nil {
			return err
		}
	}
	if h.Kind != "" && h.Kind != stored.Kind {
		return ErrInvalidHabit{Reason: "The kind of a habit can't be changed."}
	}
	h.Kind = stored.Kind
	if stored.Shared && !h.Shared && stored.OwnerID != u.ID {
		return ErrOnlyOwnerCanUnshare{}
	}
	if err := h.normalize(s, u.ID, stored.GroupID); err != nil {
		return err
	}
	h.OwnerID = stored.OwnerID
	h.Created = stored.Created

	wasPaused, err := isPaused(s, h.ID)
	if err != nil {
		return err
	}
	if wasPaused != h.Paused {
		if err := setPaused(s, h, h.Paused, today(u)); err != nil {
			return err
		}
	}

	// Writing an unchanged row would still bump updated (and take SQLite's
	// write lock), so it's skipped.
	if !sameFields(stored, h) {
		cols := []string{"title", "weekdays", "goal", "color", "group_id", "position", "shared", "reminder_times", "archived"}
		if _, err := s.ID(h.ID).Cols(cols...).Update(h); err != nil {
			return fmt.Errorf("update habit %d: %w", h.ID, err)
		}
	} else {
		h.Updated = stored.Updated
	}
	return addProgress(s, u, h)
}

func sameFields(a, b *Habit) bool {
	return a.Title == b.Title && a.Weekdays == b.Weekdays && a.Goal == b.Goal &&
		a.Color == b.Color && a.GroupID == b.GroupID && a.Position == b.Position &&
		a.Shared == b.Shared && slices.Equal(a.ReminderTimes, b.ReminderTimes) &&
		a.Archived == b.Archived
}

// Delete removes the habit with its ticks, skips and pauses.
func (h *Habit) Delete(s *xorm.Session, _ web.Auth) error {
	for _, bean := range []any{&HabitEntry{}, &HabitSkip{}, &HabitPause{}} {
		if _, err := s.Where("habit_id = ?", h.ID).Delete(bean); err != nil {
			return fmt.Errorf("delete records of habit %d: %w", h.ID, err)
		}
	}
	if _, err := s.ID(h.ID).Delete(&Habit{}); err != nil {
		return fmt.Errorf("delete habit %d: %w", h.ID, err)
	}
	return nil
}

// addProgress loads the habits' history and fills in Paused and Progress.
func addProgress(s *xorm.Session, u *user.User, hs ...*Habit) error {
	return addProgressOn(s, u, today(u), hs...)
}

// addProgressOn is addProgress as of the user's calendar day `day`.
func addProgressOn(s *xorm.Session, u *user.User, day time.Time, hs ...*Habit) error {
	if len(hs) == 0 {
		return nil
	}
	ids := make([]int64, len(hs))
	for i, h := range hs {
		ids[i] = h.ID
	}
	hists, err := loadHistories(s, ids...)
	if err != nil {
		return err
	}
	loc := location(u)
	for _, h := range hs {
		hist := hists[h.ID]
		h.Paused = hist.paused(day.Format(dayFormat))
		h.Progress = computeProgress(h, hist, startDay(h, hist, loc), day)
		h.Progress.TodayEntries, err = currentEntries(s, h, day)
		if err != nil {
			return err
		}
	}
	return nil
}

// startDay is the first day a habit counts from: the day it was made, in
// the caller's time zone, or its first tick if that is earlier.
func startDay(h *Habit, hist *history, loc *time.Location) time.Time {
	start := civil(h.Created.In(loc))
	if h.Created.IsZero() {
		start = civil(time.Now().In(loc))
	}
	for day := range hist.counts {
		if d, err := parseCivil(day); err == nil && d.Before(start) {
			start = d
		}
	}
	return start
}
