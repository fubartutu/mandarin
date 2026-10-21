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
	"strings"
	"time"

	"xorm.io/xorm"
)

// Duplicate merging. Two phones offline can both add "milk". When the second
// add arrives and an identical item is already waiting on the list, the add
// is folded into it instead of making a second row. The phone that sent it
// still knows the item by its own uid, so that uid becomes an alias: later
// edits, ticks and deletes sent with it reach the merged item.

// ShoppingItemAlias maps the uid of an add that was merged to the item it
// was merged into.
type ShoppingItemAlias struct { //nolint:revive // stutters, but matches the other table names.
	UID     string    `xorm:"'uid' varchar(36) not null pk"`
	ItemUID string    `xorm:"varchar(36) not null index"`
	Created time.Time `xorm:"created not null"`
}

// TableName sets the database table name.
func (*ShoppingItemAlias) TableName() string {
	return "shopping_item_aliases"
}

// resolveUID returns the uid an op should act on: the merged item's uid when
// uid was merged away, otherwise uid itself.
func resolveUID(s *xorm.Session, uid string) (string, error) {
	alias := &ShoppingItemAlias{}
	exists, err := s.Where("uid = ?", uid).Get(alias)
	if err != nil || !exists {
		return uid, err
	}
	return alias.ItemUID, nil
}

// sameItem says whether two items are the same thing to buy: same name
// (ignoring case), same amount and same note. Category and position don't
// count; whoever was first keeps theirs.
func sameItem(a, b *ShoppingItem) bool {
	return strings.EqualFold(strings.TrimSpace(a.Name), strings.TrimSpace(b.Name)) &&
		a.Quantity == b.Quantity &&
		strings.EqualFold(strings.TrimSpace(a.Unit), strings.TrimSpace(b.Unit)) &&
		strings.TrimSpace(a.Note) == strings.TrimSpace(b.Note)
}

// findDuplicate returns an unticked item on the same list identical to item.
func findDuplicate(s *xorm.Session, item *ShoppingItem) (*ShoppingItem, error) {
	if item.Done {
		return nil, nil
	}
	open := []*ShoppingItem{}
	if err := s.Where("list_id = ? AND done = ?", item.ListID, false).OrderBy("id").Find(&open); err != nil {
		return nil, err
	}
	for _, other := range open {
		if sameItem(item, other) {
			return other, nil
		}
	}
	return nil, nil
}

// mergeInto records that the add with uid became existing.
func mergeInto(s *xorm.Session, uid string, existing *ShoppingItem) error {
	_, err := s.Insert(&ShoppingItemAlias{UID: uid, ItemUID: existing.UID})
	return err
}
