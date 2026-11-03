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

func TestHabitReminders20261105120000(t *testing.T) {
	// An upgraded install: the table as 20261022120000 made it, with a row.
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.DropTables(pushPreferences20261022120000{}))
	t.Cleanup(func() { require.NoError(t, x.DropTables(pushPreferences20261022120000{})) })
	require.NoError(t, x.Sync(pushPreferences20261022120000{})) //nolint:forbidigo // the table before this migration
	_, err = x.Exec("INSERT INTO push_preferences (user_id, reminders, overdue, quiet_hours) VALUES (7, ?, ?, ?)", false, true, true)
	require.NoError(t, err)

	before := pushPreferencesTable20261105120000(t, x)
	require.NoError(t, addHabitReminders20261105120000(x))
	after := pushPreferencesTable20261105120000(t, x)

	require.NotNil(t, after.GetColumn("habit_reminders"))
	for _, column := range before.ColumnsSeq() {
		require.NotNilf(t, after.GetColumn(column), "migration dropped column %s", column)
	}
	for name := range before.Indexes {
		_, found := after.Indexes[name]
		require.Truef(t, found, "migration dropped index %s", name)
	}

	// The existing row keeps its choices and gets habit reminders on.
	type row struct {
		UserID         int64
		Reminders      bool
		HabitReminders bool
	}
	got := []row{}
	require.NoError(t, x.Table("push_preferences").Find(&got))
	require.Len(t, got, 1)
	assert.False(t, got[0].Reminders)
	assert.True(t, got[0].HabitReminders)
}

func pushPreferencesTable20261105120000(t *testing.T, x *xorm.Engine) *schemas.Table {
	t.Helper()
	tables, err := x.DBMetas()
	require.NoError(t, err)
	for _, table := range tables {
		if table.Name == "push_preferences" {
			return table
		}
	}
	t.Fatal("push_preferences table not found")
	return nil
}
