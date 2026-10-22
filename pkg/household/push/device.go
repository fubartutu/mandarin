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

package push

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// Device is one phone or browser that turned push on, for one user.
type Device struct {
	ID     int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true"`
	UserID int64 `xorm:"bigint not null index" json:"-"`
	// The push service's address for this device, e.g. on web.push.apple.com.
	// It's long, so it's matched by its hash.
	Endpoint     string `xorm:"text not null" json:"-"`
	EndpointHash string `xorm:"varchar(64) not null unique" json:"-"`
	P256dh       string `xorm:"'p256dh' varchar(255) not null" json:"-"`
	Auth         string `xorm:"varchar(255) not null" json:"-"`
	Name         string `xorm:"varchar(100) not null default ''" json:"name" doc:"What the app calls this device, e.g. iPhone · Safari."`

	Created  time.Time  `xorm:"created not null" json:"created" readOnly:"true"`
	LastSent *time.Time `xorm:"datetime null" json:"last_sent" readOnly:"true" doc:"When a push last reached the push service for it; null if never."`
}

// TableName sets the database table name.
func (*Device) TableName() string {
	return "push_devices"
}

func hashEndpoint(endpoint string) string {
	sum := sha256.Sum256([]byte(endpoint))
	return hex.EncodeToString(sum[:])
}

// DeviceInput is what the app sends after the browser subscribed: the
// PushSubscription's endpoint and keys, and a name for the list.
type DeviceInput struct {
	Endpoint string `json:"endpoint" minLength:"1" maxLength:"2000"`
	Keys     struct {
		P256dh string `json:"p256dh" minLength:"1" maxLength:"255"`
		Auth   string `json:"auth" minLength:"1" maxLength:"255"`
	} `json:"keys"`
	Name string `json:"name" maxLength:"100"`
}

// SaveDevice adds the device for the caller, or updates it when the same
// push address is already known (the browser re-subscribed, or another
// person signed in on this device: it then belongs to them).
func SaveDevice(s *xorm.Session, a web.Auth, in *DeviceInput) (*Device, error) {
	if err := requireMember(a); err != nil {
		return nil, err
	}
	u, err := user.GetFromAuth(a)
	if err != nil {
		return nil, err
	}
	if parsed, err := url.Parse(in.Endpoint); err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, ErrInvalidDevice{Reason: "The push address must be https."}
	}
	name := strings.TrimSpace(in.Name)
	if utf8.RuneCountInString(name) > 100 {
		name = string([]rune(name)[:100])
	}

	d := &Device{}
	exists, err := s.Where("endpoint_hash = ?", hashEndpoint(in.Endpoint)).Get(d)
	if err != nil {
		return nil, err
	}
	d.UserID = u.ID
	d.Endpoint = in.Endpoint
	d.EndpointHash = hashEndpoint(in.Endpoint)
	d.P256dh = in.Keys.P256dh
	d.Auth = in.Keys.Auth
	d.Name = name
	if exists {
		_, err = s.ID(d.ID).Cols("user_id", "p256dh", "auth", "name").Update(d)
	} else {
		_, err = s.Insert(d)
	}
	return d, err
}

// ListDevices returns the caller's devices, newest first.
func ListDevices(s *xorm.Session, a web.Auth) ([]*Device, error) {
	if err := requireMember(a); err != nil {
		return nil, err
	}
	devices := []*Device{}
	err := s.Where("user_id = ?", a.GetID()).OrderBy("id DESC").Find(&devices)
	return devices, err
}

// DeleteDevice removes one of the caller's devices.
func DeleteDevice(s *xorm.Session, a web.Auth, id int64) error {
	if err := requireMember(a); err != nil {
		return err
	}
	n, err := s.Where("id = ? AND user_id = ?", id, a.GetID()).Delete(&Device{})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrDeviceDoesNotExist{DeviceID: id}
	}
	return nil
}

// DeleteDeviceByEndpoint removes the caller's device with this push
// address, e.g. when "turn off on this device" is pressed.
func DeleteDeviceByEndpoint(s *xorm.Session, a web.Auth, endpoint string) error {
	if err := requireMember(a); err != nil {
		return err
	}
	_, err := s.Where("endpoint_hash = ? AND user_id = ?", hashEndpoint(endpoint), a.GetID()).Delete(&Device{})
	return err
}
