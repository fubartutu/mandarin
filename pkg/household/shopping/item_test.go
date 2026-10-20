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

func TestShoppingItem_Permissions(t *testing.T) {
	t.Run("members can use any item", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Milk"})

		read := &ShoppingItem{ListID: item.ListID, ID: item.ID}
		can, _, err := read.CanRead(s, user2)
		require.NoError(t, err)
		assert.True(t, can)
		assert.Equal(t, "Milk", read.Name)

		can, err = (&ShoppingItem{ListID: item.ListID, ID: item.ID}).CanUpdate(s, user2)
		require.NoError(t, err)
		assert.True(t, can)

		can, err = (&ShoppingItem{ListID: item.ListID, ID: item.ID}).CanDelete(s, user2)
		require.NoError(t, err)
		assert.True(t, can)
	})

	t.Run("link shares are refused", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Milk"})

		_, err := (&ShoppingItem{ListID: item.ListID}).CanCreate(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, _, err = (&ShoppingItem{ListID: item.ListID, ID: item.ID}).CanRead(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, err = (&ShoppingItem{ListID: item.ListID, ID: item.ID}).CanUpdate(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, err = (&ShoppingItem{ListID: item.ListID, ID: item.ID}).CanDelete(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, _, _, err = (&ShoppingItem{ListID: item.ListID}).ReadAll(s, linkShare, "", 0, 0)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
	})

	t.Run("an item is only reachable through its own list", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Milk"})
		other := &ShoppingList{Title: "Kmart"}
		require.NoError(t, other.Create(s, user1))

		_, _, err := (&ShoppingItem{ListID: other.ID, ID: item.ID}).CanRead(s, user1)
		require.ErrorIs(t, err, ErrItemDoesNotExist{ItemID: item.ID})
		_, err = (&ShoppingItem{ListID: other.ID, ID: item.ID}).CanUpdate(s, user1)
		require.ErrorIs(t, err, ErrItemDoesNotExist{ItemID: item.ID})
		_, err = (&ShoppingItem{ListID: other.ID, ID: item.ID}).CanDelete(s, user1)
		require.ErrorIs(t, err, ErrItemDoesNotExist{ItemID: item.ID})
	})

	t.Run("can't add to a list that doesn't exist", func(t *testing.T) {
		s := setup(t)
		_, err := (&ShoppingItem{ListID: 9999, Name: "Milk"}).CanCreate(s, user1)
		require.ErrorIs(t, err, ErrListDoesNotExist{ListID: 9999})
		_, _, _, err = (&ShoppingItem{ListID: 9999}).ReadAll(s, user1, "", 0, 0)
		require.ErrorIs(t, err, ErrListDoesNotExist{ListID: 9999})
	})
}

func TestShoppingItem_CRUD(t *testing.T) {
	t.Run("create with quantity, unit and category", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Mince", Quantity: 500, Unit: "g", CategoryID: 3})
		assert.Equal(t, int64(1), item.CreatedByID)
		assert.Nil(t, item.DoneAt)

		stored, err := getItem(s, item.ListID, item.ID)
		require.NoError(t, err)
		assert.InDelta(t, 500, stored.Quantity, 0)
		assert.Equal(t, "g", stored.Unit)
		assert.Equal(t, int64(3), stored.CategoryID)
	})

	t.Run("unknown category is rejected", func(t *testing.T) {
		s := setup(t)
		err := (&ShoppingItem{ListID: groceries(t, s).ID, Name: "Milk", CategoryID: 9999}).Create(s, user1)
		require.ErrorIs(t, err, ErrCategoryDoesNotExist{CategoryID: 9999})
	})

	t.Run("ticking stamps done_at and unticking clears it", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Milk"})

		item.Done = true
		require.NoError(t, item.Update(s, user2))
		require.NotNil(t, item.DoneAt)
		tickedAt := *item.DoneAt

		// Editing a ticked item keeps the original tick time.
		item.Note = "full cream"
		require.NoError(t, item.Update(s, user2))
		require.NotNil(t, item.DoneAt)
		assert.True(t, tickedAt.Equal(*item.DoneAt))

		item.Done = false
		require.NoError(t, item.Update(s, user2))
		assert.Nil(t, item.DoneAt)
	})

	t.Run("read all puts ticked items last", func(t *testing.T) {
		s := setup(t)
		listID := groceries(t, s).ID
		addItem(t, s, &ShoppingItem{ListID: listID, Name: "Milk", Done: true})
		addItem(t, s, &ShoppingItem{ListID: listID, Name: "Bread"})

		result, _, total, err := (&ShoppingItem{ListID: listID}).ReadAll(s, user2, "", 0, 0)
		require.NoError(t, err)
		items := result.([]*ShoppingItem)
		assert.Equal(t, int64(2), total)
		assert.Equal(t, "Bread", items[0].Name)
		assert.Equal(t, "Milk", items[1].Name)
		require.NotNil(t, items[0].CreatedBy)
	})

	t.Run("delete", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Milk"})
		require.NoError(t, (&ShoppingItem{ListID: item.ListID, ID: item.ID}).Delete(s, user1))
		_, err := getItem(s, item.ListID, item.ID)
		require.ErrorIs(t, err, ErrItemDoesNotExist{ItemID: item.ID})
	})
}
