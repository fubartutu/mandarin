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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// ptr returns a pointer to v, for the optional fields of an op.
func ptr[T any](v T) *T { return &v }

// ms turns minutes after a fixed (past) morning into an op time.
func ms(minute int) int64 {
	return time.Date(2026, 10, 1, 9, minute, 0, 0, time.UTC).UnixMilli()
}

func createOp(listID int64, uid, name string, minute int) ShoppingSyncOp {
	return ShoppingSyncOp{ID: uuid.NewString(), Type: OpItemCreate, At: ms(minute), ItemUID: uid, ListID: listID,
		Set: &ShoppingItemFields{Name: ptr(name)}}
}

func tickOp(uid string, done bool, minute int) ShoppingSyncOp {
	return ShoppingSyncOp{ID: uuid.NewString(), Type: OpItemUpdate, At: ms(minute), ItemUID: uid,
		Set: &ShoppingItemFields{Done: ptr(done)}, Before: &ShoppingItemFields{Done: ptr(!done)}}
}

// apply runs ops as user1 and expects none to be rejected.
func apply(t *testing.T, s *xorm.Session, ops ...ShoppingSyncOp) {
	t.Helper()
	rejected, err := ApplySyncOps(s, user1, ops)
	require.NoError(t, err)
	require.Empty(t, rejected)
}

func itemByUID(t *testing.T, s *xorm.Session, uid string) *ShoppingItem {
	t.Helper()
	item, exists, err := getItemByUID(s, uid)
	require.NoError(t, err)
	require.True(t, exists, "item %s exists", uid)
	return item
}

func conflicts(t *testing.T, s *xorm.Session) []*ShoppingConflict {
	t.Helper()
	all := []*ShoppingConflict{}
	require.NoError(t, s.OrderBy("id").Find(&all))
	return all
}

func TestSyncCreate(t *testing.T) {
	t.Run("adds the item with the phone's uid and time", func(t *testing.T) {
		s := setup(t)
		list := groceries(t, s)
		uid := uuid.NewString()
		op := createOp(list.ID, uid, "  Milk ", 5)
		op.Set.Done = ptr(true)
		apply(t, s, op)

		item := itemByUID(t, s, uid)
		assert.Equal(t, "Milk", item.Name)
		assert.Equal(t, list.ID, item.ListID)
		assert.Equal(t, int64(1), item.CreatedByID)
		require.NotNil(t, item.DoneAt)
		assert.Equal(t, ms(5), item.DoneAt.UnixMilli(), "ticked when the phone ticked it")
		assert.Equal(t, catalogEntry(t, s, "milk").CategoryID, item.CategoryID, "category from the catalogue")
	})
	t.Run("sending it twice adds it once", func(t *testing.T) {
		s := setup(t)
		op := createOp(groceries(t, s).ID, uuid.NewString(), "Milk", 5)
		apply(t, s, op, op)
		count, err := s.Where("name = ?", "Milk").Count(&ShoppingItem{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
	t.Run("a create replayed after a delete doesn't bring the item back", func(t *testing.T) {
		s := setup(t)
		uid := uuid.NewString()
		op := createOp(groceries(t, s).ID, uid, "Milk", 5)
		apply(t, s, op, ShoppingSyncOp{ID: "d", Type: OpItemDelete, At: ms(6), ItemUID: uid}, op)
		_, exists, err := getItemByUID(s, uid)
		require.NoError(t, err)
		assert.False(t, exists)
	})
	t.Run("a category deleted meanwhile becomes none", func(t *testing.T) {
		s := setup(t)
		uid := uuid.NewString()
		op := createOp(groceries(t, s).ID, uid, "Thing nobody has added", 5)
		op.Set.CategoryID = ptr(int64(9999))
		apply(t, s, op)
		assert.Equal(t, int64(0), itemByUID(t, s, uid).CategoryID)
	})
}

func TestSyncUpdate(t *testing.T) {
	t.Run("the later change wins, and the earlier one is logged", func(t *testing.T) {
		s := setup(t)
		uid := uuid.NewString()
		apply(t, s, createOp(groceries(t, s).ID, uid, "Milk", 0))

		// Phone B unticks at 9:10 and syncs first; phone A ticked at 9:05 offline.
		rejected, err := ApplySyncOps(s, user2, []ShoppingSyncOp{tickOp(uid, true, 10), tickOp(uid, false, 11)})
		require.NoError(t, err)
		require.Empty(t, rejected)
		apply(t, s, tickOp(uid, true, 5))

		item := itemByUID(t, s, uid)
		assert.False(t, item.Done, "9:11 beats 9:05")
		require.Len(t, conflicts(t, s), 1)
		c := conflicts(t, s)[0]
		assert.Equal(t, fieldDone, c.Field)
		assert.Equal(t, "false", c.WinnerValue)
		assert.Equal(t, int64(2), c.WinnerByID)
		assert.Equal(t, "true", c.LoserValue)
		assert.Equal(t, int64(1), c.LoserByID)
		assert.Equal(t, "Milk", c.ItemName)
	})
	t.Run("a later change that overwrites one the phone never saw is logged", func(t *testing.T) {
		s := setup(t)
		uid := uuid.NewString()
		apply(t, s, createOp(groceries(t, s).ID, uid, "Milk", 0))
		// user2 sets 2 L at 9:05; user1, still seeing no amount, sets 1 L at 9:07.
		_, err := ApplySyncOps(s, user2, []ShoppingSyncOp{{ID: "a", Type: OpItemUpdate, At: ms(5), ItemUID: uid,
			Set: &ShoppingItemFields{Quantity: ptr(2.0), Unit: ptr("L")}, Before: &ShoppingItemFields{Quantity: ptr(0.0), Unit: ptr("")}}})
		require.NoError(t, err)
		apply(t, s, ShoppingSyncOp{ID: "b", Type: OpItemUpdate, At: ms(7), ItemUID: uid,
			Set: &ShoppingItemFields{Quantity: ptr(1.0), Unit: ptr("L")}, Before: &ShoppingItemFields{Quantity: ptr(0.0), Unit: ptr("")}})

		item := itemByUID(t, s, uid)
		assert.InDelta(t, 1.0, item.Quantity, 0)
		require.Len(t, conflicts(t, s), 1)
		assert.Equal(t, "1 L", conflicts(t, s)[0].WinnerValue)
		assert.Equal(t, "2 L", conflicts(t, s)[0].LoserValue)
		assert.Equal(t, fieldAmount, conflicts(t, s)[0].Field)
	})
	t.Run("changes to different fields both survive", func(t *testing.T) {
		s := setup(t)
		uid := uuid.NewString()
		apply(t, s, createOp(groceries(t, s).ID, uid, "Milk", 0))
		_, err := ApplySyncOps(s, user2, []ShoppingSyncOp{{ID: "a", Type: OpItemUpdate, At: ms(9), ItemUID: uid,
			Set: &ShoppingItemFields{Note: ptr("lactose free")}, Before: &ShoppingItemFields{Note: ptr("")}}})
		require.NoError(t, err)
		apply(t, s, tickOp(uid, true, 5))

		item := itemByUID(t, s, uid)
		assert.True(t, item.Done)
		assert.Equal(t, "lactose free", item.Note)
		assert.Empty(t, conflicts(t, s))
	})
	t.Run("an edit made online counts from when it was saved", func(t *testing.T) {
		s := setup(t)
		uid := uuid.NewString()
		apply(t, s, createOp(groceries(t, s).ID, uid, "Milk", 0))
		item := itemByUID(t, s, uid)
		item.Note = "edited on the laptop"
		require.NoError(t, item.Update(s, user2))

		// A phone's tick from earlier still applies: it's a different field.
		apply(t, s, tickOp(uid, true, 5))
		item = itemByUID(t, s, uid)
		assert.True(t, item.Done)
		assert.Equal(t, "edited on the laptop", item.Note)
	})
	t.Run("a time in the future counts as now", func(t *testing.T) {
		s := setup(t)
		uid := uuid.NewString()
		apply(t, s, createOp(groceries(t, s).ID, uid, "Milk", 0))
		future := tickOp(uid, true, 0)
		future.At = time.Now().Add(24 * time.Hour).UnixMilli()
		apply(t, s, future)
		assert.LessOrEqual(t, itemByUID(t, s, uid).FieldTimes[fieldDone].At, time.Now().UnixMilli())
	})
	t.Run("sending it twice changes nothing more", func(t *testing.T) {
		s := setup(t)
		uid := uuid.NewString()
		apply(t, s, createOp(groceries(t, s).ID, uid, "Milk", 0))
		op := tickOp(uid, true, 5)
		apply(t, s, op, op)
		assert.True(t, itemByUID(t, s, uid).Done)
		assert.Empty(t, conflicts(t, s))
	})
	t.Run("an edit to a removed item loses to the removal", func(t *testing.T) {
		s := setup(t)
		uid := uuid.NewString()
		apply(t, s, createOp(groceries(t, s).ID, uid, "Milk", 0),
			ShoppingSyncOp{ID: "d", Type: OpItemDelete, At: ms(10), ItemUID: uid})
		op := tickOp(uid, true, 5)
		op.ItemName = "Milk"
		apply(t, s, op)

		_, exists, err := getItemByUID(s, uid)
		require.NoError(t, err)
		assert.False(t, exists)
		require.Len(t, conflicts(t, s), 1)
		assert.Equal(t, fieldDeleted, conflicts(t, s)[0].Field)
		assert.Equal(t, "Milk", conflicts(t, s)[0].ItemName)
	})
}

func TestSyncRejects(t *testing.T) {
	s := setup(t)
	list := groceries(t, s)
	good := createOp(list.ID, uuid.NewString(), "Bread", 0)
	noName := createOp(list.ID, uuid.NewString(), " ", 0)
	noList := createOp(9999, uuid.NewString(), "Eggs", 0)
	badUID := createOp(list.ID, "not-a-uuid", "Eggs", 0)
	unknown := ShoppingSyncOp{ID: "x", Type: "item.teleport"}
	missing := tickOp(uuid.NewString(), true, 0)

	rejected, err := ApplySyncOps(s, user1, []ShoppingSyncOp{noName, good, noList, badUID, unknown, missing})
	require.NoError(t, err)

	ids := []string{}
	for _, r := range rejected {
		ids = append(ids, r.OpID)
		assert.NotEmpty(t, r.Reason)
	}
	assert.Equal(t, []string{noName.ID, noList.ID, badUID.ID, unknown.ID, missing.ID}, ids)
	itemByUID(t, s, good.ItemUID) // the good op still went through
}

func TestSyncNotStocked(t *testing.T) {
	s := setup(t)
	list := groceries(t, s)
	aldi := addShop(t, s, list.ID, "Aldi")
	uid := uuid.NewString()
	apply(t, s, createOp(list.ID, uid, "Vegemite", 0),
		ShoppingSyncOp{ID: "n", Type: OpItemNotStocked, At: ms(1), ItemUID: uid, ShopIDs: []int64{aldi.ID, 9999}})

	entry := catalogEntry(t, s, "Vegemite")
	require.NoError(t, fillDetails(s, entry))
	assert.Equal(t, []int64{aldi.ID}, entry.NotStockedShopIDs, "unknown shop dropped")
}

func TestPullChanges(t *testing.T) {
	t.Run("cursor 0 gets everything", func(t *testing.T) {
		s := setup(t)
		apply(t, s, createOp(groceries(t, s).ID, uuid.NewString(), "Milk", 0))

		result, err := PullChanges(s, user1, 0)
		require.NoError(t, err)
		assert.True(t, result.Full)
		assert.Len(t, result.Lists, 1)
		assert.Len(t, result.Categories, len(DefaultCategories))
		assert.Greater(t, len(result.Catalog), 200)
		assert.Len(t, result.Items, 1)
		assert.Positive(t, result.Cursor)
	})
	t.Run("a later pull gets only what changed", func(t *testing.T) {
		s := setup(t)
		list := groceries(t, s)
		milk, bread := uuid.NewString(), uuid.NewString()
		apply(t, s, createOp(list.ID, milk, "Milk", 0), createOp(list.ID, bread, "Bread", 0))
		first, err := PullChanges(s, user1, 0)
		require.NoError(t, err)

		apply(t, s, tickOp(milk, true, 5), ShoppingSyncOp{ID: "d", Type: OpItemDelete, At: ms(6), ItemUID: bread})
		addShop(t, s, list.ID, "Aldi")

		result, err := PullChanges(s, user1, first.Cursor)
		require.NoError(t, err)
		assert.False(t, result.Full)
		require.Len(t, result.Items, 1)
		assert.Equal(t, milk, result.Items[0].UID)
		assert.True(t, result.Items[0].Done)
		assert.Equal(t, []string{bread}, result.DeletedItemUIDs)
		require.Len(t, result.Lists, 1, "the list whose shops changed")
		assert.Len(t, result.Lists[0].Shops, 1)
		assert.Empty(t, result.Categories)
		assert.Greater(t, result.Cursor, first.Cursor)

		again, err := PullChanges(s, user1, result.Cursor)
		require.NoError(t, err)
		assert.Empty(t, again.Items)
		assert.Equal(t, result.Cursor, again.Cursor)
	})
	t.Run("an unknown cursor gets everything", func(t *testing.T) {
		s := setup(t)
		result, err := PullChanges(s, user1, 1_000_000)
		require.NoError(t, err)
		assert.True(t, result.Full)
	})
	t.Run("link shares get nothing", func(t *testing.T) {
		s := setup(t)
		_, err := PullChanges(s, linkShare, 0)
		require.Error(t, err)
		_, err = ApplySyncOps(s, linkShare, nil)
		require.Error(t, err)
	})
}

func TestSyncMergesDuplicates(t *testing.T) {
	milk := func(listID int64, uid string, minute int) ShoppingSyncOp {
		op := createOp(listID, uid, "Milk", minute)
		op.Set.Quantity, op.Set.Unit = ptr(2.0), ptr("L")
		return op
	}
	countMilk := func(t *testing.T, s *xorm.Session) int64 {
		t.Helper()
		count, err := s.Where("LOWER(name) = ?", "milk").Count(&ShoppingItem{})
		require.NoError(t, err)
		return count
	}

	t.Run("an identical add is folded into the item already there", func(t *testing.T) {
		s := setup(t)
		list := groceries(t, s)
		first, second := uuid.NewString(), uuid.NewString()
		apply(t, s, milk(list.ID, first, 0))
		secondAdd := milk(list.ID, second, 1)
		secondAdd.Set.Name = ptr(" milk ") // case and spaces don't matter
		apply(t, s, secondAdd, secondAdd)
		assert.Equal(t, int64(1), countMilk(t, s))

		// The second phone's later edits use its own uid and reach the merged item.
		note := ShoppingSyncOp{ID: uuid.NewString(), Type: OpItemUpdate, At: ms(2), ItemUID: second,
			Set: &ShoppingItemFields{Note: ptr("lactose free")}, Before: &ShoppingItemFields{Note: ptr("")}}
		apply(t, s, note, tickOp(second, true, 3))
		item := itemByUID(t, s, first)
		assert.Equal(t, "lactose free", item.Note)
		assert.True(t, item.Done)

		apply(t, s, ShoppingSyncOp{ID: uuid.NewString(), Type: OpItemDelete, At: ms(4), ItemUID: second})
		assert.Equal(t, int64(0), countMilk(t, s))
	})
	t.Run("different amounts, notes, ticked items and other lists stay separate", func(t *testing.T) {
		s := setup(t)
		list := groceries(t, s)
		other := &ShoppingList{Title: "Kmart"}
		require.NoError(t, other.Create(s, user1))

		ticked := uuid.NewString()
		apply(t, s, milk(list.ID, ticked, 0), tickOp(ticked, true, 1))
		apply(t, s, milk(list.ID, uuid.NewString(), 2)) // the ticked one doesn't count
		oneLitre := milk(list.ID, uuid.NewString(), 3)
		oneLitre.Set.Quantity = ptr(1.0)
		withNote := milk(list.ID, uuid.NewString(), 4)
		withNote.Set.Note = ptr("for the cake")
		apply(t, s, oneLitre, withNote, milk(other.ID, uuid.NewString(), 5))
		assert.Equal(t, int64(5), countMilk(t, s))
	})
}
