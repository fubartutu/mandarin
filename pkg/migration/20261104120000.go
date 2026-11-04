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

package migration

import (
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// Mandarin: habits and routines. Frozen copies of pkg/household/habits'
// tables as they were at this migration (that package can't be imported
// here: it would be an import cycle).

type habits20261104120000 struct {
	ID            int64     `xorm:"bigint autoincr not null unique pk"`
	Title         string    `xorm:"varchar(250) not null"`
	Kind          string    `xorm:"varchar(20) not null"`
	Weekdays      string    `xorm:"varchar(7) not null default '1111111'"`
	Goal          int       `xorm:"int not null default 1"`
	Color         string    `xorm:"varchar(6) not null default ''"`
	GroupID       int64     `xorm:"bigint not null default 0 index"`
	Position      int       `xorm:"int not null default 0"`
	Shared        bool      `xorm:"not null default false"`
	ReminderTimes []string  `xorm:"text null json"`
	Archived      bool      `xorm:"not null default false"`
	OwnerID       int64     `xorm:"bigint not null index"`
	Created       time.Time `xorm:"created not null"`
	Updated       time.Time `xorm:"updated not null"`
}

func (habits20261104120000) TableName() string { return "habits" }

type habitGroups20261104120000 struct {
	ID       int64     `xorm:"bigint autoincr not null unique pk"`
	Title    string    `xorm:"varchar(250) not null"`
	Position int       `xorm:"int not null default 0"`
	Shared   bool      `xorm:"not null default false"`
	OwnerID  int64     `xorm:"bigint not null index"`
	Created  time.Time `xorm:"created not null"`
	Updated  time.Time `xorm:"updated not null"`
}

func (habitGroups20261104120000) TableName() string { return "habit_groups" }

type habitEntries20261104120000 struct {
	ID      int64     `xorm:"bigint autoincr not null unique pk"`
	UID     string    `xorm:"'uid' varchar(36) not null unique"`
	HabitID int64     `xorm:"bigint not null index"`
	Day     string    `xorm:"varchar(10) not null index"`
	UserID  int64     `xorm:"bigint not null"`
	Created time.Time `xorm:"created not null"`
}

func (habitEntries20261104120000) TableName() string { return "habit_entries" }

type habitSkips20261104120000 struct {
	ID      int64     `xorm:"bigint autoincr not null unique pk"`
	HabitID int64     `xorm:"bigint not null unique(habit_day)"`
	Day     string    `xorm:"varchar(10) not null unique(habit_day)"`
	Reason  string    `xorm:"varchar(250) not null default ''"`
	UserID  int64     `xorm:"bigint not null"`
	Created time.Time `xorm:"created not null"`
}

func (habitSkips20261104120000) TableName() string { return "habit_skips" }

type habitPauses20261104120000 struct {
	ID      int64     `xorm:"bigint autoincr not null unique pk"`
	HabitID int64     `xorm:"bigint not null index"`
	From    string    `xorm:"'from_day' varchar(10) not null"`
	Until   string    `xorm:"'until_day' varchar(10) not null default ''"`
	Created time.Time `xorm:"created not null"`
}

func (habitPauses20261104120000) TableName() string { return "habit_pauses" }

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261104120000",
		Description: "Mandarin: habits and routines",
		Migrate: func(tx *xorm.Engine) error {
			// Brand-new tables, so plain Sync is safe.
			return tx.Sync( //nolint:forbidigo // new tables only
				habits20261104120000{},
				habitGroups20261104120000{},
				habitEntries20261104120000{},
				habitSkips20261104120000{},
				habitPauses20261104120000{},
			)
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(
				habits20261104120000{},
				habitGroups20261104120000{},
				habitEntries20261104120000{},
				habitSkips20261104120000{},
				habitPauses20261104120000{},
			)
		},
	})
}
