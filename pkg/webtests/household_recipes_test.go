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
	"bytes"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/household/recipes"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mandarin: /api/v2/recipes webtests.

func setupRecipesEnv(t *testing.T) *echo.Echo {
	t.Helper()
	files.InitTestFileHandler()
	e, err := setupTestEnv()
	require.NoError(t, err)
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.Sync2(recipes.GetTables()...))
	_, err = x.Exec("DELETE FROM recipes")
	require.NoError(t, err)
	return e
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	pic := &bytes.Buffer{}
	require.NoError(t, png.Encode(pic, image.NewRGBA(image.Rect(0, 0, 40, 30))))
	return pic.Bytes()
}

// fakeScraperAndImages stands in for the scraper container and a site with a picture.
func fakeScraperAndImages(t *testing.T) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/scrape":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"title":"Pancakes","source_url":"https://example.com/pancakes","servings":4,
				"ingredients":[{"group":"","text":"100 g plain flour","quantity":100,"unit":"g","name":"plain flour","note":""}],
				"steps":["Mix.","Fry."],"tags":["Breakfast"],"image_urls":["%s/pancakes.png"]}`, server.URL)
		case "/parse":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ingredients":[{"text":"3 eggs","quantity":3,"unit":"","name":"eggs","note":"from the scraper"}]}`))
		case "/pancakes.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(testPNG(t))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("MANDARIN_SCRAPER_URL", server.URL)
	return server
}

func TestHouseholdRecipes(t *testing.T) {
	e := setupRecipesEnv(t)
	t.Setenv("MANDARIN_SCRAPER_URL", "http://127.0.0.1:1") // down
	user1 := humaTokenFor(t, &testuser1)
	user2 := humaTokenFor(t, &testuser2)

	// 1. A typed-in recipe saves even with the scraper down (simple parser).
	rec := humaRequest(t, e, http.MethodPost, "/api/v2/recipes",
		`{"title":"Toast","ingredients":[{"text":"2 slices bread, toasted"}],"steps":["Toast it."],"tags":["Breakfast"]}`, user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	toast := decode[recipes.Recipe](t, rec.Body.Bytes())
	require.Len(t, toast.Ingredients, 1)
	assert.Equal(t, recipes.RecipeIngredient{Text: "2 slices bread, toasted", Quantity: 2, Unit: "slices", Name: "bread", Note: "toasted"}, *toast.Ingredients[0])
	path := fmt.Sprintf("/api/v2/recipes/%d", toast.ID)

	// 2. user2 reads and PATCHes it.
	rec = humaRequest(t, e, http.MethodGet, path, "", user2, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"max_permission":2`)
	rec = humaRequest(t, e, http.MethodPatch, path, `{"notes":"Use sourdough."}`, user2, "application/merge-patch+json")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	patched := decode[recipes.Recipe](t, rec.Body.Bytes())
	assert.Equal(t, "Use sourdough.", patched.Notes)
	assert.Equal(t, "bread", patched.Ingredients[0].Name, "PATCH keeps the ingredients")

	// 3. Search.
	rec = humaRequest(t, e, http.MethodGet, "/api/v2/recipes?q=bread", "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Len(t, decode[struct{ Items []recipes.Recipe }](t, rec.Body.Bytes()).Items, 1)

	// 4. Import needs the scraper.
	rec = humaRequest(t, e, http.MethodPost, "/api/v2/recipes/import", `{"url":"https://example.com/pancakes"}`, user1, "")
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())

	// 5. Delete.
	rec = humaRequest(t, e, http.MethodDelete, path, "", user1, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	db.AssertMissing(t, "recipes", map[string]any{"id": toast.ID})
}

func TestHouseholdRecipes_ImportAndImages(t *testing.T) {
	e := setupRecipesEnv(t)
	site := fakeScraperAndImages(t)
	token := humaTokenFor(t, &testuser1)

	// Preview, then save it as the app does.
	rec := humaRequest(t, e, http.MethodPost, "/api/v2/recipes/import", `{"url":"https://example.com/pancakes"}`, token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	preview := decode[recipes.Recipe](t, rec.Body.Bytes())
	assert.Equal(t, "Pancakes", preview.Title)
	require.Equal(t, []string{site.URL + "/pancakes.png"}, preview.ImageURLs)
	db.AssertMissing(t, "recipes", map[string]any{"title": "Pancakes"})

	rec = humaRequest(t, e, http.MethodPost, "/api/v2/recipes",
		`{"title":"Pancakes","ingredients":[{"text":"3 eggs"}]}`, token, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	saved := decode[recipes.Recipe](t, rec.Body.Bytes())
	assert.Equal(t, "from the scraper", saved.Ingredients[0].Note, "parsed by the scraper")
	path := fmt.Sprintf("/api/v2/recipes/%d", saved.ID)

	// Fetch the found picture, upload another, read one back.
	rec = humaRequest(t, e, http.MethodPost, path+"/images/fetch", fmt.Sprintf(`{"url":"%s/pancakes.png"}`, site.URL), token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, decode[recipes.Recipe](t, rec.Body.Bytes()).ImageIDs, 1)

	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	part, err := w.CreateFormFile("image", "plate.png")
	require.NoError(t, err)
	_, err = part.Write(testPNG(t))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	rec = humaRequest(t, e, http.MethodPost, path+"/images", buf.String(), token, w.FormDataContentType())
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	ids := decode[recipes.Recipe](t, rec.Body.Bytes()).ImageIDs
	require.Len(t, ids, 2)

	rec = humaRequest(t, e, http.MethodPost, fmt.Sprintf("%s/images/%d/main", path, ids[1]), "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, ids[1], decode[recipes.Recipe](t, rec.Body.Bytes()).ImageIDs[0])

	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("%s/images/%d", path, ids[0]), "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))

	rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("%s/images/%d", path, ids[0]), "", token, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("%s/images/%d", path, ids[0]), "", token, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	// An unknown recipe is a 404.
	rec = humaRequest(t, e, http.MethodGet, "/api/v2/recipes/99999", "", token, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}
