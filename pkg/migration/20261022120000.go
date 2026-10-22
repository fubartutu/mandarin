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

// Mandarin: push notifications. Frozen copies of pkg/household/push's tables
// as they were at this migration; JSON columns are plain text here.

type pushDevices20261022120000 struct {
	ID           int64      `xorm:"bigint autoincr not null unique pk"`
	UserID       int64      `xorm:"bigint not null index"`
	Endpoint     string     `xorm:"text not null"`
	EndpointHash string     `xorm:"varchar(64) not null unique"`
	P256dh       string     `xorm:"'p256dh' varchar(255) not null"`
	Auth         string     `xorm:"varchar(255) not null"`
	Name         string     `xorm:"varchar(100) not null default ''"`
	Created      time.Time  `xorm:"created not null"`
	LastSent     *time.Time `xorm:"datetime null"`
}

func (pushDevices20261022120000) TableName() string { return "push_devices" }

type pushPreferences20261022120000 struct {
	UserID          int64  `xorm:"bigint not null unique pk"`
	Reminders       bool   `xorm:"bool not null default true"`
	Overdue         bool   `xorm:"bool not null default true"`
	QuietHours      bool   `xorm:"bool not null default true"`
	ShoppingListIDs string `xorm:"'shopping_list_ids' text null"`
}

func (pushPreferences20261022120000) TableName() string { return "push_preferences" }

type pushVapidKeys20261022120000 struct {
	ID         int64  `xorm:"bigint autoincr not null unique pk"`
	PublicKey  string `xorm:"varchar(255) not null"`
	PrivateKey string `xorm:"varchar(255) not null"`
}

func (pushVapidKeys20261022120000) TableName() string { return "push_vapid_keys" }

type pushShoppingPending20261022120000 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	RecipientID int64     `xorm:"bigint not null index"`
	ListID      int64     `xorm:"bigint not null"`
	ActorID     int64     `xorm:"bigint not null"`
	ItemName    string    `xorm:"varchar(250) not null"`
	Created     time.Time `xorm:"created not null"`
}

func (pushShoppingPending20261022120000) TableName() string { return "push_shopping_pending" }

func init() {
	tables := []any{pushDevices20261022120000{}, pushPreferences20261022120000{}, pushVapidKeys20261022120000{}, pushShoppingPending20261022120000{}}
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261022120000",
		Description: "Mandarin: push notifications (devices, preferences, server keys, shopping summary queue)",
		Migrate: func(tx *xorm.Engine) error {
			// Brand-new tables, so plain Sync is safe.
			return tx.Sync(tables...) //nolint:forbidigo // new tables only
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(tables...)
		},
	})
}
