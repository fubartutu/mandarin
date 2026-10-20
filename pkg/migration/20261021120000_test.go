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

func TestAddShoppingSync20261021120000(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)

	// Start from the Phase 3 schema, as an upgraded install has it.
	before := []any{shoppingItems20261020120000{}}
	added := []any{shoppingChanges20261021120000{}, shoppingConflicts20261021120000{}}
	require.NoError(t, x.DropTables(append(before, added...)...))
	require.NoError(t, x.Sync(before...)) //nolint:forbidigo // fresh tables for the test
	require.NoError(t, partialSync(x, shoppingItems20261020130000{}))
	t.Cleanup(func() { require.NoError(t, x.DropTables(append(before, added...)...)) })

	for _, name := range []string{"Milk", "Bread"} {
		_, err = x.Exec("INSERT INTO shopping_items (list_id, name, quantity, unit, category_id, done, position, created_by_id, catalog_item_id, created, updated) VALUES (1, ?, 0, '', 0, ?, 0, 1, 0, ?, ?)", name, false, "2026-10-20 10:00:00", "2026-10-20 10:00:00")
		require.NoError(t, err)
	}
	indexesBefore := indexNames(t, x, "shopping_items")

	require.NoError(t, addShoppingSync20261021120000(x))

	assert.Subset(t, indexNames(t, x, "shopping_items"), indexesBefore, "no existing index dropped")
	// xorm reports UQE_shopping_items_uid by its short name.
	assert.Contains(t, indexNames(t, x, "shopping_items"), "uid")

	rows := []map[string]any{}
	require.NoError(t, x.Table("shopping_items").Cols("uid").Find(&rows))
	require.Len(t, rows, 2)
	assert.Len(t, rows[0]["uid"], 36, "existing items got a uid")
	assert.NotEqual(t, rows[0]["uid"], rows[1]["uid"])

	// The index is unique.
	_, err = x.Exec("UPDATE shopping_items SET uid = ? WHERE name = 'Bread'", rows[0]["uid"])
	require.Error(t, err)

	for _, table := range []string{"shopping_changes", "shopping_conflicts"} {
		exists, err := x.IsTableExist(table)
		require.NoError(t, err)
		assert.True(t, exists, table)
	}
	// One change-log row per entity.
	assert.Contains(t, indexNames(t, x, "shopping_changes"), "entity_row")

	// Running it again (e.g. after a failed start) is harmless.
	require.NoError(t, addShoppingSync20261021120000(x))
}
