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
	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// Preferences are one person's choices about what pushes them.
type Preferences struct {
	UserID int64 `xorm:"bigint not null unique pk" json:"-"`

	Reminders bool `xorm:"bool not null default true" json:"reminders" doc:"A push at each task reminder."`
	Overdue   bool `xorm:"bool not null default true" json:"overdue" doc:"The daily overdue summary (at the time set for overdue reminders)."`
	// Quiet hours hold shopping summaries from 22:00 to 07:00 in the
	// person's time zone. Reminders and the overdue summary come at the
	// times the person set, so they aren't held.
	QuietHours bool `xorm:"bool not null default true" json:"quiet_hours" doc:"Hold shopping summaries from 10pm to 7am."`
	// Shopping lists whose changes by others are summarised; none by default.
	ShoppingListIDs []int64 `xorm:"'shopping_list_ids' text null json" json:"shopping_list_ids" doc:"Shopping lists to send summaries of others' changes for."`
}

// TableName sets the database table name.
func (*Preferences) TableName() string {
	return "push_preferences"
}

func defaultPreferences(userID int64) *Preferences {
	return &Preferences{UserID: userID, Reminders: true, Overdue: true, QuietHours: true, ShoppingListIDs: []int64{}}
}

func preferencesFor(s *xorm.Session, userID int64) (*Preferences, error) {
	p := &Preferences{}
	exists, err := s.Where("user_id = ?", userID).Get(p)
	if err != nil {
		return nil, err
	}
	if !exists {
		return defaultPreferences(userID), nil
	}
	if p.ShoppingListIDs == nil {
		p.ShoppingListIDs = []int64{}
	}
	return p, nil
}

// GetPreferences returns the caller's preferences (defaults until saved).
func GetPreferences(s *xorm.Session, a web.Auth) (*Preferences, error) {
	if err := household.RequireMember(a); err != nil {
		return nil, err
	}
	return preferencesFor(s, a.GetID())
}

// SavePreferences replaces the caller's preferences.
func SavePreferences(s *xorm.Session, a web.Auth, p *Preferences) (*Preferences, error) {
	if err := household.RequireMember(a); err != nil {
		return nil, err
	}
	p.UserID = a.GetID()
	if p.ShoppingListIDs == nil {
		p.ShoppingListIDs = []int64{}
	}
	exists, err := s.Where("user_id = ?", p.UserID).Exist(&Preferences{})
	if err != nil {
		return nil, err
	}
	if exists {
		_, err = s.Where("user_id = ?", p.UserID).Cols("reminders", "overdue", "quiet_hours", "shopping_list_ids").Update(p)
	} else {
		_, err = s.Insert(p)
	}
	return p, err
}
