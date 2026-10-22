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

func TestShoppingShop(t *testing.T) {
	t.Run("shops belong to a list and come back with it, in tab order", func(t *testing.T) {
		s := setup(t)
		list := groceries(t, s)
		aldi := addShop(t, s, list.ID, "Aldi")
		supabarn := addShop(t, s, list.ID, "Supabarn")
		assert.Less(t, aldi.Position, supabarn.Position, "new shops go last")

		read := &ShoppingList{ID: list.ID}
		_, _, err := read.CanRead(s, user2)
		require.NoError(t, err)
		require.NoError(t, read.ReadOne(s, user2))
		require.Len(t, read.Shops, 2)
		assert.Equal(t, "Aldi", read.Shops[0].Title)
		assert.Equal(t, "Supabarn", read.Shops[1].Title)

		result, _, _, err := (&ShoppingList{}).ReadAll(s, user1, "", 0, 0)
		require.NoError(t, err)
		assert.Len(t, result.([]*ShoppingList)[0].Shops, 2)
	})

	t.Run("lists without shops have an empty shops array", func(t *testing.T) {
		s := setup(t)
		result, _, _, err := (&ShoppingList{}).ReadAll(s, user1, "", 0, 0)
		require.NoError(t, err)
		assert.NotNil(t, result.([]*ShoppingList)[0].Shops)
	})

	t.Run("a shop is only reachable through its own list", func(t *testing.T) {
		s := setup(t)
		aldi := addShop(t, s, groceries(t, s).ID, "Aldi")
		kmart := &ShoppingList{Title: "Kmart"}
		require.NoError(t, kmart.Create(s, user1))

		_, _, err := (&ShoppingShop{ListID: kmart.ID, ID: aldi.ID}).CanRead(s, user1)
		require.ErrorIs(t, err, ErrShopDoesNotExist{ShopID: aldi.ID})
		_, err = (&ShoppingShop{ListID: kmart.ID, ID: aldi.ID}).CanDelete(s, user1)
		require.ErrorIs(t, err, ErrShopDoesNotExist{ShopID: aldi.ID})
	})

	t.Run("link shares are refused", func(t *testing.T) {
		s := setup(t)
		listID := groceries(t, s).ID
		aldi := addShop(t, s, listID, "Aldi")

		_, err := (&ShoppingShop{ListID: listID}).CanCreate(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, _, err = (&ShoppingShop{ListID: listID, ID: aldi.ID}).CanRead(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, err = (&ShoppingShop{ListID: listID, ID: aldi.ID}).CanUpdate(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, _, _, err = (&ShoppingShop{ListID: listID}).ReadAll(s, linkShare, "", 0, 0)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
	})

	t.Run("deleting a shop removes its not-stocked marks", func(t *testing.T) {
		s := setup(t)
		listID := groceries(t, s).ID
		aldi := addShop(t, s, listID, "Aldi")
		require.NoError(t, setNotStocked(s, catalogEntry(t, s, "Vegemite").ID, []int64{aldi.ID}))

		require.NoError(t, (&ShoppingShop{ListID: listID, ID: aldi.ID}).Delete(s, user1))

		count, err := s.Count(&ShoppingNotStocked{})
		require.NoError(t, err)
		assert.Zero(t, count)
	})

	t.Run("deleting a list removes its shops", func(t *testing.T) {
		s := setup(t)
		list := groceries(t, s)
		aldi := addShop(t, s, list.ID, "Aldi")
		require.NoError(t, setNotStocked(s, catalogEntry(t, s, "Vegemite").ID, []int64{aldi.ID}))

		require.NoError(t, (&ShoppingList{ID: list.ID}).Delete(s, user1))

		for _, bean := range []any{&ShoppingShop{}, &ShoppingNotStocked{}} {
			count, err := s.Count(bean)
			require.NoError(t, err)
			assert.Zero(t, count)
		}
	})

	t.Run("the aisle order keeps known categories once each, in the order given", func(t *testing.T) {
		s := setup(t)
		listID := groceries(t, s).ID
		aldi := addShop(t, s, listID, "Aldi")
		categories := []*ShoppingCategory{}
		require.NoError(t, s.OrderBy("position").Limit(3).Find(&categories))
		a, b, c := categories[0].ID, categories[1].ID, categories[2].ID

		update := &ShoppingShop{ListID: listID, ID: aldi.ID, Title: "Aldi", Position: aldi.Position,
			CategoryOrder: []int64{c, 9999, a, c, b}}
		require.NoError(t, update.Update(s, user1))
		assert.Equal(t, []int64{c, a, b}, update.CategoryOrder)

		read := &ShoppingList{ID: listID}
		_, _, err := read.CanRead(s, user1)
		require.NoError(t, err)
		require.NoError(t, read.ReadOne(s, user1))
		assert.Equal(t, []int64{c, a, b}, read.Shops[0].CategoryOrder)
	})

	t.Run("a shop from before aisle orders has an empty one", func(t *testing.T) {
		s := setup(t)
		listID := groceries(t, s).ID
		aldi := addShop(t, s, listID, "Aldi")
		_, err := s.Exec("UPDATE shopping_shops SET category_order = '' WHERE id = ?", aldi.ID)
		require.NoError(t, err)
		_, err = s.Exec("INSERT INTO shopping_shops (list_id, title, position, created, updated) VALUES (?, 'Supabarn', 9, ?, ?)", listID, "2026-10-22 10:00:00", "2026-10-22 10:00:00")
		require.NoError(t, err)

		result, _, _, err := (&ShoppingShop{ListID: listID}).ReadAll(s, user1, "", 0, 0)
		require.NoError(t, err)
		shops := result.([]*ShoppingShop)
		require.Len(t, shops, 2)
		for _, shop := range shops {
			assert.Equal(t, []int64{}, shop.CategoryOrder, shop.Title)
		}
	})

	t.Run("an item's shop must be one of its list's shops", func(t *testing.T) {
		s := setup(t)
		kmart := &ShoppingList{Title: "Kmart"}
		require.NoError(t, kmart.Create(s, user1))
		other := addShop(t, s, kmart.ID, "Kmart Belconnen")

		item := &ShoppingItem{ListID: groceries(t, s).ID, Name: "Brie", ShopID: other.ID}
		require.ErrorIs(t, item.Create(s, user1), ErrShopDoesNotExist{ShopID: other.ID})
	})
}
