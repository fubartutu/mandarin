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

// Mandarin: household shopping lists. Frozen copies of the
// pkg/household/shopping models as they were when the tables were created.

type shoppingLists20261020120000 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	Title       string    `xorm:"varchar(250) not null"`
	Position    float64   `xorm:"double not null default 0"`
	CreatedByID int64     `xorm:"bigint not null default 0"`
	Created     time.Time `xorm:"created not null"`
	Updated     time.Time `xorm:"updated not null"`
}

func (shoppingLists20261020120000) TableName() string {
	return "shopping_lists"
}

type shoppingCategories20261020120000 struct {
	ID       int64     `xorm:"bigint autoincr not null unique pk"`
	Title    string    `xorm:"varchar(250) not null"`
	Position float64   `xorm:"double not null default 0"`
	Created  time.Time `xorm:"created not null"`
	Updated  time.Time `xorm:"updated not null"`
}

func (shoppingCategories20261020120000) TableName() string {
	return "shopping_categories"
}

type shoppingItems20261020120000 struct {
	ID          int64      `xorm:"bigint autoincr not null unique pk"`
	ListID      int64      `xorm:"bigint not null index"`
	Name        string     `xorm:"varchar(250) not null"`
	Quantity    float64    `xorm:"double not null default 0"`
	Unit        string     `xorm:"varchar(50) not null default ''"`
	Note        string     `xorm:"text null"`
	CategoryID  int64      `xorm:"bigint not null default 0 index"`
	Done        bool       `xorm:"bool not null default false"`
	DoneAt      *time.Time `xorm:"datetime null"`
	Position    float64    `xorm:"double not null default 0"`
	CreatedByID int64      `xorm:"bigint not null default 0"`
	Created     time.Time  `xorm:"created not null"`
	Updated     time.Time  `xorm:"updated not null"`
}

func (shoppingItems20261020120000) TableName() string {
	return "shopping_items"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261020120000",
		Description: "Mandarin: create shopping list, item and category tables",
		Migrate: func(tx *xorm.Engine) error {
			// Brand-new tables, so plain Sync is safe here (see the migration skill).
			return tx.Sync( //nolint:forbidigo // new tables only
				shoppingLists20261020120000{},
				shoppingCategories20261020120000{},
				shoppingItems20261020120000{},
			)
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(
				shoppingItems20261020120000{},
				shoppingCategories20261020120000{},
				shoppingLists20261020120000{},
			)
		},
	})
}
