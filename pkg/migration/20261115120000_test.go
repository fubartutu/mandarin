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
	"xorm.io/xorm/schemas"
)

func TestCatalogUsual20261115120000(t *testing.T) {
	// An upgraded install: the table as 20261020130000 made it, with a row.
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.DropTables(shoppingCatalogItems20261020130000{}))
	t.Cleanup(func() { require.NoError(t, x.DropTables(shoppingCatalogItems20261020130000{})) })
	require.NoError(t, x.Sync(shoppingCatalogItems20261020130000{})) //nolint:forbidigo // the table before this migration
	_, err = x.Exec("INSERT INTO shopping_catalog_items (name, name_key, quantity, unit, created, updated) VALUES ('Milk', 'milk', 2, 'L', ?, ?)", "2026-11-01 10:00:00", "2026-11-01 10:00:00")
	require.NoError(t, err)

	before := tableMeta20261115120000(t, x)
	require.NoError(t, addCatalogUsual20261115120000(x))
	after := tableMeta20261115120000(t, x)

	require.NotNil(t, after.GetColumn("usual"))
	for _, column := range before.ColumnsSeq() {
		require.NotNilf(t, after.GetColumn(column), "migration dropped column %s", column)
	}
	for name := range before.Indexes {
		_, found := after.Indexes[name]
		require.Truef(t, found, "migration dropped index %s", name)
	}

	// The existing entry keeps its data and isn't in the usual shop.
	type row struct {
		Name     string
		Quantity float64
		Usual    bool
	}
	got := []row{}
	require.NoError(t, x.Table("shopping_catalog_items").Find(&got))
	require.Len(t, got, 1)
	assert.Equal(t, "Milk", got[0].Name)
	assert.InDelta(t, 2.0, got[0].Quantity, 0)
	assert.False(t, got[0].Usual)
}

func tableMeta20261115120000(t *testing.T, x *xorm.Engine) *schemas.Table {
	t.Helper()
	tables, err := x.DBMetas()
	require.NoError(t, err)
	for _, table := range tables {
		if table.Name == "shopping_catalog_items" {
			return table
		}
	}
	t.Fatal("shopping_catalog_items table not found")
	return nil
}
