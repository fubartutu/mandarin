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

// Mandarin: snoozed push reminders (the Snooze button on task notifications).
// A frozen copy of pkg/household/push's pushSnooze as it was at this migration.

type pushSnoozes20261103120000 struct {
	ID     int64     `xorm:"bigint autoincr not null unique pk"`
	UserID int64     `xorm:"bigint not null unique(user_task)"`
	TaskID int64     `xorm:"bigint not null unique(user_task)"`
	At     time.Time `xorm:"datetime not null index"`
}

func (pushSnoozes20261103120000) TableName() string { return "push_snoozes" }

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261103120000",
		Description: "Mandarin: snoozed push reminders",
		Migrate: func(tx *xorm.Engine) error {
			// A brand-new table, so plain Sync is safe.
			return tx.Sync(pushSnoozes20261103120000{}) //nolint:forbidigo // new table only
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(pushSnoozes20261103120000{})
		},
	})
}
