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
	"code.vikunja.io/api/pkg/household/menu"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMenuEntries20261024120000(t *testing.T) {
	// Fresh installs skip migrations and create schemaBeans() instead.
	assert.Contains(t, schemaBeans(), any(&menu.MenuEntry{}), "menu_entries is created on fresh installs")

	// An upgraded install gets the same columns from the migration.
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.DropTables(menuEntries20261024120000{}))
	t.Cleanup(func() { require.NoError(t, x.DropTables(menuEntries20261024120000{})) })
	for _, m := range migrations {
		if m.ID == "20261024120000" {
			require.NoError(t, m.Migrate(x))
		}
	}
	_, err = x.Exec("INSERT INTO menu_entries (day, meal, position, note, created_by_id, created, updated) VALUES ('2026-10-24', 'dinner', 0, 'Leftovers', 1, ?, ?)",
		"2026-10-24 10:00:00", "2026-10-24 10:00:00")
	require.NoError(t, err)
	entries := []menu.MenuEntry{}
	require.NoError(t, x.Find(&entries))
	require.Len(t, entries, 1)
	assert.Equal(t, "Leftovers", entries[0].Note)
}
