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

// Mandarin: when a repeating task stops repeating. Frozen copy of
// models.TaskRepeat as it was at this migration.
type taskRepeats20261026120000 struct {
	TaskID      int64      `xorm:"bigint not null unique pk"`
	Until       *time.Time `xorm:"datetime null 'repeat_until'"`
	RepeatsLeft int        `xorm:"int not null default 0"`
}

func (taskRepeats20261026120000) TableName() string {
	return "task_repeats"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261026120000",
		Description: "Mandarin: repeat ends for repeating tasks",
		Migrate: func(tx *xorm.Engine) error {
			// A brand-new table, so plain Sync is safe.
			return tx.Sync(taskRepeats20261026120000{}) //nolint:forbidigo // new table only
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(taskRepeats20261026120000{})
		},
	})
}
