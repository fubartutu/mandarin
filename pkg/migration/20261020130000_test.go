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
	"xorm.io/xorm"
)

// indexNames lists a table's index names, so we can prove none were dropped.
func indexNames(t *testing.T, x *xorm.Engine, table string) []string {
	t.Helper()
	tables, err := x.DBMetas()
	require.NoError(t, err)
	for _, tbl := range tables {
		if tbl.Name == table {
			names := []string{}
			for name := range tbl.Indexes {
				names = append(names, name)
			}
			return names
		}
	}
	t.Fatalf("table %s not found", table)
	return nil
}

func TestAddShoppingCatalog20261020130000(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)

	// Start from the Phase 2 schema, as an upgraded install has it.
	phase2 := []any{shoppingLists20261020120000{}, shoppingCategories20261020120000{}, shoppingItems20261020120000{}}
	phase3 := []any{shoppingShops20261020130000{}, shoppingCatalogItems20261020130000{}, shoppingNotStocked20261020130000{}}
	require.NoError(t, x.DropTables(append(phase2, phase3...)...))
	require.NoError(t, x.Sync(phase2...)) //nolint:forbidigo // fresh tables for the test
	t.Cleanup(func() { require.NoError(t, x.DropTables(append(phase2, phase3...)...)) })

	_, err = x.Exec("INSERT INTO shopping_items (list_id, name, quantity, unit, category_id, done, position, created_by_id, created, updated) VALUES (1, 'Milk', 2, 'L', 5, ?, 0, 1, ?, ?)", false, "2026-10-20 10:00:00", "2026-10-20 10:00:00")
	require.NoError(t, err)
	indexesBefore := indexNames(t, x, "shopping_items")
	require.NotEmpty(t, indexesBefore)

	require.NoError(t, addShoppingCatalog20261020130000(x))

	has, err := columnExists(x, "shopping_items", "catalog_item_id")
	require.NoError(t, err)
	assert.True(t, has, "catalog_item_id column added")
	assert.Subset(t, indexNames(t, x, "shopping_items"), indexesBefore, "no existing index dropped")

	row := map[string]any{}
	has, err = x.Table("shopping_items").Where("name = ?", "Milk").Get(&row)
	require.NoError(t, err)
	require.True(t, has, "existing item kept")
	assert.EqualValues(t, 0, row["catalog_item_id"])

	for _, table := range []string{"shopping_shops", "shopping_catalog_items", "shopping_not_stocked"} {
		exists, err := x.IsTableExist(table)
		require.NoError(t, err)
		assert.True(t, exists, table)
	}

	// The unique name index stops two "Milk" entries.
	_, err = x.Exec("INSERT INTO shopping_catalog_items (name, name_key, created, updated) VALUES ('Milk', 'milk', ?, ?)", "2026-10-20 10:00:00", "2026-10-20 10:00:00")
	require.NoError(t, err)
	_, err = x.Exec("INSERT INTO shopping_catalog_items (name, name_key, created, updated) VALUES ('milk', 'milk', ?, ?)", "2026-10-20 10:00:00", "2026-10-20 10:00:00")
	require.Error(t, err)

	// Running it again (e.g. after a failed start) is harmless.
	require.NoError(t, addShoppingCatalog20261020130000(x))
}
