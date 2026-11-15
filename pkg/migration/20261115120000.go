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

// Mandarin: the "usual shop" mark on shopping catalogue entries.
// Only the new column; partialSync keeps the rest of shopping_catalog_items.
type shoppingCatalogItems20261115120000 struct {
	Usual bool `xorm:"bool not null default false"`
}

func (shoppingCatalogItems20261115120000) TableName() string {
	return "shopping_catalog_items"
}

func addCatalogUsual20261115120000(tx *xorm.Engine) error {
	return partialSync(tx, shoppingCatalogItems20261115120000{})
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261115120000",
		Description: "Mandarin: usual shop mark on shopping catalogue items",
		Migrate:     addCatalogUsual20261115120000,
		Rollback: func(_ *xorm.Engine) error {
			return nil
		},
	})
}
