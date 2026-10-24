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
	"code.vikunja.io/api/pkg/household/menu"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/auth"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mandarin: /api/v2/menu webtests.

func setupMenuEnv(t *testing.T) *echo.Echo {
	t.Helper()
	e, err := setupTestEnv()
	require.NoError(t, err)
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.Sync2(menu.GetTables()...))
	_, err = x.Exec("DELETE FROM menu_entries")
	require.NoError(t, err)
	return e
}

type menuWeek struct {
	Items []menu.MenuEntry
}

func TestHouseholdMenu(t *testing.T) {
	e := setupMenuEnv(t)
	user1 := humaTokenFor(t, &testuser1)

	// A note on a dinner shows up when listing that week.
	rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu",
		`{"day":"2026-10-21","meal":"dinner","note":"Leftovers"}`, user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	leftovers := decode[menu.MenuEntry](t, rec.Body.Bytes())
	assert.NotZero(t, leftovers.ID)

	rec = humaRequest(t, e, http.MethodGet, "/api/v2/menu?from=2026-10-19&to=2026-10-25", "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	week := decode[menuWeek](t, rec.Body.Bytes()).Items
	require.Len(t, week, 1)
	assert.Equal(t, "2026-10-21", week[0].Day)
	assert.Equal(t, "dinner", week[0].Meal)
	assert.Equal(t, "Leftovers", week[0].Note)
}

func TestHouseholdMenu_WeekOrder(t *testing.T) {
	e := setupMenuEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	add := func(day, meal, note string) menu.MenuEntry {
		t.Helper()
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu",
			`{"day":"`+day+`","meal":"`+meal+`","note":"`+note+`"}`, user1, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		return decode[menu.MenuEntry](t, rec.Body.Bytes())
	}

	add("2026-10-25", "dinner", "Roast")       // Sunday
	add("2026-10-20", "dinner", "Curry")       // Tuesday, main
	add("2026-10-20", "dinner", "Naan")        // Tuesday, side: after the main
	add("2026-10-20", "breakfast", "Porridge") // breakfast comes before dinner
	add("2026-10-18", "dinner", "Last week")   // the Sunday before
	add("2026-10-26", "dinner", "Next week")   // the Monday after
	add("2026-10-19", "lunch", "Sandwiches")   // Monday

	rec := humaRequest(t, e, http.MethodGet, "/api/v2/menu?from=2026-10-19&to=2026-10-25", "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	notes := []string{}
	for _, entry := range decode[menuWeek](t, rec.Body.Bytes()).Items {
		notes = append(notes, entry.Note)
	}
	assert.Equal(t, []string{"Sandwiches", "Porridge", "Curry", "Naan", "Roast"}, notes)
}

func TestHouseholdMenu_SharedAndDeleted(t *testing.T) {
	e := setupMenuEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	user2 := humaTokenFor(t, &testuser2)

	rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu", `{"day":"2026-10-23","meal":"dinner","note":"Takeaway"}`, user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	takeaway := decode[menu.MenuEntry](t, rec.Body.Bytes())

	// The other member sees the same plan and can delete user1's entry.
	rec = humaRequest(t, e, http.MethodGet, "/api/v2/menu?from=2026-10-19&to=2026-10-25", "", user2, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, decode[menuWeek](t, rec.Body.Bytes()).Items, 1)

	rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/menu/%d", takeaway.ID), "", user2, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	rec = humaRequest(t, e, http.MethodGet, "/api/v2/menu?from=2026-10-19&to=2026-10-25", "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Empty(t, decode[menuWeek](t, rec.Body.Bytes()).Items)

	rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/menu/%d", takeaway.ID), "", user1, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

func TestHouseholdMenu_Refusals(t *testing.T) {
	e := setupMenuEnv(t)
	user1 := humaTokenFor(t, &testuser1)

	rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu", `{"day":"2026-10-23","meal":"dinner","note":"Pizza"}`, user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	pizza := decode[menu.MenuEntry](t, rec.Body.Bytes())

	t.Run("every meal is accepted", func(t *testing.T) {
		for _, meal := range []string{"breakfast", "lunch", "dinner"} {
			rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu", `{"day":"2026-10-24","meal":"`+meal+`","note":"Eggs"}`, user1, "")
			assert.Equal(t, http.StatusCreated, rec.Code, "%s: %s", meal, rec.Body.String())
		}
	})

	t.Run("bad entries", func(t *testing.T) {
		for _, body := range []string{
			`{"day":"2026-10-23","meal":"supper","note":"Soup"}`,
			`{"day":"2026-02-30","meal":"dinner","note":"Soup"}`,
			`{"day":"23/10/2026","meal":"dinner","note":"Soup"}`,
			`{"day":"2026-10-23","meal":"dinner","note":"   "}`,
		} {
			rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu", body, user1, "")
			assert.True(t, rec.Code == http.StatusBadRequest || rec.Code == http.StatusUnprocessableEntity, "%s: %d %s", body, rec.Code, rec.Body.String())
		}
	})

	t.Run("bad ranges", func(t *testing.T) {
		for _, query := range []string{
			"",                               // no range
			"?from=2026-10-25&to=2026-10-19", // backwards
			"?from=2026-01-01&to=2026-12-31", // too long
		} {
			rec := humaRequest(t, e, http.MethodGet, "/api/v2/menu"+query, "", user1, "")
			assert.True(t, rec.Code == http.StatusBadRequest || rec.Code == http.StatusUnprocessableEntity, "%q: %d %s", query, rec.Code, rec.Body.String())
		}
	})

	t.Run("not logged in", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodGet, "/api/v2/menu?from=2026-10-19&to=2026-10-25", "", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	})

	t.Run("link shares are not household members", func(t *testing.T) {
		token, err := auth.NewLinkShareJWTAuthtoken(&models.LinkSharing{
			ID:          1,
			Hash:        "test",
			ProjectID:   1,
			Permission:  models.PermissionAdmin,
			SharingType: models.SharingTypeWithoutPassword,
			SharedByID:  1,
		})
		require.NoError(t, err)
		for _, req := range []struct{ method, path, body string }{
			{http.MethodGet, "/api/v2/menu?from=2026-10-19&to=2026-10-25", ""},
			{http.MethodPost, "/api/v2/menu", `{"day":"2026-10-23","meal":"dinner","note":"Chips"}`},
			{http.MethodDelete, fmt.Sprintf("/api/v2/menu/%d", pizza.ID), ""},
		} {
			rec := humaRequest(t, e, req.method, req.path, req.body, token, "")
			assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s: %s", req.method, req.path, rec.Body.String())
		}
		db.AssertExists(t, "menu_entries", map[string]any{"id": pizza.ID}, false)
	})
}
