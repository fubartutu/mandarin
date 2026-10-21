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

// Mandarin: the recipe box. Frozen copy of pkg/household/recipes.Recipe as it
// was at this migration; the JSON columns are plain text here.
type recipes20261021160000 struct {
	ID           int64     `xorm:"bigint autoincr not null unique pk"`
	Title        string    `xorm:"varchar(250) not null"`
	SourceURL    string    `xorm:"'source_url' varchar(2000) not null default ''"`
	Description  string    `xorm:"text null"`
	Servings     float64   `xorm:"double not null default 0"`
	Yields       string    `xorm:"varchar(100) not null default ''"`
	PrepMinutes  int       `xorm:"int not null default 0"`
	CookMinutes  int       `xorm:"int not null default 0"`
	TotalMinutes int       `xorm:"int not null default 0"`
	Ingredients  string    `xorm:"text null"`
	Steps        string    `xorm:"text null"`
	Tags         string    `xorm:"text null"`
	Notes        string    `xorm:"text null"`
	ImageIDs     string    `xorm:"'image_ids' text null"`
	CreatedByID  int64     `xorm:"bigint not null default 0"`
	Created      time.Time `xorm:"created not null"`
	Updated      time.Time `xorm:"updated not null"`
}

func (recipes20261021160000) TableName() string {
	return "recipes"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261021160000",
		Description: "Mandarin: recipes",
		Migrate: func(tx *xorm.Engine) error {
			// A brand-new table, so plain Sync is safe.
			return tx.Sync(recipes20261021160000{}) //nolint:forbidigo // new table only
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(recipes20261021160000{})
		},
	})
}
