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
	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// Mandarin: an optional shop per shopping item, and an aisle order per shop.
// Only the new columns: partialSync keeps the rest of each table as is.

type shoppingItems20261022150000 struct {
	ShopID int64 `xorm:"bigint not null default 0 index"`
}

func (shoppingItems20261022150000) TableName() string {
	return "shopping_items"
}

type shoppingShops20261022150000 struct {
	CategoryOrder string `xorm:"text null"`
}

func (shoppingShops20261022150000) TableName() string {
	return "shopping_shops"
}

func addShoppingItemShop20261022150000(tx *xorm.Engine) error {
	if err := partialSync(tx, shoppingItems20261022150000{}); err != nil {
		return err
	}
	return partialSync(tx, shoppingShops20261022150000{})
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261022150000",
		Description: "Mandarin: a shop per shopping item, and an aisle order per shop",
		Migrate:     addShoppingItemShop20261022150000,
		Rollback: func(tx *xorm.Engine) error {
			if err := dropTableColum(tx, "shopping_items", "shop_id"); err != nil {
				return err
			}
			return dropTableColum(tx, "shopping_shops", "category_order")
		},
	})
}
