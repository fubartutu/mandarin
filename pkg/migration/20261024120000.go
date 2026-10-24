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

// Mandarin: the household menu. Frozen copy of
// pkg/household/menu.MenuEntry as it was at this migration.
type menuEntries20261024120000 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	Day         string    `xorm:"varchar(10) not null index"`
	Meal        string    `xorm:"varchar(20) not null"`
	Position    int       `xorm:"int not null default 0"`
	Note        string    `xorm:"varchar(250) not null default ''"`
	RecipeID    int64     `xorm:"bigint not null default 0 index"`
	Servings    float64   `xorm:"double not null default 0"`
	CreatedByID int64     `xorm:"bigint not null default 0"`
	Created     time.Time `xorm:"created not null"`
	Updated     time.Time `xorm:"updated not null"`
}

func (menuEntries20261024120000) TableName() string {
	return "menu_entries"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261024120000",
		Description: "Mandarin: menu entries",
		Migrate: func(tx *xorm.Engine) error {
			// A brand-new table, so plain Sync is safe.
			return tx.Sync(menuEntries20261024120000{}) //nolint:forbidigo // new table only
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(menuEntries20261024120000{})
		},
	})
}
