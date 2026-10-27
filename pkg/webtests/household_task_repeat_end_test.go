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

	"code.vikunja.io/api/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mandarin: /api/v2/tasks/{task}/repeat-end webtests.

func TestHouseholdTaskRepeatEnd(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	user1 := humaTokenFor(t, &testuser1)
	const path = "/api/v2/tasks/28/repeat-end"

	// A task without an end reads as an empty one.
	rec := humaRequest(t, e, http.MethodGet, path, "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	re := decode[models.TaskRepeatEnd](t, rec.Body.Bytes())
	assert.Nil(t, re.Until)
	assert.Zero(t, re.RepeatsLeft)

	rec = humaRequest(t, e, http.MethodPut, path,
		`{"until":"2026-11-05T23:59:59Z","repeats_left":5}`, user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = humaRequest(t, e, http.MethodGet, path, "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	re = decode[models.TaskRepeatEnd](t, rec.Body.Bytes())
	assert.Equal(t, int64(28), re.TaskID)
	assert.Equal(t, 5, re.RepeatsLeft)
	require.NotNil(t, re.Until)
	assert.Equal(t, 5, re.Until.Day())

	// PATCH comes for free and keeps what isn't sent.
	rec = humaRequest(t, e, http.MethodPatch, path, `{"repeats_left":0}`, user1, "application/merge-patch+json")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	re = decode[models.TaskRepeatEnd](t, rec.Body.Bytes())
	assert.Zero(t, re.RepeatsLeft)
	assert.NotNil(t, re.Until)

	t.Run("count out of range", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, path, `{"until":null,"repeats_left":5000}`, user1, "")
		assert.GreaterOrEqual(t, rec.Code, 400)
		assert.Less(t, rec.Code, 500)
	})
	t.Run("someone else's task", func(t *testing.T) {
		user2 := humaTokenFor(t, &testuser2)
		rec := humaRequest(t, e, http.MethodGet, path, "", user2, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		rec = humaRequest(t, e, http.MethodPut, path, `{"until":null,"repeats_left":2}`, user2, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})
}
