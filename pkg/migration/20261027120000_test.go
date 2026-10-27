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
	"code.vikunja.io/api/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func TestTaskRepeatRule20261027120000(t *testing.T) {
	// Fresh installs get the column from schemaBeans() (models.TaskRepeat).
	assert.Contains(t, schemaBeans(), any(&models.TaskRepeat{}))

	// An upgraded install: the table as 20261026120000 made it, with a row.
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.DropTables(taskRepeats20261026120000{}))
	t.Cleanup(func() { require.NoError(t, x.DropTables(taskRepeats20261026120000{})) })
	require.NoError(t, x.Sync(taskRepeats20261026120000{})) //nolint:forbidigo // the table before this migration
	_, err = x.Exec("INSERT INTO task_repeats (task_id, repeat_until, repeats_left) VALUES (7, NULL, 3)")
	require.NoError(t, err)

	before := taskRepeatsTable20261027120000(t, x)
	require.NoError(t, addTaskRepeatRule20261027120000(x))
	after := taskRepeatsTable20261027120000(t, x)

	require.NotNil(t, after.GetColumn("rule"))
	for _, column := range before.ColumnsSeq() {
		require.NotNilf(t, after.GetColumn(column), "migration dropped column %s", column)
	}
	for name := range before.Indexes {
		_, found := after.Indexes[name]
		require.Truef(t, found, "migration dropped index %s", name)
	}

	// The existing row keeps its end and gets an empty rule.
	got := []models.TaskRepeat{}
	require.NoError(t, x.Find(&got))
	require.Len(t, got, 1)
	assert.Equal(t, 3, got[0].RepeatsLeft)
	assert.Empty(t, got[0].Rule)
}

func taskRepeatsTable20261027120000(t *testing.T, x *xorm.Engine) *schemas.Table {
	t.Helper()
	tables, err := x.DBMetas()
	require.NoError(t, err)
	for _, table := range tables {
		if table.Name == "task_repeats" {
			return table
		}
	}
	t.Fatal("task_repeats table not found")
	return nil
}
