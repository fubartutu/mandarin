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

// MenuEntry is one thing planned for one meal on one day: either a recipe
// with its own servings (RecipeID set, Note empty) or a free-text note
// ("leftovers", "takeaway"; RecipeID 0). A meal slot (day + meal) can hold
// several entries, kept in Position order.
type MenuEntry struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	ID       int64   `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this entry."`
	Day      string  `xorm:"varchar(10) not null index" json:"day" format:"date" doc:"The calendar day it's planned for, as YYYY-MM-DD (no time or timezone)."`
	Meal     string  `xorm:"varchar(20) not null" json:"meal" enum:"breakfast,lunch,dinner" doc:"Which meal of the day."`
	Position int     `xorm:"int not null default 0" json:"position" readOnly:"true" doc:"Order within the day's meal, lowest first. New entries go last."`
	Note     string  `xorm:"varchar(250) not null default ''" json:"note" maxLength:"250" doc:"A note entry's text, e.g. leftovers or takeaway. Empty on a recipe entry."`
	RecipeID int64   `xorm:"bigint not null default 0 index" json:"recipe_id" minimum:"0" doc:"The planned recipe, or 0 for a note entry. Can't be changed after adding."`
	Servings float64 `xorm:"double not null default 0" json:"servings" minimum:"0" doc:"A recipe entry's servings, e.g. 6 when cooking for guests. Left out or 0 on create: the recipe's own servings. Always 0 on a note entry."`

	// RecipeTitle is filled in on every response; it isn't stored.
	RecipeTitle string `xorm:"-" json:"recipe_title" readOnly:"true" doc:"A recipe entry's recipe title; empty on a note entry."`

	CreatedByID int64     `xorm:"bigint not null default 0" json:"-"`
	Created     time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When this entry was added."`
	Updated     time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this entry was last changed."`

	// The range a listing covers, from the query string.
	From string `xorm:"-" json:"-"`
	To   string `xorm:"-" json:"-"`
	// What an update changes, from the PUT body.
	Changes EntryChanges `xorm:"-" json:"-"`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// EntryChanges is the body of PUT /menu/{entry}. A field left out (nil)
// stays as it is.
type EntryChanges struct {
	Servings *float64 `json:"servings,omitempty" minimum:"0" doc:"New servings. Recipe entries only."`
	Note     *string  `json:"note,omitempty" maxLength:"250" doc:"New text, 1–250 characters. Note entries only."`
	Day      *string  `json:"day,omitempty" format:"date" doc:"Move the entry to this day (YYYY-MM-DD), last in the same meal there. The same day changes nothing."`
}

// recipeRef is the part of a recipe the menu needs. It reads the recipes
// table directly instead of importing pkg/household/recipes, so recipes can
// later call into the menu (a deleted recipe turns its entries into notes).
type recipeRef struct {
	ID       int64
	Title    string
	Servings float64
}

func getRecipeRef(s *xorm.Session, id int64) (*recipeRef, error) {
	r := &recipeRef{}
	exists, err := s.Table("recipes").Cols("id", "title", "servings").Where("id = ?", id).Get(r)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrInvalidMenuEntry{Reason: "This recipe does not exist."}
	}
	return r, nil
}

// addRecipeTitles fills in RecipeTitle with one query for all the entries.
func addRecipeTitles(s *xorm.Session, entries ...*MenuEntry) error {
	ids := []int64{}
	for _, e := range entries {
		if e.RecipeID != 0 && !slices.Contains(ids, e.RecipeID) {
			ids = append(ids, e.RecipeID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	refs := []*recipeRef{}
	if err := s.Table("recipes").Cols("id", "title").In("id", ids).Find(&refs); err != nil {
		return err
	}
	titles := make(map[int64]string, len(refs))
	for _, r := range refs {
		titles[r.ID] = r.Title
	}
	for _, e := range entries {
		e.RecipeTitle = titles[e.RecipeID]
	}
	return nil
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

// CanUpdate lets every household member change any entry. It loads the
// stored entry into e, keeping the requested Changes for Update.
func (e *MenuEntry) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	if err := household.RequireMember(a); err != nil {
		return false, err
	}
	stored, err := getEntry(s, e.ID)
	if err != nil {
		return false, err
	}
	changes := e.Changes
	*e = *stored
	e.Changes = changes
	return true, nil
}

func canWrite(s *xorm.Session, id int64, a web.Auth) (bool, error) {
	if err := household.RequireMember(a); err != nil {
		return false, err
	}
	if _, err := getEntry(s, id); err != nil {
		return false, err
	}
	return true, nil
}

// checkNote trims a note entry's text and checks its length.
func checkNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if n := utf8.RuneCountInString(note); n < 1 || n > maxNoteLength {
		return "", ErrInvalidMenuEntry{Reason: "The note must be 1–250 characters."}
	}
	return note, nil
}

func checkServings(servings float64) error {
	if servings < 0 {
		return ErrInvalidMenuEntry{Reason: "The servings can't be negative."}
	}
	return nil
}

// parseDay checks a YYYY-MM-DD day. time.Parse also rejects 2026-02-30.
func parseDay(day string) (time.Time, error) {
	t, err := time.Parse(dayFormat, day)
	if err != nil {
		return time.Time{}, ErrInvalidMenuEntry{Reason: "The day must be a date written YYYY-MM-DD."}
	}
	return t, nil
}

// normalize checks a new entry. A recipe entry without servings gets the
// recipe's.
func (e *MenuEntry) normalize(s *xorm.Session) error {
	if _, err := parseDay(e.Day); err != nil {
		return err
	}
	if !slices.Contains(meals, e.Meal) {
		return ErrInvalidMenuEntry{Reason: "The meal must be breakfast, lunch or dinner."}
	}
	if err := checkServings(e.Servings); err != nil {
		return err
	}
	if e.RecipeID == 0 {
		if e.Servings != 0 {
			return ErrInvalidMenuEntry{Reason: "Only a recipe entry has servings."}
		}
		note, err := checkNote(e.Note)
		e.Note = note
		return err
	}
	if strings.TrimSpace(e.Note) != "" {
		return ErrInvalidMenuEntry{Reason: "An entry is either a recipe or a note, not both."}
	}
	e.Note = ""
	recipe, err := getRecipeRef(s, e.RecipeID)
	if err != nil {
		return err
	}
	if e.Servings == 0 {
		e.Servings = recipe.Servings
	}
	e.RecipeTitle = recipe.Title
	return nil
}

// nextPosition is the position after the last entry in a day's meal, or 0
// when the meal is empty.
func nextPosition(s *xorm.Session, day, meal string) (int, error) {
	last := &MenuEntry{}
	has, err := s.Where("day = ? AND meal = ?", day, meal).Desc("position").Get(last)
	if err != nil || !has {
		return 0, err
	}
	return last.Position + 1, nil
}

// Create adds an entry at the end of its day's meal.
func (e *MenuEntry) Create(s *xorm.Session, a web.Auth) error {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	if err := e.normalize(s); err != nil {
		return err
	}
	position, err := nextPosition(s, e.Day, e.Meal)
	if err != nil {
		return err
	}
	e.Position = position
	e.ID = 0
	e.CreatedByID = u.ID
	_, err = s.Insert(e)
	return err
}

// ReadAll returns the entries from From to To (both included), by day, then
// meal (breakfast first), then position. Paging is allowed but a week fits
// on one page.
func (e *MenuEntry) ReadAll(s *xorm.Session, a web.Auth, _ string, page int, perPage int) (any, int, int64, error) {
	if err := household.RequireMember(a); err != nil {
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
	if limit, start := household.LimitFromPage(page, perPage); limit > 0 {
		all = all[min(start, len(all)):min(start+limit, len(all))]
	}
	if err := addRecipeTitles(s, all...); err != nil {
		return nil, 0, 0, err
	}
	return all, len(all), total, nil
}

// Update applies e.Changes to the entry CanUpdate loaded: servings on a
// recipe entry, the text of a note entry, or another day (where it goes last
// in its meal). Nothing is written when nothing changes (SQLite allows one
// writer at a time).
func (e *MenuEntry) Update(s *xorm.Session, _ web.Auth) error {
	cols := []string{}
	if c := e.Changes.Servings; c != nil {
		if e.RecipeID == 0 {
			return ErrInvalidMenuEntry{Reason: "Only a recipe entry has servings."}
		}
		if err := checkServings(*c); err != nil {
			return err
		}
		if *c != e.Servings {
			e.Servings = *c
			cols = append(cols, "servings")
		}
	}
	if c := e.Changes.Note; c != nil {
		if e.RecipeID != 0 {
			return ErrInvalidMenuEntry{Reason: "A recipe entry has no note."}
		}
		note, err := checkNote(*c)
		if err != nil {
			return err
		}
		if note != e.Note {
			e.Note = note
			cols = append(cols, "note")
		}
	}
	if c := e.Changes.Day; c != nil && *c != e.Day {
		if _, err := parseDay(*c); err != nil {
			return err
		}
		position, err := nextPosition(s, *c, e.Meal)
		if err != nil {
			return err
		}
		e.Day = *c
		e.Position = position
		cols = append(cols, "day", "position")
	}
	if len(cols) > 0 {
		if _, err := s.ID(e.ID).Cols(cols...).Update(e); err != nil {
			return err
		}
	}
	return addRecipeTitles(s, e)
}

// Delete removes an entry. The others in its meal keep their positions; only
// their order matters.
func (e *MenuEntry) Delete(s *xorm.Session, _ web.Auth) error {
	_, err := s.ID(e.ID).Delete(&MenuEntry{})
	return err
}
