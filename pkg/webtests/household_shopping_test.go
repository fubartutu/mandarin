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
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/household/shopping"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/auth"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mandarin: /api/v2/shopping webtests.

// setupShoppingEnv builds the API and a freshly seeded household. The
// shared test env doesn't know our tables, so we create them here.
func setupShoppingEnv(t *testing.T) *echo.Echo {
	t.Helper()
	e, err := setupTestEnv()
	require.NoError(t, err)

	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.Sync2(shopping.GetTables()...))

	s := db.NewSession()
	defer s.Close()
	for _, table := range []string{"shopping_items", "shopping_lists", "shopping_categories", "shopping_shops", "shopping_catalog_items", "shopping_not_stocked"} {
		_, err := s.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
	require.NoError(t, shopping.Seed(s))
	require.NoError(t, s.Commit())
	return e
}

// decode reads a JSON response body into T. Go generics: decode[shopping.ShoppingList](t, rec.Body.Bytes()).
func decode[T any](t *testing.T, body []byte) T {
	t.Helper()
	var out T
	require.NoError(t, json.Unmarshal(body, &out), string(body))
	return out
}

func TestHouseholdShopping(t *testing.T) {
	e := setupShoppingEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	user2 := humaTokenFor(t, &testuser2)

	// 1. Both users see the seeded list.
	rec := humaRequest(t, e, http.MethodGet, "/api/v2/shopping/lists", "", user2, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	lists := decode[struct{ Items []shopping.ShoppingList }](t, rec.Body.Bytes())
	require.Len(t, lists.Items, 1)
	listID := lists.Items[0].ID
	listPath := fmt.Sprintf("/api/v2/shopping/lists/%d", listID)

	// 2. user1 adds an item; user2 can see it.
	rec = humaRequest(t, e, http.MethodPost, listPath+"/items",
		`{"name":"Mince","quantity":500,"unit":"g","category_id":3}`, user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	item := decode[shopping.ShoppingItem](t, rec.Body.Bytes())
	assert.Equal(t, listID, item.ListID)
	itemPath := fmt.Sprintf("%s/items/%d", listPath, item.ID)

	rec = humaRequest(t, e, http.MethodGet, itemPath, "", user2, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"unit":"g"`)
	assert.Contains(t, rec.Body.String(), `"max_permission":2`)

	// 3. user2 ticks it with a PATCH (AutoPatch merges into a PUT).
	rec = humaRequest(t, e, http.MethodPatch, itemPath, `{"done":true}`, user2, "application/merge-patch+json")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	ticked := decode[shopping.ShoppingItem](t, rec.Body.Bytes())
	assert.True(t, ticked.Done)
	assert.NotNil(t, ticked.DoneAt)
	assert.Equal(t, "Mince", ticked.Name, "PATCH keeps the other fields")

	// 4. Clear ticked items.
	rec = humaRequest(t, e, http.MethodPost, listPath+"/clear-done", "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, int64(1), decode[struct{ Deleted int64 }](t, rec.Body.Bytes()).Deleted)
	db.AssertMissing(t, "shopping_items", map[string]any{"id": item.ID})

	// 5. Categories are listed in order.
	rec = humaRequest(t, e, http.MethodGet, "/api/v2/shopping/categories", "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	categories := decode[struct{ Items []shopping.ShoppingCategory }](t, rec.Body.Bytes())
	require.Len(t, categories.Items, len(shopping.DefaultCategories))
	assert.Equal(t, "Fruit & Veg", categories.Items[0].Title)
}

func TestHouseholdShopping_Refusals(t *testing.T) {
	e := setupShoppingEnv(t)
	user1 := humaTokenFor(t, &testuser1)

	rec := humaRequest(t, e, http.MethodPost, "/api/v2/shopping/lists", `{"title":"Kmart"}`, user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	kmart := decode[shopping.ShoppingList](t, rec.Body.Bytes())

	rec = humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/shopping/lists/%d/items", kmart.ID), `{"name":"Shoes"}`, user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	shoes := decode[shopping.ShoppingItem](t, rec.Body.Bytes())

	t.Run("not logged in", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodGet, "/api/v2/shopping/lists", "", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	})

	t.Run("link shares are not household members", func(t *testing.T) {
		token, err := auth.NewLinkShareJWTAuthtoken(&models.LinkSharing{
			ID:          1,
			Hash:        "test",
			ProjectID:   1,
			Permission:  models.PermissionRead,
			SharingType: models.SharingTypeWithoutPassword,
			SharedByID:  1,
		})
		require.NoError(t, err)

		for _, req := range []struct{ method, path, body string }{
			{http.MethodGet, "/api/v2/shopping/lists", ""},
			{http.MethodPost, "/api/v2/shopping/lists", `{"title":"x"}`},
			{http.MethodGet, fmt.Sprintf("/api/v2/shopping/lists/%d/items", kmart.ID), ""},
			{http.MethodPost, fmt.Sprintf("/api/v2/shopping/lists/%d/clear-done", kmart.ID), ""},
			{http.MethodGet, "/api/v2/shopping/categories", ""},
		} {
			rec := humaRequest(t, e, req.method, req.path, req.body, token, "")
			assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s: %s", req.method, req.path, rec.Body.String())
		}
	})

	t.Run("unknown ids are 404", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodGet, "/api/v2/shopping/lists/9999", "", user1, "")
		assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		rec = humaRequest(t, e, http.MethodPost, "/api/v2/shopping/lists/9999/items", `{"name":"Milk"}`, user1, "")
		assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		rec = humaRequest(t, e, http.MethodDelete, "/api/v2/shopping/categories/9999", "", user1, "")
		assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	})

	t.Run("an item can't be reached through another list", func(t *testing.T) {
		groceriesID := kmart.ID - 1
		rec := humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/shopping/lists/%d/items/%d", groceriesID, shoes.ID), "", user1, "")
		assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	})

	t.Run("validation", func(t *testing.T) {
		path := fmt.Sprintf("/api/v2/shopping/lists/%d/items", kmart.ID)
		rec := humaRequest(t, e, http.MethodPost, path, `{"name":""}`, user1, "")
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		rec = humaRequest(t, e, http.MethodPost, path, `{"name":"Milk","quantity":-1}`, user1, "")
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		rec = humaRequest(t, e, http.MethodPost, path, `{"name":"Milk","category_id":9999}`, user1, "")
		assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	})
}

// photoUpload builds a multipart body with a small PNG in the "photo" field.
func photoUpload(t *testing.T) (body, contentType string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	pic := &bytes.Buffer{}
	require.NoError(t, png.Encode(pic, img))

	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	part, err := w.CreateFormFile("photo", "vegemite.png")
	require.NoError(t, err)
	_, err = part.Write(pic.Bytes())
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.String(), w.FormDataContentType()
}

func TestHouseholdShopping_ShopsAndCatalog(t *testing.T) {
	files.InitTestFileHandler()
	e := setupShoppingEnv(t)
	token := humaTokenFor(t, &testuser1)

	// Ids keep counting across test resets, so look the seeded list up.
	rec := humaRequest(t, e, http.MethodGet, "/api/v2/shopping/lists", "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	listPath := fmt.Sprintf("/api/v2/shopping/lists/%d", decode[struct{ Items []shopping.ShoppingList }](t, rec.Body.Bytes()).Items[0].ID)

	// Shops: add two, they show up on the list.
	rec = humaRequest(t, e, http.MethodPost, listPath+"/shops", `{"title":"Aldi"}`, token, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	aldi := decode[shopping.ShoppingShop](t, rec.Body.Bytes())
	rec = humaRequest(t, e, http.MethodPost, listPath+"/shops", `{"title":"Supabarn"}`, token, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	rec = humaRequest(t, e, http.MethodGet, listPath, "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	list := decode[shopping.ShoppingList](t, rec.Body.Bytes())
	require.Len(t, list.Shops, 2)
	assert.Equal(t, "Aldi", list.Shops[0].Title)

	// Catalogue: search finds the pre-loaded Vegemite.
	rec = humaRequest(t, e, http.MethodGet, "/api/v2/shopping/catalog?q=vegemite", "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	found := decode[struct {
		Items []shopping.ShoppingCatalogItem
	}](t, rec.Body.Bytes())
	require.Len(t, found.Items, 1)
	entryPath := fmt.Sprintf("/api/v2/shopping/catalog/%d", found.Items[0].ID)

	// Mark it not stocked at Aldi with a PATCH.
	rec = humaRequest(t, e, http.MethodPatch, entryPath, fmt.Sprintf(`{"not_stocked_shop_ids":[%d]}`, aldi.ID), token, "application/merge-patch+json")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []int64{aldi.ID}, decode[shopping.ShoppingCatalogItem](t, rec.Body.Bytes()).NotStockedShopIDs)

	// Adding "vegemite" to the list picks up the category and the mark.
	rec = humaRequest(t, e, http.MethodPost, listPath+"/items", `{"name":"vegemite"}`, token, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	item := decode[shopping.ShoppingItem](t, rec.Body.Bytes())
	assert.Equal(t, found.Items[0].CategoryID, item.CategoryID)
	assert.Equal(t, []int64{aldi.ID}, item.NotStockedShopIDs)
	assert.False(t, item.HasPhoto)

	// Photo: upload, fetch, see it flagged on the item, delete.
	body, contentType := photoUpload(t)
	rec = humaRequest(t, e, http.MethodPut, entryPath+"/photo", body, token, contentType)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.True(t, decode[shopping.ShoppingCatalogItem](t, rec.Body.Bytes()).HasPhoto)

	rec = humaRequest(t, e, http.MethodGet, entryPath+"/photo", "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
	_, format, err := image.DecodeConfig(bytes.NewReader(rec.Body.Bytes()))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)

	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("%s/items/%d", listPath, item.ID), "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"has_photo":true`)

	rec = humaRequest(t, e, http.MethodDelete, entryPath+"/photo", "", token, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	rec = humaRequest(t, e, http.MethodGet, entryPath+"/photo", "", token, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	// Duplicate names are a 409.
	rec = humaRequest(t, e, http.MethodPost, "/api/v2/shopping/catalog", `{"name":"VEGEMITE"}`, token, "")
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

	// Deleting the shop clears the mark.
	rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("%s/shops/%d", listPath, aldi.ID), "", token, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	rec = humaRequest(t, e, http.MethodGet, entryPath, "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Empty(t, decode[shopping.ShoppingCatalogItem](t, rec.Body.Bytes()).NotStockedShopIDs)

	t.Run("not an image is a 400", func(t *testing.T) {
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		part, err := w.CreateFormFile("photo", "notes.png")
		require.NoError(t, err)
		_, err = part.Write([]byte("not a picture"))
		require.NoError(t, err)
		require.NoError(t, w.Close())
		rec := humaRequest(t, e, http.MethodPut, entryPath+"/photo", buf.String(), token, w.FormDataContentType())
		assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})

	t.Run("link shares are refused", func(t *testing.T) {
		share, err := auth.NewLinkShareJWTAuthtoken(&models.LinkSharing{
			ID: 1, Hash: "test", ProjectID: 1, Permission: models.PermissionRead,
			SharingType: models.SharingTypeWithoutPassword, SharedByID: 1,
		})
		require.NoError(t, err)
		for _, req := range []struct{ method, path string }{
			{http.MethodGet, listPath + "/shops"},
			{http.MethodGet, "/api/v2/shopping/catalog"},
			{http.MethodGet, entryPath},
			{http.MethodGet, entryPath + "/photo"},
			{http.MethodDelete, entryPath + "/photo"},
		} {
			rec := humaRequest(t, e, req.method, req.path, "", share, "")
			assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s: %s", req.method, req.path, rec.Body.String())
		}
	})
}
