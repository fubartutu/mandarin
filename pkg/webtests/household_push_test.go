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
	"fmt"
	"net/http"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household/push"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mandarin: /api/v2/push webtests (sending itself is tested in pkg/household/push).

func TestHouseholdPush(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.Sync2(push.GetTables()...))
	token := humaTokenFor(t, &testuser1)

	rec := humaRequest(t, e, http.MethodGet, "/api/v2/push/key", "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotEmpty(t, decode[struct {
		PublicKey string `json:"public_key"`
	}](t, rec.Body.Bytes()).PublicKey)

	rec = humaRequest(t, e, http.MethodPost, "/api/v2/push/devices",
		`{"endpoint":"https://web.push.apple.com/abc","keys":{"p256dh":"BNNL5ZaTfK81qhXOx23-wewhigUeFb632jN6LvRWCFH1ubQr77FE_9qV1FuojuRmHP42zmf34rXgW80OvUVDgTk","auth":"zqbxT6JKstKSY9JKibZLSQ"},"name":"iPhone"}`, token, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	device := decode[push.Device](t, rec.Body.Bytes())

	rec = humaRequest(t, e, http.MethodGet, "/api/v2/push/devices", "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Len(t, decode[struct{ Devices []push.Device }](t, rec.Body.Bytes()).Devices, 1)

	rec = humaRequest(t, e, http.MethodGet, "/api/v2/push/settings", "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"reminders":true`)
	rec = humaRequest(t, e, http.MethodPut, "/api/v2/push/settings", `{"reminders":true,"overdue":false,"quiet_hours":true,"shopping_list_ids":[1]}`, token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"shopping_list_ids":[1]`)

	rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/push/devices/%d", device.ID), "", token, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
}
