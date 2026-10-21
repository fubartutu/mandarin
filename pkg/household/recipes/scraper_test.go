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

package recipes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeScraper answers like household-scraper/app.py.
func fakeScraper(t *testing.T) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/scrape":
			var body struct{ URL string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.URL == "https://example.com/none" {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"detail": "No recipe found on that page."}`))
				return
			}
			_, _ = w.Write([]byte(`{"title": "Pasta", "source_url": "https://example.com/pasta", "servings": 4,
				"ingredients": [{"group": "", "text": "2 cups (250g) flour", "quantity": 250, "unit": "g", "name": "flour", "note": ""}],
				"steps": ["Boil."], "tags": ["Mains"], "image_urls": ["https://example.com/a.jpg"]}`))
		case "/paperless/documents":
			assert.Equal(t, "beef pie", r.URL.Query().Get("q"))
			_, _ = w.Write([]byte(`{"documents": [{"id": 7, "title": "Beef pie", "created": "2026-01-02"}], "ai_available": true}`))
		case "/paperless/documents/7/thumb":
			w.Header().Set("Content-Type", "image/webp")
			_, _ = w.Write([]byte("webp"))
		case "/paperless/documents/7/image":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("jpeg"))
		case "/paperless/documents/7/import":
			method := "rules"
			if r.URL.Query().Get("ai_first") == "true" {
				method = "ai"
			}
			_, _ = w.Write([]byte(`{"title": "Beef pie", "source_url": "https://paperless.example.com/documents/7/details",
				"ingredients": [{"group": "", "text": "1 kg beef", "quantity": 1, "unit": "kg", "name": "beef", "note": ""}],
				"steps": ["Bake."], "image_urls": ["paperless:documents/7/image"],
				"import_method": "` + method + `", "import_complete": true, "ai_available": true}`))
		case "/parse":
			var body struct{ Lines []string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			out := []map[string]any{}
			for _, line := range body.Lines {
				out = append(out, map[string]any{"text": line, "quantity": 7, "unit": "x", "name": "parsed " + line, "note": ""})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ingredients": out})
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("MANDARIN_SCRAPER_URL", server.URL)
}

func TestImport(t *testing.T) {
	fakeScraper(t)
	r, err := Import(context.Background(), "https://example.com/pasta")
	require.NoError(t, err)
	assert.Equal(t, "Pasta", r.Title)
	assert.InDelta(t, 4.0, r.Servings, 0)
	assert.Equal(t, []string{"https://example.com/a.jpg"}, r.ImageURLs)
	require.Len(t, r.Ingredients, 1)
	assert.Equal(t, "flour", r.Ingredients[0].Name, "the scraper's parse is kept")

	_, err = Import(context.Background(), "https://example.com/none")
	var failed ErrImportFailed
	require.ErrorAs(t, err, &failed)
	assert.Equal(t, "No recipe found on that page.", failed.Reason)

	_, err = Import(context.Background(), "ftp://example.com")
	assert.ErrorAs(t, err, &ErrImportFailed{})
}

func TestImportWithScraperDown(t *testing.T) {
	_, err := Import(context.Background(), "https://example.com/pasta")
	assert.ErrorAs(t, err, &ErrScraperUnavailable{})
}

func TestParseIngredients(t *testing.T) {
	t.Run("uses the scraper and replaces any stale parse", func(t *testing.T) {
		fakeScraper(t)
		ings := []*RecipeIngredient{{Text: "a", Name: "stale"}, {Text: "b"}}
		ParseIngredients(context.Background(), ings)
		assert.Equal(t, "parsed a", ings[0].Name)
		assert.InDelta(t, 7.0, ings[1].Quantity, 0)
	})
	t.Run("leaves lines for the simple parser when the scraper is down", func(t *testing.T) {
		ings := []*RecipeIngredient{{Text: "2 eggs", Name: "stale"}}
		ParseIngredients(context.Background(), ings)
		assert.Empty(t, ings[0].Name)

		s := setup(t)
		r := &Recipe{Title: "Eggs", Ingredients: ings}
		require.NoError(t, r.Create(s, user1))
		assert.Equal(t, "eggs", r.Ingredients[0].Name)
	})
}

func TestPaperless(t *testing.T) {
	fakeScraper(t)
	ctx := context.Background()

	docs, err := ListPaperless(ctx, "beef pie")
	require.NoError(t, err)
	require.Len(t, docs.Documents, 1)
	assert.Equal(t, "Beef pie", docs.Documents[0].Title)
	assert.True(t, docs.AIAvailable)

	thumb, contentType, err := PaperlessThumbnail(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, "webp", string(thumb))
	assert.Equal(t, "image/webp", contentType)

	r, err := ImportPaperless(ctx, 7, false)
	require.NoError(t, err)
	assert.Equal(t, "rules", r.ImportMethod)
	assert.True(t, r.ImportComplete)
	assert.Equal(t, []string{"paperless:documents/7/image"}, r.ImageURLs)
	r, err = ImportPaperless(ctx, 7, true)
	require.NoError(t, err)
	assert.Equal(t, "ai", r.ImportMethod)

	// The scanned page is fetched from the scraper, which has the Paperless token.
	page, err := DownloadImage(ctx, "paperless:documents/7/image")
	require.NoError(t, err)
	assert.Equal(t, "jpeg", string(page))
}
