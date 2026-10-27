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

package webtests

import (
	"net/http"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mandarin: /api/v2/tasks/{task}/repeat webtests.

// setupRepeatEnv is setupTestEnv with task_repeats emptied: it has no
// fixture file, so rows from an earlier test would otherwise stay.
func setupRepeatEnv(t *testing.T) *echo.Echo {
	t.Helper()
	e, err := setupTestEnv()
	require.NoError(t, err)
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	_, err = x.Exec("DELETE FROM task_repeats")
	require.NoError(t, err)
	return e
}

func TestHouseholdTaskRepeat(t *testing.T) {
	e := setupRepeatEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	const path = "/api/v2/tasks/28/repeat"

	// A task without an end reads as an empty one.
	rec := humaRequest(t, e, http.MethodGet, path, "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	re := decode[models.TaskRepeat](t, rec.Body.Bytes())
	assert.Nil(t, re.Until)
	assert.Zero(t, re.RepeatsLeft)

	rec = humaRequest(t, e, http.MethodPut, path,
		`{"rule":"","until":"2026-11-05T23:59:59Z","repeats_left":5}`, user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = humaRequest(t, e, http.MethodGet, path, "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	re = decode[models.TaskRepeat](t, rec.Body.Bytes())
	assert.Equal(t, int64(28), re.TaskID)
	assert.Equal(t, 5, re.RepeatsLeft)
	require.NotNil(t, re.Until)
	assert.Equal(t, 5, re.Until.Day())

	// PATCH comes for free and keeps what isn't sent.
	rec = humaRequest(t, e, http.MethodPatch, path, `{"repeats_left":0}`, user1, "application/merge-patch+json")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	re = decode[models.TaskRepeat](t, rec.Body.Bytes())
	assert.Zero(t, re.RepeatsLeft)
	assert.NotNil(t, re.Until)

	t.Run("a rule", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPatch, path, `{"rule":"FREQ=WEEKLY;BYDAY=MO,TH"}`, user1, "application/merge-patch+json")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		re := decode[models.TaskRepeat](t, rec.Body.Bytes())
		assert.Equal(t, "FREQ=WEEKLY;BYDAY=MO,TH", re.Rule)
		assert.NotNil(t, re.Until, "PATCH keeps the end")

		rec = humaRequest(t, e, http.MethodPatch, path, `{"rule":"FREQ=DAILY"}`, user1, "application/merge-patch+json")
		assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})
	t.Run("PUT needs the rule, so a client can't drop it by accident", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, path, `{"until":null,"repeats_left":2}`, user1, "")
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	})
	t.Run("count out of range", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, path, `{"rule":"","until":null,"repeats_left":5000}`, user1, "")
		assert.GreaterOrEqual(t, rec.Code, 400)
		assert.Less(t, rec.Code, 500)
	})
	t.Run("someone else's task", func(t *testing.T) {
		user2 := humaTokenFor(t, &testuser2)
		rec := humaRequest(t, e, http.MethodGet, path, "", user2, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		rec = humaRequest(t, e, http.MethodPut, path, `{"rule":"","until":null,"repeats_left":2}`, user2, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})
}

func TestHouseholdTaskSkip(t *testing.T) {
	e := setupRepeatEnv(t)
	user1 := humaTokenFor(t, &testuser1)

	// Fixture task 28 repeats hourly; next is filled in on read.
	rec := humaRequest(t, e, http.MethodGet, "/api/v2/tasks/28/repeat", "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	re := decode[models.TaskRepeat](t, rec.Body.Bytes())
	require.NotNil(t, re.Next)

	rec = humaRequest(t, e, http.MethodPost, "/api/v2/tasks/28/skip", "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	task := decode[models.Task](t, rec.Body.Bytes())
	assert.False(t, task.Done)
	assert.True(t, task.DueDate.Equal(*re.Next), "lands on the next it announced: %s vs %s", task.DueDate, re.Next)

	t.Run("a task that doesn't repeat", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/1/skip", "", user1, "")
		assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})
	t.Run("someone else's task", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/28/skip", "", humaTokenFor(t, &testuser2), "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})
}
