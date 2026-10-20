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

	"github.com/google/uuid"
	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

// Mandarin: offline sync. Items get a device-made uid and per-field change
// times; a change log and a conflict log are added. Frozen copies of the
// pkg/household/shopping models as they were at this migration.

// Only the new columns: partialSync keeps the rest of shopping_items as is.
// uid starts without its unique index: existing rows get their uids first.
type shoppingItems20261021120000 struct {
	UID        string `xorm:"'uid' varchar(36) not null default ''"`
	FieldTimes string `xorm:"text null"`
}

func (shoppingItems20261021120000) TableName() string {
	return "shopping_items"
}

type shoppingChanges20261021120000 struct {
	ID       int64     `xorm:"bigint autoincr not null unique pk"`
	Entity   string    `xorm:"varchar(20) not null unique(entity_row)"`
	EntityID int64     `xorm:"bigint not null unique(entity_row)"`
	UID      string    `xorm:"'uid' varchar(36) not null default '' index"`
	Deleted  bool      `xorm:"bool not null default false"`
	Created  time.Time `xorm:"created not null"`
}

func (shoppingChanges20261021120000) TableName() string {
	return "shopping_changes"
}

type shoppingConflicts20261021120000 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	ItemUID     string    `xorm:"varchar(36) not null default '' index"`
	ItemName    string    `xorm:"varchar(250) not null default ''"`
	Field       string    `xorm:"varchar(20) not null"`
	WinnerValue string    `xorm:"text null"`
	WinnerByID  int64     `xorm:"bigint not null default 0"`
	WinnerAt    time.Time `xorm:"datetime not null"`
	LoserValue  string    `xorm:"text null"`
	LoserByID   int64     `xorm:"bigint not null default 0"`
	LoserAt     time.Time `xorm:"datetime not null"`
	Created     time.Time `xorm:"created not null"`
}

func (shoppingConflicts20261021120000) TableName() string {
	return "shopping_conflicts"
}

func addShoppingSync20261021120000(tx *xorm.Engine) error {
	// Brand-new tables, so plain Sync is safe (see the migration skill).
	err := tx.Sync( //nolint:forbidigo // new tables only
		shoppingChanges20261021120000{},
		shoppingConflicts20261021120000{},
	)
	if err != nil {
		return err
	}
	if err := partialSync(tx, shoppingItems20261021120000{}); err != nil {
		return err
	}

	// Give existing items a uid each, then make uids unique. The model's
	// `unique` tag names the index UQE_shopping_items_uid.
	var ids []int64
	if err := tx.Table("shopping_items").Where("uid = ''").Cols("id").Find(&ids); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.Exec("UPDATE shopping_items SET uid = ? WHERE id = ?", uuid.NewString(), id); err != nil {
			return err
		}
	}
	query := "CREATE UNIQUE INDEX IF NOT EXISTS UQE_shopping_items_uid ON shopping_items (uid)"
	if tx.Dialect().URI().DBType == schemas.MYSQL {
		// MySQL has no IF NOT EXISTS for indexes.
		exists, err := tx.Table("information_schema.statistics").
			Where("table_schema = DATABASE() AND table_name = 'shopping_items' AND index_name = 'UQE_shopping_items_uid'").
			Exist()
		if err != nil || exists {
			return err
		}
		query = "CREATE UNIQUE INDEX UQE_shopping_items_uid ON shopping_items (uid)"
	}
	_, err = tx.Exec(query)
	return err
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261021120000",
		Description: "Mandarin: offline sync (item uids and field times, change log, conflict log)",
		Migrate:     addShoppingSync20261021120000,
		Rollback: func(tx *xorm.Engine) error {
			if _, err := tx.Exec("DROP INDEX UQE_shopping_items_uid ON shopping_items"); err != nil {
				return err
			}
			for _, col := range []string{"uid", "field_times"} {
				if err := dropTableColum(tx, "shopping_items", col); err != nil {
					return err
				}
			}
			return tx.DropTables(shoppingConflicts20261021120000{}, shoppingChanges20261021120000{})
		},
	})
}
