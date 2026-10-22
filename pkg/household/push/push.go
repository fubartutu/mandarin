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

// Package push sends Web Push notifications to the household's phones and
// browsers: task reminders and the daily overdue summary (both raised by
// Vikunja's notification system, see notify.go), and batched summaries of
// shopping list changes (shopping.go). See SPEC.md, "Push notifications".
//
// A device turns push on in the app; its push address is stored per user
// (Device). The server signs every push with its own key pair (VAPID), made
// on first start and kept in the database, so backups include it: losing it
// would silently break every device's subscription.
package push

import (
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web"

	"github.com/SherClockHolmes/webpush-go"
	"xorm.io/xorm"
)

func init() {
	db.RegisterTables(GetTables())
}

// GetTables returns all structs which are also a table.
func GetTables() []any {
	return []any{
		&Device{},
		&Preferences{},
		&vapidKeys{},
		&pendingShoppingAdd{},
	}
}

func requireMember(a web.Auth) error {
	if !household.IsMember(a) {
		return models.ErrGenericForbidden{}
	}
	return nil
}

// vapidKeys is the server's key pair for signing pushes; one row.
type vapidKeys struct {
	ID         int64  `xorm:"bigint autoincr not null unique pk"`
	PublicKey  string `xorm:"varchar(255) not null"`
	PrivateKey string `xorm:"varchar(255) not null"`
}

func (*vapidKeys) TableName() string {
	return "push_vapid_keys"
}

// keys loads the key pair, making it the first time.
func keys(s *xorm.Session) (*vapidKeys, error) {
	k := &vapidKeys{}
	exists, err := s.OrderBy("id").Get(k)
	if err != nil || exists {
		return k, err
	}
	k.PrivateKey, k.PublicKey, err = webpush.GenerateVAPIDKeys()
	if err != nil {
		return nil, err
	}
	_, err = s.Insert(k)
	return k, err
}

// PublicKey is what a device needs to subscribe (applicationServerKey).
func PublicKey(s *xorm.Session, a web.Auth) (string, error) {
	if err := requireMember(a); err != nil {
		return "", err
	}
	k, err := keys(s)
	if err != nil {
		return "", err
	}
	return k.PublicKey, nil
}

// Init makes the key pair if needed, hooks into Vikunja's notifications and
// the shopping list, and schedules the shopping summaries. Call after
// cron.Init and before Vikunja registers its reminder crons.
func Init() {
	s := db.NewSession()
	defer s.Close()
	if _, err := keys(s); err != nil {
		_ = s.Rollback()
		log.Errorf("Push notifications are off: could not make the server's push keys: %s", err)
		return
	}
	if err := s.Commit(); err != nil {
		log.Errorf("Push notifications are off: %s", err)
		return
	}
	hookNotifications()
	hookShopping()
	log.Info("Push notifications are on.")
}
