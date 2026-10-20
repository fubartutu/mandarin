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

func TestShoppingCategory(t *testing.T) {
	t.Run("seeded in order, and seeding twice adds nothing", func(t *testing.T) {
		s := setup(t)
		require.NoError(t, Seed(s))

		result, _, total, err := (&ShoppingCategory{}).ReadAll(s, user1, "", 0, 0)
		require.NoError(t, err)
		categories := result.([]*ShoppingCategory)
		assert.Equal(t, int64(len(DefaultCategories)), total)
		assert.Equal(t, "Fruit & Veg", categories[0].Title)
		assert.Equal(t, "Other", categories[len(categories)-1].Title)

		lists, err := s.Count(&ShoppingList{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), lists)
	})

	t.Run("members can manage categories, link shares can't", func(t *testing.T) {
		s := setup(t)
		c := &ShoppingCategory{Title: "Garden"}
		can, err := c.CanCreate(s, user2)
		require.NoError(t, err)
		assert.True(t, can)
		require.NoError(t, c.Create(s, user2))
		assert.InDelta(t, float64(len(DefaultCategories)+1), c.Position, 0, "new categories go last")

		can, err = (&ShoppingCategory{ID: c.ID}).CanUpdate(s, user1)
		require.NoError(t, err)
		assert.True(t, can)

		can, err = c.CanCreate(s, linkShare)
		require.NoError(t, err)
		assert.False(t, can)
		_, _, err = (&ShoppingCategory{ID: c.ID}).CanRead(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, err = (&ShoppingCategory{ID: c.ID}).CanDelete(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, _, _, err = (&ShoppingCategory{}).ReadAll(s, linkShare, "", 0, 0)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})

		_, err = (&ShoppingCategory{ID: 9999}).CanDelete(s, user1)
		require.ErrorIs(t, err, ErrCategoryDoesNotExist{CategoryID: 9999})
	})

	t.Run("deleting a category uncategorises its items", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Milk", CategoryID: 5})

		require.NoError(t, (&ShoppingCategory{ID: 5}).Delete(s, user1))

		stored, err := getItem(s, item.ListID, item.ID)
		require.NoError(t, err)
		assert.Zero(t, stored.CategoryID)
	})
}
