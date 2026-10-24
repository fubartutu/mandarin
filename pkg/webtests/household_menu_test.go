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
	"code.vikunja.io/api/pkg/household/recipes"
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
	require.NoError(t, x.Sync2(recipes.GetTables()...))
	_, err = x.Exec("DELETE FROM menu_entries")
	require.NoError(t, err)
	_, err = x.Exec("DELETE FROM recipes")
	require.NoError(t, err)
	return e
}

// addMenuRecipe saves a recipe to plan from.
func addMenuRecipe(t *testing.T, e *echo.Echo, token, title string, servings float64) recipes.Recipe {
	t.Helper()
	rec := humaRequest(t, e, http.MethodPost, "/api/v2/recipes",
		fmt.Sprintf(`{"title":%q,"servings":%v}`, title, servings), token, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return decode[recipes.Recipe](t, rec.Body.Bytes())
}

// menuLinkShareToken is a link share's token: link shares aren't household members.
func menuLinkShareToken(t *testing.T) string {
	t.Helper()
	token, err := auth.NewLinkShareJWTAuthtoken(&models.LinkSharing{
		ID:          1,
		Hash:        "test",
		ProjectID:   1,
		Permission:  models.PermissionAdmin,
		SharingType: models.SharingTypeWithoutPassword,
		SharedByID:  1,
	})
	require.NoError(t, err)
	return token
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
		token := menuLinkShareToken(t)
		for _, req := range []struct{ method, path, body string }{
			{http.MethodGet, "/api/v2/menu?from=2026-10-19&to=2026-10-25", ""},
			{http.MethodPost, "/api/v2/menu", `{"day":"2026-10-23","meal":"dinner","note":"Chips"}`},
			{http.MethodPut, fmt.Sprintf("/api/v2/menu/%d", pizza.ID), `{"note":"Chips"}`},
			{http.MethodDelete, fmt.Sprintf("/api/v2/menu/%d", pizza.ID), ""},
		} {
			rec := humaRequest(t, e, req.method, req.path, req.body, token, "")
			assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s: %s", req.method, req.path, rec.Body.String())
		}
		db.AssertExists(t, "menu_entries", map[string]any{"id": pizza.ID, "note": "Pizza"}, false)
	})
}

func TestHouseholdMenu_RecipeEntries(t *testing.T) {
	e := setupMenuEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	lasagne := addMenuRecipe(t, e, user1, "Lasagne", 4)

	// Without servings, the entry takes the recipe's.
	rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu",
		fmt.Sprintf(`{"day":"2026-10-24","meal":"dinner","recipe_id":%d}`, lasagne.ID), user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	planned := decode[menu.MenuEntry](t, rec.Body.Bytes())
	assert.Equal(t, lasagne.ID, planned.RecipeID)
	assert.InDelta(t, 4.0, planned.Servings, 0.001)
	assert.Equal(t, "Lasagne", planned.RecipeTitle)
	assert.Empty(t, planned.Note)

	// Given servings win ("for 6 on Saturday"); a note sits on the same day after it.
	rec = humaRequest(t, e, http.MethodPost, "/api/v2/menu",
		fmt.Sprintf(`{"day":"2026-10-24","meal":"dinner","recipe_id":%d,"servings":6}`, lasagne.ID), user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.InDelta(t, 6.0, decode[menu.MenuEntry](t, rec.Body.Bytes()).Servings, 0.001)
	rec = humaRequest(t, e, http.MethodPost, "/api/v2/menu", `{"day":"2026-10-24","meal":"dinner","note":"Salad"}`, user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	rec = humaRequest(t, e, http.MethodGet, "/api/v2/menu?from=2026-10-19&to=2026-10-25", "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	week := decode[menuWeek](t, rec.Body.Bytes()).Items
	require.Len(t, week, 3)
	assert.Equal(t, "Lasagne", week[0].RecipeTitle)
	assert.InDelta(t, 4.0, week[0].Servings, 0.001)
	assert.Equal(t, "Lasagne", week[1].RecipeTitle)
	assert.InDelta(t, 6.0, week[1].Servings, 0.001)
	assert.Equal(t, "Salad", week[2].Note)
	assert.Empty(t, week[2].RecipeTitle)
}

func TestHouseholdMenu_RecipeEntryRefusals(t *testing.T) {
	e := setupMenuEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	lasagne := addMenuRecipe(t, e, user1, "Lasagne", 4)

	for _, body := range []string{
		`{"day":"2026-10-24","meal":"dinner","recipe_id":999999}`,                                    // unknown recipe
		fmt.Sprintf(`{"day":"2026-10-24","meal":"dinner","recipe_id":%d,"note":"Pie"}`, lasagne.ID),  // both
		`{"day":"2026-10-24","meal":"dinner"}`,                                                       // neither
		fmt.Sprintf(`{"day":"2026-10-24","meal":"dinner","recipe_id":%d,"servings":-1}`, lasagne.ID), // negative
		`{"day":"2026-10-24","meal":"dinner","note":"Pie","servings":2}`,                             // a note has no servings
	} {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu", body, user1, "")
		assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound, http.StatusUnprocessableEntity}, rec.Code, "%s: %s", body, rec.Body.String())
	}
	rec := humaRequest(t, e, http.MethodGet, "/api/v2/menu?from=2026-10-19&to=2026-10-25", "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Empty(t, decode[menuWeek](t, rec.Body.Bytes()).Items)
}

func TestHouseholdMenu_Update(t *testing.T) {
	e := setupMenuEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	user2 := humaTokenFor(t, &testuser2)
	lasagne := addMenuRecipe(t, e, user1, "Lasagne", 4)

	rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu",
		fmt.Sprintf(`{"day":"2026-10-24","meal":"dinner","recipe_id":%d}`, lasagne.ID), user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	planned := decode[menu.MenuEntry](t, rec.Body.Bytes())
	rec = humaRequest(t, e, http.MethodPost, "/api/v2/menu", `{"day":"2026-10-24","meal":"dinner","note":"Salad"}`, user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	salad := decode[menu.MenuEntry](t, rec.Body.Bytes())

	t.Run("servings of a recipe entry, by another member", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/menu/%d", planned.ID), `{"servings":6}`, user2, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		got := decode[menu.MenuEntry](t, rec.Body.Bytes())
		assert.InDelta(t, 6.0, got.Servings, 0.001)
		assert.Equal(t, "Lasagne", got.RecipeTitle)
		assert.Equal(t, "2026-10-24", got.Day)
		assert.Equal(t, "dinner", got.Meal)
		db.AssertExists(t, "menu_entries", map[string]any{"id": planned.ID, "servings": 6, "recipe_id": lasagne.ID, "day": "2026-10-24"}, false)
	})

	t.Run("text of a note entry", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/menu/%d", salad.ID), `{"note":" Green salad "}`, user1, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, "Green salad", decode[menu.MenuEntry](t, rec.Body.Bytes()).Note)
		db.AssertExists(t, "menu_entries", map[string]any{"id": salad.ID, "note": "Green salad"}, false)
	})

	t.Run("nothing to change", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/menu/%d", salad.ID), `{"note":"Green salad"}`, user1, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, "Green salad", decode[menu.MenuEntry](t, rec.Body.Bytes()).Note)
	})

	t.Run("refusals", func(t *testing.T) {
		for _, req := range []struct {
			id   int64
			body string
		}{
			{planned.ID, `{"servings":-1}`},
			{planned.ID, `{"note":"Pie"}`}, // a recipe entry has no note
			{salad.ID, `{"servings":2}`},   // a note has no servings
			{salad.ID, `{"note":"   "}`},   // empty note
		} {
			rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/menu/%d", req.id), req.body, user1, "")
			assert.True(t, rec.Code == http.StatusBadRequest || rec.Code == http.StatusUnprocessableEntity, "%s: %d %s", req.body, rec.Code, rec.Body.String())
		}
		db.AssertExists(t, "menu_entries", map[string]any{"id": planned.ID, "servings": 6}, false)
		db.AssertExists(t, "menu_entries", map[string]any{"id": salad.ID, "note": "Green salad"}, false)

		rec := humaRequest(t, e, http.MethodPut, "/api/v2/menu/999999", `{"servings":2}`, user1, "")
		assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	})

	t.Run("link shares cannot update", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/menu/%d", planned.ID), `{"servings":2}`, menuLinkShareToken(t), "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		db.AssertExists(t, "menu_entries", map[string]any{"id": planned.ID, "servings": 6}, false)
	})

	t.Run("not logged in", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/menu/%d", planned.ID), `{"servings":2}`, "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	})
}

func TestHouseholdMenu_Move(t *testing.T) {
	e := setupMenuEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	user2 := humaTokenFor(t, &testuser2)
	add := func(day, note string) menu.MenuEntry {
		t.Helper()
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu",
			`{"day":"`+day+`","meal":"dinner","note":"`+note+`"}`, user1, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		return decode[menu.MenuEntry](t, rec.Body.Bytes())
	}
	move := func(id int64, body, token string) (int, menu.MenuEntry) {
		t.Helper()
		rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/menu/%d", id), body, token, "")
		if rec.Code != http.StatusOK {
			return rec.Code, menu.MenuEntry{}
		}
		return rec.Code, decode[menu.MenuEntry](t, rec.Body.Bytes())
	}

	curry := add("2026-10-20", "Curry")
	naan := add("2026-10-20", "Naan")
	lasagne := addMenuRecipe(t, e, user1, "Lasagne", 4)
	rec := humaRequest(t, e, http.MethodPost, "/api/v2/menu",
		fmt.Sprintf(`{"day":"2026-10-20","meal":"dinner","recipe_id":%d}`, lasagne.ID), user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	planned := decode[menu.MenuEntry](t, rec.Body.Bytes())

	t.Run("to an empty day, in another week", func(t *testing.T) {
		code, got := move(curry.ID, `{"day":"2026-11-03"}`, user1)
		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, "2026-11-03", got.Day)
		assert.Equal(t, "dinner", got.Meal)
		assert.Equal(t, "Curry", got.Note)
		assert.Equal(t, 0, got.Position)
		db.AssertExists(t, "menu_entries", map[string]any{"id": curry.ID, "day": "2026-11-03", "position": 0}, false)
	})

	t.Run("to a day with entries goes last, by another member", func(t *testing.T) {
		add("2026-10-22", "Soup")  // position 0
		add("2026-10-22", "Bread") // position 1
		code, got := move(planned.ID, `{"day":"2026-10-22"}`, user2)
		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, "2026-10-22", got.Day)
		assert.Equal(t, 2, got.Position)
		assert.Equal(t, "Lasagne", got.RecipeTitle)
		assert.InDelta(t, 4.0, got.Servings, 0.001)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/menu?from=2026-10-22&to=2026-10-22", "", user1, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		titles := []string{}
		for _, entry := range decode[menuWeek](t, rec.Body.Bytes()).Items {
			titles = append(titles, entry.Note+entry.RecipeTitle)
		}
		assert.Equal(t, []string{"Soup", "Bread", "Lasagne"}, titles)
	})

	t.Run("same day changes nothing", func(t *testing.T) {
		code, got := move(naan.ID, `{"day":"2026-10-20"}`, user1)
		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, "2026-10-20", got.Day)
		assert.Equal(t, naan.Position, got.Position)
		db.AssertExists(t, "menu_entries", map[string]any{"id": naan.ID, "day": "2026-10-20", "position": naan.Position}, false)
	})

	t.Run("invalid days", func(t *testing.T) {
		for _, body := range []string{`{"day":"2026-02-30"}`, `{"day":"tomorrow"}`, `{"day":""}`} {
			code, _ := move(naan.ID, body, user1)
			assert.True(t, code == http.StatusBadRequest || code == http.StatusUnprocessableEntity, "%s: %d", body, code)
		}
		db.AssertExists(t, "menu_entries", map[string]any{"id": naan.ID, "day": "2026-10-20"}, false)
	})

	t.Run("link shares cannot move", func(t *testing.T) {
		code, _ := move(naan.ID, `{"day":"2026-10-21"}`, menuLinkShareToken(t))
		assert.Equal(t, http.StatusForbidden, code)
		db.AssertExists(t, "menu_entries", map[string]any{"id": naan.ID, "day": "2026-10-20"}, false)
	})

	t.Run("not logged in", func(t *testing.T) {
		code, _ := move(naan.ID, `{"day":"2026-10-21"}`, "")
		assert.Equal(t, http.StatusUnauthorized, code)
	})
}
