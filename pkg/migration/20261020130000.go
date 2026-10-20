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

// Mandarin: shops, the item catalogue and "not stocked" marks. Frozen copies
// of the pkg/household/shopping models as they were at this migration.

type shoppingShops20261020130000 struct {
	ID       int64     `xorm:"bigint autoincr not null unique pk"`
	ListID   int64     `xorm:"bigint not null index"`
	Title    string    `xorm:"varchar(250) not null"`
	Position float64   `xorm:"double not null default 0"`
	Created  time.Time `xorm:"created not null"`
	Updated  time.Time `xorm:"updated not null"`
}

func (shoppingShops20261020130000) TableName() string {
	return "shopping_shops"
}

type shoppingCatalogItems20261020130000 struct {
	ID          int64      `xorm:"bigint autoincr not null unique pk"`
	Name        string     `xorm:"varchar(250) not null"`
	NameKey     string     `xorm:"varchar(250) not null unique"`
	CategoryID  int64      `xorm:"bigint not null default 0 index"`
	Quantity    float64    `xorm:"double not null default 0"`
	Unit        string     `xorm:"varchar(50) not null default ''"`
	TimesAdded  int64      `xorm:"bigint not null default 0"`
	LastAdded   *time.Time `xorm:"datetime null"`
	PhotoFileID int64      `xorm:"bigint not null default 0"`
	Created     time.Time  `xorm:"created not null"`
	Updated     time.Time  `xorm:"updated not null"`
}

func (shoppingCatalogItems20261020130000) TableName() string {
	return "shopping_catalog_items"
}

type shoppingNotStocked20261020130000 struct {
	ID            int64 `xorm:"bigint autoincr not null unique pk"`
	CatalogItemID int64 `xorm:"bigint not null unique(item_shop)"`
	ShopID        int64 `xorm:"bigint not null unique(item_shop) index"`
}

func (shoppingNotStocked20261020130000) TableName() string {
	return "shopping_not_stocked"
}

// Only the new column: partialSync keeps the rest of shopping_items as is.
type shoppingItems20261020130000 struct {
	CatalogItemID int64 `xorm:"bigint not null default 0 index"`
}

func (shoppingItems20261020130000) TableName() string {
	return "shopping_items"
}

func addShoppingCatalog20261020130000(tx *xorm.Engine) error {
	// Brand-new tables, so plain Sync is safe (see the migration skill).
	err := tx.Sync( //nolint:forbidigo // new tables only
		shoppingShops20261020130000{},
		shoppingCatalogItems20261020130000{},
		shoppingNotStocked20261020130000{},
	)
	if err != nil {
		return err
	}
	// Existing table: partialSync, never plain Sync (it would drop indexes).
	return partialSync(tx, shoppingItems20261020130000{})
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261020130000",
		Description: "Mandarin: shops, item catalogue, not-stocked marks, and items' catalogue link",
		Migrate:     addShoppingCatalog20261020130000,
		Rollback: func(tx *xorm.Engine) error {
			if err := dropTableColum(tx, "shopping_items", "catalog_item_id"); err != nil {
				return err
			}
			return tx.DropTables(
				shoppingNotStocked20261020130000{},
				shoppingCatalogItems20261020130000{},
				shoppingShops20261020130000{},
			)
		},
	})
}
