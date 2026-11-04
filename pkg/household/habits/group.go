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
	"code.vikunja.io/api/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// HabitGroup is a routine: a named, ordered set of habits, e.g. Morning or
// Kids' bedtime. Like a habit it's personal or shared with the household.
type HabitGroup struct {
	ID       int64  `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this group."`
	Title    string `xorm:"varchar(250) not null" json:"title" minLength:"1" maxLength:"250" doc:"The routine's name, e.g. Morning."`
	Position int    `xorm:"int not null default 0" json:"position" doc:"Order among groups, lowest first."`
	Shared   bool   `xorm:"not null default false" json:"shared" doc:"true when every household member sees the group. Only the owner can make a shared group personal again."`
	OwnerID  int64  `xorm:"bigint not null index" json:"owner_id" readOnly:"true" doc:"The user who made the group."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When the group was made."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When the group was last changed."`

	stored *HabitGroup

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName sets the database table name.
func (*HabitGroup) TableName() string {
	return "habit_groups"
}

func getGroup(s *xorm.Session, id, userID int64) (*HabitGroup, error) {
	g := &HabitGroup{}
	exists, err := s.Where(builder.Eq{"id": id}.And(visibleTo(userID))).Get(g)
	if err != nil {
		return nil, fmt.Errorf("get habit group %d: %w", id, err)
	}
	if !exists {
		return nil, ErrHabitGroupDoesNotExist{GroupID: id}
	}
	return g, nil
}

// CanCreate lets every household member make groups.
func (g *HabitGroup) CanCreate(_ *xorm.Session, a web.Auth) (bool, error) {
	return household.IsMember(a), nil
}

// CanUpdate lets a member change their own groups and shared ones.
func (g *HabitGroup) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	stored, err := canWriteGroup(s, g.ID, a)
	if err != nil {
		return false, err
	}
	g.stored = stored
	return true, nil
}

// CanDelete lets a member delete their own groups and shared ones.
func (g *HabitGroup) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	_, err := canWriteGroup(s, g.ID, a)
	return err == nil, err
}

func canWriteGroup(s *xorm.Session, id int64, a web.Auth) (*HabitGroup, error) {
	if err := household.RequireMember(a); err != nil {
		return nil, err
	}
	return getGroup(s, id, a.GetID())
}

func (g *HabitGroup) normalize() error {
	g.Title = strings.TrimSpace(g.Title)
	if n := utf8.RuneCountInString(g.Title); n < 1 || n > maxTitleLength {
		return ErrInvalidHabit{Reason: "The title must be 1–250 characters."}
	}
	return nil
}

// Create makes a group owned by the caller, after the others.
func (g *HabitGroup) Create(s *xorm.Session, a web.Auth) error {
	if err := g.normalize(); err != nil {
		return err
	}
	last := &HabitGroup{}
	has, err := s.Desc("position").Get(last)
	if err != nil {
		return fmt.Errorf("get last habit group: %w", err)
	}
	g.Position = 0
	if has {
		g.Position = last.Position + 1
	}
	g.ID = 0
	g.OwnerID = a.GetID()
	if _, err := s.Insert(g); err != nil {
		return fmt.Errorf("insert habit group: %w", err)
	}
	return nil
}

// ReadAll returns the caller's groups and the shared ones, by position.
func (g *HabitGroup) ReadAll(s *xorm.Session, a web.Auth, _ string, _ int, _ int) (any, int, int64, error) {
	if err := household.RequireMember(a); err != nil {
		return nil, 0, 0, err
	}
	all := []*HabitGroup{}
	if err := s.Where(visibleTo(a.GetID())).OrderBy("position ASC, id ASC").Find(&all); err != nil {
		return nil, 0, 0, fmt.Errorf("list habit groups: %w", err)
	}
	return all, len(all), int64(len(all)), nil
}

// Update changes the title, position and sharing.
func (g *HabitGroup) Update(s *xorm.Session, a web.Auth) error {
	stored := g.stored
	if stored == nil {
		var err error
		if stored, err = getGroup(s, g.ID, a.GetID()); err != nil {
			return err
		}
	}
	if stored.Shared && !g.Shared && stored.OwnerID != a.GetID() {
		return ErrOnlyOwnerCanUnshare{}
	}
	if err := g.normalize(); err != nil {
		return err
	}
	g.OwnerID = stored.OwnerID
	g.Created = stored.Created
	if g.Title == stored.Title && g.Position == stored.Position && g.Shared == stored.Shared {
		g.Updated = stored.Updated
		return nil
	}
	if _, err := s.ID(g.ID).Cols("title", "position", "shared").Update(g); err != nil {
		return fmt.Errorf("update habit group %d: %w", g.ID, err)
	}
	return nil
}

// Delete removes the group; its habits stay, ungrouped.
func (g *HabitGroup) Delete(s *xorm.Session, _ web.Auth) error {
	if _, err := s.Table("habits").Where("group_id = ?", g.ID).Update(map[string]any{"group_id": 0}); err != nil {
		return fmt.Errorf("ungroup habits of group %d: %w", g.ID, err)
	}
	if _, err := s.ID(g.ID).Delete(&HabitGroup{}); err != nil {
		return fmt.Errorf("delete habit group %d: %w", g.ID, err)
	}
	return nil
}
