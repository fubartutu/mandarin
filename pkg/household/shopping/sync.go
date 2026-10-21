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
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

// Offline sync. A phone keeps a copy of the household's shopping data, makes
// changes to it straight away, and queues them as ops. When it has signal it
// sends the queue with its cursor (the last change-log id it has seen); the
// server applies the ops and answers with everything changed since the
// cursor. See SPEC.md, "Offline sync".
//
// Ops are safe to send twice (the phone may not hear the first answer):
// creates are skipped when the uid exists, updates set fields to values,
// deletes of a missing item do nothing.

// Op types.
const (
	OpItemCreate     = "item.create"
	OpItemUpdate     = "item.update"
	OpItemDelete     = "item.delete"
	OpItemNotStocked = "item.not_stocked"
)

// ShoppingItemFields is part of an item. A nil field is "not included".
// Limits are checked per op (see validate), not by the API schema, so one bad
// op is rejected on its own instead of failing the whole sync.
type ShoppingItemFields struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	Name       *string  `json:"name,omitempty" doc:"1–250 characters."`
	Quantity   *float64 `json:"quantity,omitempty" doc:"0 or more; sent together with unit."`
	Unit       *string  `json:"unit,omitempty" doc:"Up to 50 characters; sent together with quantity."`
	Note       *string  `json:"note,omitempty" doc:"Up to 2000 characters."`
	CategoryID *int64   `json:"category_id,omitempty" doc:"0 for none. An unknown category becomes 0."`
	Done       *bool    `json:"done,omitempty"`
	Position   *float64 `json:"position,omitempty"`
}

// ShoppingSyncOp is one change made on a phone.
type ShoppingSyncOp struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	ID       string              `json:"id" doc:"The app's id for this change; echoed back if it's rejected."`
	Type     string              `json:"type" doc:"item.create, item.update, item.delete or item.not_stocked."`
	At       int64               `json:"at" doc:"When the change was made, in Unix milliseconds. Per field, the latest change wins. Times in the future count as now."`
	ItemUID  string              `json:"item_uid" doc:"The item's uid; for item.create a new UUID made by the app."`
	ItemName string              `json:"item_name,omitempty" doc:"The item's name as the app knew it, for the conflict log."`
	ListID   int64               `json:"list_id,omitempty" doc:"item.create: the list to add to."`
	Set      *ShoppingItemFields `json:"set,omitempty" doc:"item.create: the new item. item.update: only the changed fields."`
	Before   *ShoppingItemFields `json:"before,omitempty" doc:"item.update: the changed fields as the app had them before; used to spot conflicts."`
	ShopIDs  []int64             `json:"shop_ids,omitempty" doc:"item.not_stocked: every shop that doesn't stock the item (replaces the marks)."`
}

// ShoppingSyncRejection says an op was dropped and why. The app discards it;
// the data it gets back shows the server's version.
type ShoppingSyncRejection struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	OpID   string `json:"op_id"`
	Reason string `json:"reason"`
}

// errRejected marks a problem with one op, as opposed to a database error
// that should fail the whole sync.
type errRejected struct{ reason string }

func (e errRejected) Error() string { return e.reason }

// rejectionReason returns the reason when err rejects just this op.
func rejectionReason(err error) (string, bool) {
	var rejected errRejected
	if errors.As(err, &rejected) {
		return rejected.reason, true
	}
	// Our not-found errors (an item on a deleted list, say) also reject the op.
	var domain web.HTTPErrorProcessor
	if errors.As(err, &domain) {
		return domain.HTTPError().Message, true
	}
	return "", false
}

// ApplySyncOps applies a phone's ops in order and returns the rejected ones.
// An error means the whole batch failed and the transaction must roll back.
func ApplySyncOps(s *xorm.Session, a web.Auth, ops []ShoppingSyncOp) ([]*ShoppingSyncRejection, error) {
	if err := requireMember(a); err != nil {
		return nil, err
	}
	u, err := user.GetFromAuth(a)
	if err != nil {
		return nil, err
	}

	rejected := []*ShoppingSyncRejection{}
	now := time.Now()
	for i := range ops {
		op := &ops[i]
		// A phone with its clock ahead mustn't win every future conflict.
		at := time.UnixMilli(min(op.At, now.UnixMilli()))
		var err error
		if op.Type != OpItemCreate {
			// An edit to an add that was merged acts on the merged item.
			if op.ItemUID, err = resolveUID(s, op.ItemUID); err != nil {
				return nil, err
			}
		}
		switch op.Type {
		case OpItemCreate:
			err = syncCreate(s, u, op, at)
		case OpItemUpdate:
			err = syncUpdate(s, u, op, at)
		case OpItemDelete:
			err = syncDelete(s, op)
		case OpItemNotStocked:
			err = syncNotStocked(s, op)
		default:
			err = errRejected{"unknown change type " + op.Type}
		}
		if err == nil {
			continue
		}
		reason, ok := rejectionReason(err)
		if !ok {
			return nil, err
		}
		rejected = append(rejected, &ShoppingSyncRejection{OpID: op.ID, Reason: reason})
	}
	return rejected, nil
}

func getItemByUID(s *xorm.Session, uid string) (*ShoppingItem, bool, error) {
	item := &ShoppingItem{}
	exists, err := s.Where("uid = ?", uid).Get(item)
	return item, exists, err
}

// validate checks the fields that are present against the item limits.
func (f *ShoppingItemFields) validate() error {
	if f.Name != nil {
		*f.Name = strings.TrimSpace(*f.Name)
		if n := utf8.RuneCountInString(*f.Name); n < 1 || n > 250 {
			return errRejected{"name must be 1–250 characters"}
		}
	}
	if (f.Quantity == nil) != (f.Unit == nil) {
		return errRejected{"quantity and unit go together"}
	}
	if f.Quantity != nil && *f.Quantity < 0 {
		return errRejected{"quantity can't be negative"}
	}
	if f.Unit != nil && utf8.RuneCountInString(*f.Unit) > 50 {
		return errRejected{"unit must be at most 50 characters"}
	}
	if f.Note != nil && utf8.RuneCountInString(*f.Note) > 2000 {
		return errRejected{"note must be at most 2000 characters"}
	}
	return nil
}

// values returns the included fields, keyed like fieldValue.
func (f *ShoppingItemFields) values() map[string]any {
	v := map[string]any{}
	if f == nil {
		return v
	}
	if f.Name != nil {
		v[fieldName] = *f.Name
	}
	if f.Quantity != nil && f.Unit != nil {
		v[fieldAmount] = amount{*f.Quantity, *f.Unit}
	}
	if f.Note != nil {
		v[fieldNote] = *f.Note
	}
	if f.CategoryID != nil {
		v[fieldCategory] = *f.CategoryID
	}
	if f.Done != nil {
		v[fieldDone] = *f.Done
	}
	if f.Position != nil {
		v[fieldPosition] = *f.Position
	}
	return v
}

// knownCategory turns a category that was deleted meanwhile into 0, so the
// item still arrives, just uncategorised.
func knownCategory(s *xorm.Session, id int64) (int64, error) {
	if id == 0 {
		return 0, nil
	}
	exists, err := s.Where("id = ?", id).Exist(&ShoppingCategory{})
	if err != nil || !exists {
		return 0, err
	}
	return id, nil
}

func syncCreate(s *xorm.Session, u *user.User, op *ShoppingSyncOp, at time.Time) error {
	if _, err := uuid.Parse(op.ItemUID); err != nil {
		return errRejected{"item_uid must be a UUID"}
	}
	if op.Set == nil || op.Set.Name == nil {
		return errRejected{"a new item needs a name"}
	}
	if err := op.Set.validate(); err != nil {
		return err
	}
	// Already applied (or merged), or added and removed again before this arrived.
	if _, exists, err := getItemByUID(s, op.ItemUID); err != nil || exists {
		return err
	}
	if merged, err := resolveUID(s, op.ItemUID); err != nil || merged != op.ItemUID {
		return err
	}
	if deleted, err := itemDeleted(s, op.ItemUID); err != nil || deleted {
		return err
	}
	if _, err := getListByID(s, op.ListID); err != nil {
		return err
	}

	item := &ShoppingItem{UID: op.ItemUID, ListID: op.ListID}
	for field, value := range op.Set.values() {
		setFieldValue(item, field, value)
	}
	var err error
	if item.CategoryID, err = knownCategory(s, item.CategoryID); err != nil {
		return err
	}
	// Someone else added the same thing while this phone was offline.
	duplicate, err := findDuplicate(s, item)
	if err != nil {
		return err
	}
	if duplicate != nil {
		return mergeInto(s, op.ItemUID, duplicate)
	}
	return item.insert(s, u, at)
}

func syncUpdate(s *xorm.Session, u *user.User, op *ShoppingSyncOp, at time.Time) error {
	if op.Set == nil {
		return errRejected{"an update needs set"}
	}
	if err := op.Set.validate(); err != nil {
		return err
	}
	stored, exists, err := getItemByUID(s, op.ItemUID)
	if err != nil {
		return err
	}
	if !exists {
		return editOfDeletedItem(s, u, op, at)
	}

	mine := fieldStamp{At: at.UnixMilli(), By: u.ID}
	before := op.Before.values()
	updated := *stored
	stamps := stored.FieldTimes.with(nil, at, u.ID)
	changed := false

	for field, value := range op.Set.values() {
		if field == fieldCategory {
			if value, err = knownCategory(s, value.(int64)); err != nil {
				return err
			}
		}
		current := fieldValue(stored, field)
		theirs := stored.FieldTimes[field]
		if value == current {
			continue
		}
		// Position only orders the list; crossing reorders aren't worth a log entry.
		logged := field != fieldPosition

		if mine.At < theirs.At {
			// Someone changed this field after the phone did: theirs stays.
			if logged {
				if err := addConflict(s, stored, field, side{current, theirs}, side{value, mine}); err != nil {
					return err
				}
			}
			continue
		}

		setFieldValue(&updated, field, value)
		stamps[field] = mine
		changed = true
		// The phone's change wins. If the phone hadn't seen the value it
		// replaced, someone else's change is lost: log that.
		if seen, ok := before[field]; ok && seen != current && logged {
			if err := addConflict(s, stored, field, side{value, mine}, side{current, theirs}); err != nil {
				return err
			}
		}
	}

	if !changed {
		return nil
	}
	updated.setDoneAt(stored.Done, stored.DoneAt, at)
	updated.FieldTimes = stamps
	return updated.save(s)
}

// editOfDeletedItem logs a conflict for an edit to an item that has since
// been removed (the removal wins), or rejects an edit to an unknown item.
func editOfDeletedItem(s *xorm.Session, u *user.User, op *ShoppingSyncOp, at time.Time) error {
	deleted, err := itemDeleted(s, op.ItemUID)
	if err != nil {
		return err
	}
	if !deleted {
		return errRejected{"no item with this uid"}
	}
	gone := &ShoppingChange{}
	if _, err := s.Where("entity = ? AND uid = ?", entityItem, op.ItemUID).Get(gone); err != nil {
		return err
	}
	item := &ShoppingItem{UID: op.ItemUID, Name: op.ItemName}
	removal := side{"", fieldStamp{At: gone.Created.UnixMilli()}}
	edit := side{"", fieldStamp{At: at.UnixMilli(), By: u.ID}}
	return addConflict(s, item, fieldDeleted, removal, edit)
}

func syncDelete(s *xorm.Session, op *ShoppingSyncOp) error {
	stored, exists, err := getItemByUID(s, op.ItemUID)
	if err != nil || !exists {
		return err
	}
	return deleteItems(s, stored)
}

func syncNotStocked(s *xorm.Session, op *ShoppingSyncOp) error {
	stored, exists, err := getItemByUID(s, op.ItemUID)
	if err != nil {
		return err
	}
	if !exists || stored.CatalogItemID == 0 {
		return errRejected{"no catalogue entry for this item"}
	}
	// Shops deleted meanwhile are dropped rather than failing the op.
	var shopIDs []int64
	if len(op.ShopIDs) > 0 {
		if err := s.Table("shopping_shops").In("id", op.ShopIDs).Cols("id").Find(&shopIDs); err != nil {
			return err
		}
	}
	details, err := loadCatalogDetails(s, []int64{stored.CatalogItemID})
	if err != nil {
		return err
	}
	if sameIDs(details[stored.CatalogItemID].notStocked, shopIDs) {
		return nil // nothing to write
	}
	return setNotStocked(s, stored.CatalogItemID, shopIDs)
}

// sameIDs reports whether two id lists hold the same ids, in any order.
func sameIDs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[int64]bool, len(a))
	for _, id := range a {
		seen[id] = true
	}
	for _, id := range b {
		if !seen[id] {
			return false
		}
	}
	return true
}
