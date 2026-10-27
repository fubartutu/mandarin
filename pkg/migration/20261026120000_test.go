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
)

func TestTaskRepeats20261026120000(t *testing.T) {
	// Fresh installs skip migrations and create schemaBeans() instead.
	assert.Contains(t, schemaBeans(), any(&models.TaskRepeat{}), "task_repeats is created on fresh installs")

	// An upgraded install gets the same columns from the migration.
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.DropTables(taskRepeats20261026120000{}))
	t.Cleanup(func() { require.NoError(t, x.DropTables(taskRepeats20261026120000{})) })
	for _, m := range migrations {
		if m.ID == "20261026120000" {
			require.NoError(t, m.Migrate(x))
		}
	}
	_, err = x.Exec("INSERT INTO task_repeats (task_id, repeat_until, repeats_left) VALUES (1, ?, 3)", "2026-11-05 23:59:59")
	require.NoError(t, err)
	ends := []taskRepeats20261026120000{} // as this migration made it, without later columns
	require.NoError(t, x.Find(&ends))
	require.Len(t, ends, 1)
	assert.Equal(t, 3, ends[0].RepeatsLeft)
	require.NotNil(t, ends[0].Until)
	assert.Equal(t, 5, ends[0].Until.Day())
}
