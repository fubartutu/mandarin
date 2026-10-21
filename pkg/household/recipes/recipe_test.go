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
	"bytes"
	"image"
	"image/png"
	"testing"

	"code.vikunja.io/api/pkg/files"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecipeCreate(t *testing.T) {
	s := setup(t)
	r := &Recipe{
		Title:       "  Pancakes ",
		Ingredients: []*RecipeIngredient{{Text: " 100 g  plain flour "}, {Text: "  "}, {Group: "To serve", Text: "lemon wedges"}},
		Steps:       []string{"Mix.", " ", "Fry."},
		Tags:        []string{"Breakfast", " breakfast", "Sweet"},
		ImageIDs:    []int64{99}, // read-only: ignored
	}
	require.NoError(t, r.Create(s, user1))

	stored, err := getRecipe(s, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "Pancakes", stored.Title)
	assert.Equal(t, []string{"Mix.", "Fry."}, stored.Steps)
	assert.Equal(t, []string{"Breakfast", "Sweet"}, stored.Tags, "repeats dropped, ignoring case")
	assert.Empty(t, stored.ImageIDs)
	require.Len(t, stored.Ingredients, 2, "blank lines dropped")
	flour := stored.Ingredients[0]
	assert.Equal(t, RecipeIngredient{Text: "100 g plain flour", Quantity: 100, Unit: "g", Name: "plain flour"}, *flour)
	assert.Equal(t, "To serve", stored.Ingredients[1].Group)
	assert.Equal(t, int64(1), stored.CreatedByID)
}

func TestRecipeLimits(t *testing.T) {
	s := setup(t)
	for name, r := range map[string]*Recipe{
		"no title":      {Title: "  "},
		"long line":     {Title: "x", Ingredients: lines(string(make([]byte, 501)))},
		"too many tags": {Title: "x", Tags: make21Tags()},
	} {
		err := r.Create(s, user1)
		assert.ErrorAs(t, err, &ErrInvalidRecipe{}, name)
	}
}

func make21Tags() []string {
	tags := []string{}
	for i := range 21 {
		tags = append(tags, string(rune('a'+i)))
	}
	return tags
}

func TestRecipeReadAllSearches(t *testing.T) {
	s := setup(t)
	addRecipe(t, s, "Lasagne", "500 g beef mince")
	addRecipe(t, s, "Chicken stir fry", "2 chicken breasts")
	curry := addRecipe(t, s, "Green curry", "400 ml coconut milk")
	curry.Tags = []string{"Thai"}
	require.NoError(t, curry.Update(s, user1))

	titles := func(search string) []string {
		result, _, total, err := (&Recipe{}).ReadAll(s, user1, search, 0, 0)
		require.NoError(t, err)
		out := []string{}
		for _, r := range result.([]*Recipe) {
			out = append(out, r.Title)
		}
		assert.Equal(t, int64(len(out)), total)
		return out
	}
	assert.Equal(t, []string{"Chicken stir fry", "Green curry", "Lasagne"}, titles(""), "by title")
	assert.Equal(t, []string{"Chicken stir fry"}, titles("chicken"))
	assert.Equal(t, []string{"Lasagne"}, titles("mince"), "matches ingredients")
	assert.Equal(t, []string{"Green curry"}, titles("thai"), "matches tags")
}

func TestRecipePermissions(t *testing.T) {
	s := setup(t)
	r := addRecipe(t, s, "Toast", "2 slices bread")

	_, _, err := (&Recipe{ID: r.ID}).CanRead(s, linkShare)
	require.Error(t, err)
	can, err := (&Recipe{ID: r.ID}).CanUpdate(s, user1)
	require.NoError(t, err)
	assert.True(t, can)
	_, err = (&Recipe{ID: 9999}).CanUpdate(s, user1)
	assert.ErrorAs(t, err, &ErrRecipeDoesNotExist{})
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 3200, 1600))
	buf := &bytes.Buffer{}
	require.NoError(t, png.Encode(buf, img))
	return buf.Bytes()
}

func TestRecipeImages(t *testing.T) {
	s := setup(t)
	r := addRecipe(t, s, "Toast", "2 slices bread")

	for range maxImages {
		require.NoError(t, r.AddImage(t.Context(), s, user1, bytes.NewReader(testPNG(t))))
	}
	require.Len(t, r.ImageIDs, maxImages)
	require.ErrorAs(t, r.AddImage(t.Context(), s, user1, bytes.NewReader(testPNG(t))), &ErrTooManyImages{})
	require.ErrorAs(t, (&Recipe{ID: r.ID}).AddImage(t.Context(), s, user1, bytes.NewReader([]byte("nope"))), &ErrTooManyImages{})

	data, err := r.Image(r.ImageIDs[0])
	require.NoError(t, err)
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
	assert.Equal(t, imageMaxEdge, cfg.Width, "shrunk to fit")

	last := r.ImageIDs[3]
	require.NoError(t, r.MakeImageMain(s, last))
	assert.Equal(t, last, r.ImageIDs[0])
	require.NoError(t, r.DeleteImage(s, last))
	assert.NotContains(t, r.ImageIDs, last)
	_, err = r.Image(last)
	require.ErrorAs(t, err, &ErrImageDoesNotExist{})

	remaining := append([]int64{}, r.ImageIDs...)
	require.NoError(t, r.Delete(s, user1))
	for _, id := range remaining {
		assert.True(t, files.IsErrFileDoesNotExist((&files.File{ID: id}).LoadFileByID()), "file %d deleted with the recipe", id)
	}
}

func TestRecipeImageRejectsNonImages(t *testing.T) {
	s := setup(t)
	r := addRecipe(t, s, "Toast", "2 slices bread")
	assert.ErrorAs(t, r.AddImage(t.Context(), s, user1, bytes.NewReader([]byte("not an image"))), &ErrNotAnImage{})
}
