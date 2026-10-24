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

package menu

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// Meals, in the order they're eaten. The app only shows dinner for now.
const (
	MealBreakfast = "breakfast"
	MealLunch     = "lunch"
	MealDinner    = "dinner"
)

var meals = []string{MealBreakfast, MealLunch, MealDinner}

const (
	dayFormat     = "2006-01-02"
	maxNoteLength = 250
	// maxRangeDays caps a listing: a week is 7 days, a month view would be 42.
	maxRangeDays = 62
)

// MenuEntry is one thing planned for one meal on one day. Today every entry
// is a free-text note ("leftovers", "takeaway"); a planned recipe will be an
// entry with a recipe id and its own servings. A meal slot (day + meal) can
// hold several entries, kept in Position order.
type MenuEntry struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	ID       int64  `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this entry."`
	Day      string `xorm:"varchar(10) not null index" json:"day" format:"date" doc:"The calendar day it's planned for, as YYYY-MM-DD (no time or timezone)."`
	Meal     string `xorm:"varchar(20) not null" json:"meal" enum:"breakfast,lunch,dinner" doc:"Which meal of the day."`
	Position int    `xorm:"int not null default 0" json:"position" readOnly:"true" doc:"Order within the day's meal, lowest first. New entries go last."`
	Note     string `xorm:"varchar(250) not null default ''" json:"note" maxLength:"250" doc:"The free-text note, e.g. leftovers or takeaway."`

	CreatedByID int64     `xorm:"bigint not null default 0" json:"-"`
	Created     time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When this entry was added."`
	Updated     time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this entry was last changed."`

	// The range a listing covers, from the query string.
	From string `xorm:"-" json:"-"`
	To   string `xorm:"-" json:"-"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName sets the database table name.
func (*MenuEntry) TableName() string {
	return "menu_entries"
}

func getEntry(s *xorm.Session, id int64) (*MenuEntry, error) {
	e := &MenuEntry{}
	exists, err := s.Where("id = ?", id).Get(e)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrMenuEntryDoesNotExist{EntryID: id}
	}
	return e, nil
}

// CanCreate lets every household member plan meals.
func (e *MenuEntry) CanCreate(_ *xorm.Session, a web.Auth) (bool, error) {
	return household.IsMember(a), nil
}

// CanDelete lets every household member delete any entry.
func (e *MenuEntry) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return canWrite(s, e.ID, a)
}

func canWrite(s *xorm.Session, id int64, a web.Auth) (bool, error) {
	if err := requireMember(a); err != nil {
		return false, err
	}
	if _, err := getEntry(s, id); err != nil {
		return false, err
	}
	return true, nil
}

// parseDay checks a YYYY-MM-DD day. time.Parse also rejects 2026-02-30.
func parseDay(day string) (time.Time, error) {
	t, err := time.Parse(dayFormat, day)
	if err != nil {
		return time.Time{}, ErrInvalidMenuEntry{Reason: "The day must be a date written YYYY-MM-DD."}
	}
	return t, nil
}

func (e *MenuEntry) normalize() error {
	if _, err := parseDay(e.Day); err != nil {
		return err
	}
	if !slices.Contains(meals, e.Meal) {
		return ErrInvalidMenuEntry{Reason: "The meal must be breakfast, lunch or dinner."}
	}
	e.Note = strings.TrimSpace(e.Note)
	if n := utf8.RuneCountInString(e.Note); n < 1 || n > maxNoteLength {
		return ErrInvalidMenuEntry{Reason: "The note must be 1–250 characters."}
	}
	return nil
}

// Create adds an entry at the end of its day's meal.
func (e *MenuEntry) Create(s *xorm.Session, a web.Auth) error {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	if err := e.normalize(); err != nil {
		return err
	}
	last := &MenuEntry{}
	has, err := s.Where("day = ? AND meal = ?", e.Day, e.Meal).Desc("position").Get(last)
	if err != nil {
		return err
	}
	e.Position = 0
	if has {
		e.Position = last.Position + 1
	}
	e.ID = 0
	e.CreatedByID = u.ID
	_, err = s.Insert(e)
	return err
}

// ReadAll returns the entries from From to To (both included), by day, then
// meal (breakfast first), then position. Paging is allowed but a week fits
// on one page.
func (e *MenuEntry) ReadAll(s *xorm.Session, a web.Auth, _ string, page int, perPage int) (any, int, int64, error) {
	if err := requireMember(a); err != nil {
		return nil, 0, 0, err
	}
	from, err := parseDay(e.From)
	if err != nil {
		return nil, 0, 0, err
	}
	to, err := parseDay(e.To)
	if err != nil {
		return nil, 0, 0, err
	}
	if to.Before(from) || to.Sub(from) > maxRangeDays*24*time.Hour {
		return nil, 0, 0, ErrInvalidMenuEntry{Reason: "The range must run forwards and cover at most 62 days."}
	}

	// YYYY-MM-DD strings sort like the dates they are, so plain string
	// comparison works on every database.
	all := []*MenuEntry{}
	if err := s.Where("day >= ? AND day <= ?", e.From, e.To).OrderBy("day ASC, position ASC, id ASC").Find(&all); err != nil {
		return nil, 0, 0, err
	}
	// Meal order isn't alphabetical, so it's sorted here. SortStableFunc
	// keeps the day/position order within each meal.
	slices.SortStableFunc(all, func(x, y *MenuEntry) int {
		if x.Day != y.Day {
			return strings.Compare(x.Day, y.Day)
		}
		return slices.Index(meals, x.Meal) - slices.Index(meals, y.Meal)
	})

	total := int64(len(all))
	if limit, start := limitFromPage(page, perPage); limit > 0 {
		all = all[min(start, len(all)):min(start+limit, len(all))]
	}
	return all, len(all), total, nil
}

// Delete removes an entry. The others in its meal keep their positions; only
// their order matters.
func (e *MenuEntry) Delete(s *xorm.Session, _ web.Auth) error {
	_, err := s.ID(e.ID).Delete(&MenuEntry{})
	return err
}
