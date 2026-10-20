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

func TestShoppingCatalog_Seed(t *testing.T) {
	s := setup(t)

	count, err := s.Count(&ShoppingCatalogItem{})
	require.NoError(t, err)
	expected := 0
	for _, names := range DefaultCatalog {
		expected += len(names)
	}
	assert.Equal(t, int64(expected), count)

	vegemite := catalogEntry(t, s, "vegemite")
	pantry := &ShoppingCategory{}
	has, err := s.Where("title = ?", "Pantry").Get(pantry)
	require.NoError(t, err)
	require.True(t, has)
	assert.Equal(t, pantry.ID, vegemite.CategoryID, "pre-loaded items sort themselves")

	require.NoError(t, Seed(s))
	again, err := s.Count(&ShoppingCatalogItem{})
	require.NoError(t, err)
	assert.Equal(t, count, again, "seeding twice adds nothing")
}

func TestShoppingCatalog_UseFromItems(t *testing.T) {
	t.Run("a known item takes its category from the catalogue", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "vegemite"})
		vegemite := catalogEntry(t, s, "Vegemite")
		assert.Equal(t, vegemite.ID, item.CatalogItemID)
		assert.Equal(t, vegemite.CategoryID, item.CategoryID)
		assert.Equal(t, int64(1), vegemite.TimesAdded)
		assert.NotNil(t, vegemite.LastAdded)
	})

	t.Run("a new name is saved to the catalogue", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: " Dragon fruit ", CategoryID: 1})
		entry := catalogEntry(t, s, "dragon fruit")
		assert.Equal(t, "Dragon fruit", entry.Name, "stored trimmed")
		assert.Equal(t, int64(1), entry.CategoryID)
		assert.Equal(t, entry.ID, item.CatalogItemID)
	})

	t.Run("re-adding keeps the last quantity, unit and category", func(t *testing.T) {
		s := setup(t)
		listID := groceries(t, s).ID
		first := addItem(t, s, &ShoppingItem{ListID: listID, Name: "Mince", Quantity: 500, Unit: "g", CategoryID: 3})
		_, err := (&ShoppingList{ID: listID}).ClearDoneItems(s)
		require.NoError(t, err)
		require.NoError(t, (&ShoppingItem{ListID: listID, ID: first.ID}).Delete(s, user1))

		again := addItem(t, s, &ShoppingItem{ListID: listID, Name: "mince"})
		assert.InDelta(t, 500, again.Quantity, 0)
		assert.Equal(t, "g", again.Unit)
		assert.Equal(t, int64(3), again.CategoryID)
		assert.Equal(t, int64(2), catalogEntry(t, s, "mince").TimesAdded)
	})

	t.Run("an explicit quantity or category wins and is remembered", func(t *testing.T) {
		s := setup(t)
		listID := groceries(t, s).ID
		addItem(t, s, &ShoppingItem{ListID: listID, Name: "Milk", Quantity: 2, Unit: "L"})
		item := addItem(t, s, &ShoppingItem{ListID: listID, Name: "Milk", Quantity: 3, Unit: "L", CategoryID: 6})
		assert.InDelta(t, 3, item.Quantity, 0)
		entry := catalogEntry(t, s, "milk")
		assert.InDelta(t, 3, entry.Quantity, 0)
		assert.Equal(t, int64(6), entry.CategoryID)
	})

	t.Run("editing doesn't refill a cleared quantity, but is remembered", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Milk", Quantity: 2, Unit: "L"})
		item.Quantity, item.Unit = 0, ""
		require.NoError(t, item.Update(s, user1))
		assert.Zero(t, item.Quantity)
		entry := catalogEntry(t, s, "milk")
		assert.Zero(t, entry.Quantity)
		assert.Equal(t, int64(1), entry.TimesAdded, "edits don't count as a use")
	})

	t.Run("ticking an item doesn't write the catalogue", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Milk", Quantity: 2, Unit: "L"})
		_, err := s.Exec("UPDATE shopping_catalog_items SET updated = ? WHERE name_key = 'milk'", "2020-01-01 00:00:00")
		require.NoError(t, err)

		item.Done = true
		require.NoError(t, item.Update(s, user1))
		assert.Equal(t, 2020, catalogEntry(t, s, "milk").Updated.Year(), "catalogue row untouched")
	})

	t.Run("renaming an item relinks it", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Milk"})
		item.Name = "Oat milk"
		require.NoError(t, item.Update(s, user1))
		assert.Equal(t, catalogEntry(t, s, "oat milk").ID, item.CatalogItemID)
	})

	t.Run("items carry their entry's not-stocked shops and photo flag", func(t *testing.T) {
		s := setup(t)
		listID := groceries(t, s).ID
		aldi := addShop(t, s, listID, "Aldi")
		require.NoError(t, setNotStocked(s, catalogEntry(t, s, "Vegemite").ID, []int64{aldi.ID}))
		addItem(t, s, &ShoppingItem{ListID: listID, Name: "Vegemite"})
		addItem(t, s, &ShoppingItem{ListID: listID, Name: "Milk"})

		result, _, _, err := (&ShoppingItem{ListID: listID}).ReadAll(s, user1, "", 0, 0)
		require.NoError(t, err)
		byName := map[string]*ShoppingItem{}
		for _, i := range result.([]*ShoppingItem) {
			byName[i.Name] = i
		}
		assert.Equal(t, []int64{aldi.ID}, byName["Vegemite"].NotStockedShopIDs)
		assert.Equal(t, []int64{}, byName["Milk"].NotStockedShopIDs)
		assert.False(t, byName["Milk"].HasPhoto)
	})

	t.Run("items from before the catalogue are linked at seed time", func(t *testing.T) {
		s := setup(t)
		_, err := s.Insert(&ShoppingItem{ListID: groceries(t, s).ID, Name: "Kumquats"})
		require.NoError(t, err)
		require.NoError(t, Seed(s))

		item := &ShoppingItem{}
		has, err := s.Where("name = ?", "Kumquats").Get(item)
		require.NoError(t, err)
		require.True(t, has)
		assert.Equal(t, catalogEntry(t, s, "kumquats").ID, item.CatalogItemID)
		assert.Zero(t, catalogEntry(t, s, "kumquats").TimesAdded, "linking isn't a use")
	})
}

func TestShoppingCatalog_CRUD(t *testing.T) {
	t.Run("read all ranks frequent, then recent, then the rest", func(t *testing.T) {
		s := setup(t)
		listID := groceries(t, s).ID
		addItem(t, s, &ShoppingItem{ListID: listID, Name: "Bread"})
		addItem(t, s, &ShoppingItem{ListID: listID, Name: "Milk"})
		addItem(t, s, &ShoppingItem{ListID: listID, Name: "Milk"})

		result, _, _, err := (&ShoppingCatalogItem{}).ReadAll(s, user1, "", 1, 3)
		require.NoError(t, err)
		entries := result.([]*ShoppingCatalogItem)
		require.Len(t, entries, 3)
		assert.Equal(t, "Milk", entries[0].Name)
		assert.Equal(t, "Bread", entries[1].Name)
		assert.Zero(t, entries[2].TimesAdded)
		assert.NotNil(t, entries[2].NotStockedShopIDs)
	})

	t.Run("search filters by name", func(t *testing.T) {
		s := setup(t)
		result, _, total, err := (&ShoppingCatalogItem{}).ReadAll(s, user1, "TIM", 0, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Tim Tams", result.([]*ShoppingCatalogItem)[0].Name)
	})

	t.Run("names are unique ignoring case", func(t *testing.T) {
		s := setup(t)
		err := (&ShoppingCatalogItem{Name: "MILK"}).Create(s, user1)
		require.ErrorIs(t, err, ErrCatalogItemExists{Name: "Milk"})

		bread := catalogEntry(t, s, "bread")
		bread.Name = "milk"
		require.ErrorIs(t, bread.Update(s, user1), ErrCatalogItemExists{Name: "Milk"})
	})

	t.Run("update sets not-stocked shops, which must exist", func(t *testing.T) {
		s := setup(t)
		listID := groceries(t, s).ID
		aldi := addShop(t, s, listID, "Aldi")
		supabarn := addShop(t, s, listID, "Supabarn")
		entry := catalogEntry(t, s, "Vegemite")

		entry.NotStockedShopIDs = []int64{supabarn.ID, aldi.ID, aldi.ID}
		require.NoError(t, entry.Update(s, user1))
		assert.Equal(t, []int64{aldi.ID, supabarn.ID}, entry.NotStockedShopIDs)

		entry.NotStockedShopIDs = nil // field not sent: left alone
		require.NoError(t, entry.Update(s, user1))
		assert.Len(t, entry.NotStockedShopIDs, 2)

		entry.NotStockedShopIDs = []int64{}
		require.NoError(t, entry.Update(s, user1))
		assert.Empty(t, entry.NotStockedShopIDs)

		entry.NotStockedShopIDs = []int64{9999}
		require.ErrorIs(t, entry.Update(s, user1), ErrShopDoesNotExist{})
	})

	t.Run("deleting an entry unlinks its items", func(t *testing.T) {
		s := setup(t)
		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Vegemite"})
		require.NoError(t, (&ShoppingCatalogItem{ID: item.CatalogItemID}).Delete(s, user1))

		stored, err := getItem(s, item.ListID, item.ID)
		require.NoError(t, err)
		assert.Zero(t, stored.CatalogItemID)
		assert.Equal(t, "Vegemite", stored.Name)
	})

	t.Run("deleting a category uncategorises catalogue entries", func(t *testing.T) {
		s := setup(t)
		categoryID := catalogEntry(t, s, "Vegemite").CategoryID
		require.NoError(t, (&ShoppingCategory{ID: categoryID}).Delete(s, user1))
		assert.Zero(t, catalogEntry(t, s, "Vegemite").CategoryID)
	})

	t.Run("permissions", func(t *testing.T) {
		s := setup(t)
		id := catalogEntry(t, s, "Milk").ID

		can, _, err := (&ShoppingCatalogItem{ID: id}).CanRead(s, user2)
		require.NoError(t, err)
		assert.True(t, can)
		can, err = (&ShoppingCatalogItem{ID: id}).CanUpdate(s, user2)
		require.NoError(t, err)
		assert.True(t, can)

		_, err = (&ShoppingCatalogItem{}).CanCreate(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, _, err = (&ShoppingCatalogItem{ID: id}).CanRead(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, err = (&ShoppingCatalogItem{ID: id}).CanDelete(s, linkShare)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})
		_, _, _, err = (&ShoppingCatalogItem{}).ReadAll(s, linkShare, "", 0, 0)
		require.ErrorIs(t, err, models.ErrGenericForbidden{})

		_, err = (&ShoppingCatalogItem{ID: 999999}).CanUpdate(s, user1)
		require.ErrorIs(t, err, ErrCatalogItemDoesNotExist{CatalogItemID: 999999})
	})
}
