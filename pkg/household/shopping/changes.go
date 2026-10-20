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
	"time"

	"code.vikunja.io/api/pkg/events"
	ws "code.vikunja.io/api/pkg/websocket"

	"github.com/ThreeDotsLabs/watermill/message"
	"xorm.io/xorm"
)

// Entities that phones keep a copy of, as named in the change log.
const (
	entityList     = "list" // includes its shops
	entityCategory = "category"
	entityCatalog  = "catalog"
	entityItem     = "item"
	entityConflict = "conflict"
)

// ShoppingChange is one row of the change log that phones sync from. Each
// entity has at most one row: changing it again deletes the old row and adds a
// new one, so the newest id is the entity's latest change. A phone remembers
// the highest id it has seen (its cursor) and asks for everything after it.
//
// Rows are never pruned, so deletions stay visible to a phone that was offline
// for a long time. Each is a few bytes; a household makes a few thousand a year.
type ShoppingChange struct { //nolint:revive // stutters, but matches the other table names.
	ID       int64  `xorm:"bigint autoincr not null unique pk"`
	Entity   string `xorm:"varchar(20) not null unique(entity_row)"`
	EntityID int64  `xorm:"bigint not null unique(entity_row)"`
	// UID is the item's client-made id; phones key items by it. Empty for
	// other entities.
	UID     string    `xorm:"varchar(36) not null default '' index"`
	Deleted bool      `xorm:"bool not null default false"`
	Created time.Time `xorm:"created not null"`
}

// TableName sets the database table name.
func (*ShoppingChange) TableName() string {
	return "shopping_changes"
}

// ChangedEvent fires after a transaction that changed shopping data
// commits. The websocket listener tells every open app to sync.
type ChangedEvent struct{}

// Name is the event name in Vikunja's event system.
func (*ChangedEvent) Name() string {
	return "household.shopping.changed"
}

// changedListener pushes ChangedEvent to every open app over the
// websocket. The message carries no data: apps respond by syncing.
type changedListener struct{}

func (*changedListener) Name() string { return "household.shopping.push" }

func (*changedListener) Handle(_ *message.Message) error {
	if hub := ws.GetHub(); hub != nil {
		hub.PublishToAll(ws.EventHouseholdShoppingChanged, nil)
	}
	return nil
}

// recordChange logs that an entity changed (or was deleted) in this
// transaction, and queues the "changed" event for when it commits.
func recordChange(s *xorm.Session, entity string, id int64, uid string, deleted bool) error {
	if _, err := s.Where("entity = ? AND entity_id = ?", entity, id).Delete(&ShoppingChange{}); err != nil {
		return err
	}
	if _, err := s.Insert(&ShoppingChange{Entity: entity, EntityID: id, UID: uid, Deleted: deleted}); err != nil {
		return err
	}
	// Dispatched by handler.Do* (or our own handlers) after s.Commit().
	events.DispatchOnCommit(s, &ChangedEvent{})
	return nil
}

func recordChanges(s *xorm.Session, entity string, ids []int64) error {
	for _, id := range ids {
		if err := recordChange(s, entity, id, "", false); err != nil {
			return err
		}
	}
	return nil
}

// recordItemChange logs a changed or deleted item.
func recordItemChange(s *xorm.Session, item *ShoppingItem, deleted bool) error {
	return recordChange(s, entityItem, item.ID, item.UID, deleted)
}

// latestChangeID is the cursor that covers everything committed so far.
func latestChangeID(s *xorm.Session) (int64, error) {
	var id int64
	_, err := s.Table("shopping_changes").Select("COALESCE(MAX(id), 0)").Get(&id)
	return id, err
}

// itemDeleted reports whether an item with this uid existed and was deleted,
// so a replayed "create" from a phone doesn't bring it back.
func itemDeleted(s *xorm.Session, uid string) (bool, error) {
	return s.Where("entity = ? AND uid = ? AND deleted = ?", entityItem, uid, true).Exist(&ShoppingChange{})
}
