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

package migration

import (
	"testing"

	"code.vikunja.io/api/pkg/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddShoppingItemShop20261022150000(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)

	// Start from the Phase 8 schema, as an upgraded install has it.
	tables := []any{shoppingItems20261020120000{}, shoppingShops20261020130000{}}
	require.NoError(t, x.DropTables(tables...))
	require.NoError(t, x.Sync(tables...)) //nolint:forbidigo // fresh tables for the test
	require.NoError(t, partialSync(x, shoppingItems20261020130000{}, shoppingItems20261021120000{}))
	t.Cleanup(func() { require.NoError(t, x.DropTables(tables...)) })

	_, err = x.Exec("INSERT INTO shopping_items (list_id, name, quantity, unit, category_id, done, position, created_by_id, catalog_item_id, uid, created, updated) VALUES (1, 'Milk', 0, '', 0, ?, 0, 1, 0, 'a', ?, ?)", false, "2026-10-22 10:00:00", "2026-10-22 10:00:00")
	require.NoError(t, err)
	_, err = x.Exec("INSERT INTO shopping_shops (list_id, title, position, created, updated) VALUES (1, 'Aldi', 1, ?, ?)", "2026-10-22 10:00:00", "2026-10-22 10:00:00")
	require.NoError(t, err)
	indexesBefore := indexNames(t, x, "shopping_items")

	require.NoError(t, addShoppingItemShop20261022150000(x))

	assert.Subset(t, indexNames(t, x, "shopping_items"), indexesBefore, "no existing index dropped")
	rows := []map[string]any{}
	require.NoError(t, x.Table("shopping_items").Cols("shop_id").Find(&rows))
	require.Len(t, rows, 1)
	assert.EqualValues(t, 0, rows[0]["shop_id"], "existing items are for any shop")
	shops := []map[string]any{}
	require.NoError(t, x.Table("shopping_shops").Cols("category_order").Find(&shops))
	require.Len(t, shops, 1)
	assert.Empty(t, shops[0]["category_order"], "existing shops have no aisle order yet")

	// Running it again (e.g. after a failed start) is harmless.
	require.NoError(t, addShoppingItemShop20261022150000(x))
}
