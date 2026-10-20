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

package shopping

import (
	"strconv"
	"time"

	"code.vikunja.io/api/pkg/user"

	"xorm.io/xorm"
)

// fieldDeleted is the conflict "field" for an edit that arrived after the
// item had been removed.
const fieldDeleted = "deleted"

// ShoppingConflict records two edits to the same item field that crossed:
// the later one was kept, the other lost. Both phones see the log.
type ShoppingConflict struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	ID       int64  `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this conflict."`
	ItemUID  string `xorm:"varchar(36) not null default '' index" json:"item_uid" doc:"The item both edits were to."`
	ItemName string `xorm:"varchar(250) not null default ''" json:"item_name" doc:"The item's name at the time."`
	Field    string `xorm:"varchar(20) not null" json:"field" doc:"What both changed: name, amount, note, category_id or done; deleted when an edit arrived after the item was removed."`

	WinnerValue string     `xorm:"text null" json:"winner_value" doc:"The kept value: a name, an amount such as \"2 kg\", a note, a category id, or true/false for done."`
	WinnerByID  int64      `xorm:"bigint not null default 0" json:"-"`
	WinnerBy    *user.User `xorm:"-" json:"winner_by" doc:"Who made the kept change; null if unknown."`
	WinnerAt    time.Time  `xorm:"datetime not null" json:"winner_at" doc:"When the kept change was made."`

	LoserValue string     `xorm:"text null" json:"loser_value" doc:"The value that was overwritten or dropped."`
	LoserByID  int64      `xorm:"bigint not null default 0" json:"-"`
	LoserBy    *user.User `xorm:"-" json:"loser_by" doc:"Who made the lost change."`
	LoserAt    time.Time  `xorm:"datetime not null" json:"loser_at" doc:"When the lost change was made."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When the server noticed the conflict."`
}

// TableName sets the database table name.
func (*ShoppingConflict) TableName() string {
	return "shopping_conflicts"
}

// side is one of two crossing edits.
type side struct {
	value any
	stamp fieldStamp
}

// addConflict logs a conflict on item's field and tells phones about it.
func addConflict(s *xorm.Session, item *ShoppingItem, field string, winner, loser side) error {
	c := &ShoppingConflict{
		ItemUID:     item.UID,
		ItemName:    item.Name,
		Field:       field,
		WinnerValue: conflictValue(winner.value),
		WinnerByID:  winner.stamp.By,
		WinnerAt:    time.UnixMilli(winner.stamp.At),
		LoserValue:  conflictValue(loser.value),
		LoserByID:   loser.stamp.By,
		LoserAt:     time.UnixMilli(loser.stamp.At),
	}
	if _, err := s.Insert(c); err != nil {
		return err
	}
	return recordChange(s, entityConflict, c.ID, "", false)
}

// conflictValue turns a fieldValue into the text the log shows.
func conflictValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case amount:
		if v.Quantity == 0 {
			return v.Unit
		}
		q := strconv.FormatFloat(v.Quantity, 'f', -1, 64)
		if v.Unit == "" {
			return q
		}
		return q + " " + v.Unit
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}

// fillConflictUsers loads the members named in conflicts.
func fillConflictUsers(s *xorm.Session, conflicts []*ShoppingConflict) error {
	ids := make([]int64, 0, len(conflicts)*2)
	for _, c := range conflicts {
		ids = append(ids, c.WinnerByID, c.LoserByID)
	}
	users, err := loadCreators(s, ids)
	if err != nil {
		return err
	}
	for _, c := range conflicts {
		c.WinnerBy = users[c.WinnerByID]
		c.LoserBy = users[c.LoserByID]
	}
	return nil
}
