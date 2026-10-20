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

	"code.vikunja.io/api/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShoppingList_Permissions(t *testing.T) {
	// t.Run makes named sub-tests: `go test -run 'TestShoppingList_Permissions/link'`.
	t.Run("every member can read, update and delete every list", func(t *testing.T) {
		s := setup(t)
		list := &ShoppingList{Title: "Kmart"}
		require.NoError(t, list.Create(s, user1))

		read := &ShoppingList{ID: list.ID}
		can, permission, err := read.CanRead(s, user2)
		require.NoError(t, err)
		assert.True(t, can)
		assert.Equal(t, int(models.PermissionAdmin), permission)
		assert.Equal(t, "Kmart", read.Title, "CanRead loads the row")

		can, err = (&ShoppingList{ID: list.ID}).CanUpdate(s, user2)
		require.NoError(t, err)
		assert.True(t, can)

		can, err = (&ShoppingList{ID: list.ID}).CanDelete(s, user2)
		require.NoError(t, err)
		assert.True(t, can)
	})

	t.Run("link shares are not household members", func(t *testing.T) {
		s := setup(t)
		id := groceries(t, s).ID

		can, err := (&ShoppingList{}).CanCreate(s, linkShare)
		require.NoError(t, err)
		assert.False(t, can)

		_, _, err = (&ShoppingList{ID: id}).CanRead(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})

		_, err = (&ShoppingList{ID: id}).CanUpdate(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})

		_, err = (&ShoppingList{ID: id}).CanDelete(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})

		_, _, _, err = (&ShoppingList{}).ReadAll(s, linkShare, "", 0, 0)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
	})

	t.Run("unknown list", func(t *testing.T) {
		s := setup(t)
		_, _, err := (&ShoppingList{ID: 9999}).CanRead(s, user1)
		require.ErrorIs(t, err, ErrListDoesNotExist{ListID: 9999})

		_, err = (&ShoppingList{ID: 9999}).CanUpdate(s, user1)
		require.ErrorIs(t, err, ErrListDoesNotExist{ListID: 9999})
	})
}

func TestShoppingList_CRUD(t *testing.T) {
	t.Run("create records the creator", func(t *testing.T) {
		s := setup(t)
		list := &ShoppingList{ID: 42, Title: "Kmart"}
		require.NoError(t, list.Create(s, user1))
		assert.NotEqual(t, int64(42), list.ID, "client-sent ids are ignored")
		assert.Equal(t, int64(1), list.CreatedByID)
	})

	t.Run("new lists go to the end unless given a position", func(t *testing.T) {
		s := setup(t)
		kmart := &ShoppingList{Title: "Kmart"}
		require.NoError(t, kmart.Create(s, user1))
		assert.InDelta(t, groceries(t, s).Position+1, kmart.Position, 0)
	})

	t.Run("read all is ordered by position and searchable", func(t *testing.T) {
		s := setup(t)
		require.NoError(t, (&ShoppingList{Title: "Kmart", Position: 0.5}).Create(s, user1))

		result, count, total, err := (&ShoppingList{}).ReadAll(s, user2, "", 0, 0)
		require.NoError(t, err)
		lists := result.([]*ShoppingList)
		assert.Equal(t, 2, count)
		assert.Equal(t, int64(2), total)
		assert.Equal(t, "Kmart", lists[0].Title)
		assert.Equal(t, "Groceries", lists[1].Title)
		assert.Nil(t, lists[1].CreatedBy, "seeded list has no creator")
		require.NotNil(t, lists[0].CreatedBy)
		assert.Equal(t, int64(1), lists[0].CreatedBy.ID)

		result, _, total, err = (&ShoppingList{}).ReadAll(s, user1, "kma", 0, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Kmart", result.([]*ShoppingList)[0].Title)
	})

	t.Run("update only touches title and position", func(t *testing.T) {
		s := setup(t)
		list := groceries(t, s)
		update := &ShoppingList{ID: list.ID, Title: "Food", Position: 3, CreatedByID: 2}
		require.NoError(t, update.Update(s, user1))
		assert.Equal(t, "Food", update.Title)
		assert.Equal(t, int64(0), update.CreatedByID, "created_by is not writable")
	})

	t.Run("delete removes the list's items", func(t *testing.T) {
		s := setup(t)
		list := groceries(t, s)
		addItem(t, s, &ShoppingItem{ListID: list.ID, Name: "Milk"})

		require.NoError(t, (&ShoppingList{ID: list.ID}).Delete(s, user1))

		count, err := s.Where("list_id = ?", list.ID).Count(&ShoppingItem{})
		require.NoError(t, err)
		assert.Zero(t, count)
	})

	t.Run("clear done only removes ticked items", func(t *testing.T) {
		s := setup(t)
		list := groceries(t, s)
		addItem(t, s, &ShoppingItem{ListID: list.ID, Name: "Milk", Done: true})
		addItem(t, s, &ShoppingItem{ListID: list.ID, Name: "Bread"})

		deleted, err := list.ClearDoneItems(s)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		var left []*ShoppingItem
		require.NoError(t, s.Where("list_id = ?", list.ID).Find(&left))
		require.Len(t, left, 1)
		assert.Equal(t, "Bread", left[0].Name)
	})
}
