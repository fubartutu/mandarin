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

// Mandarin: a switch for habit reminder pushes in each person's push settings.
// Only the new column; partialSync keeps the rest of push_preferences.
type pushPreferences20261105120000 struct {
	HabitReminders bool `xorm:"bool not null default true"`
}

func (pushPreferences20261105120000) TableName() string {
	return "push_preferences"
}

func addHabitReminders20261105120000(tx *xorm.Engine) error {
	return partialSync(tx, pushPreferences20261105120000{})
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261105120000",
		Description: "Mandarin: habit reminder pushes setting",
		Migrate:     addHabitReminders20261105120000,
		Rollback: func(_ *xorm.Engine) error {
			return nil
		},
	})
}
