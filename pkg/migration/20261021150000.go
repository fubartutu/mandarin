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

// Mandarin: duplicate adds from two offline phones are merged; the merged
// add's uid is kept as an alias of the item it was merged into. Frozen copy
// of pkg/household/shopping.ShoppingItemAlias.
type shoppingItemAliases20261021150000 struct {
	UID     string    `xorm:"'uid' varchar(36) not null pk"`
	ItemUID string    `xorm:"varchar(36) not null index"`
	Created time.Time `xorm:"created not null"`
}

func (shoppingItemAliases20261021150000) TableName() string {
	return "shopping_item_aliases"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261021150000",
		Description: "Mandarin: aliases for merged duplicate shopping items",
		Migrate: func(tx *xorm.Engine) error {
			// A brand-new table, so plain Sync is safe.
			return tx.Sync(shoppingItemAliases20261021150000{}) //nolint:forbidigo // new table only
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(shoppingItemAliases20261021150000{})
		},
	})
}
