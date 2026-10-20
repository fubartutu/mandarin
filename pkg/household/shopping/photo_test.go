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

package shopping

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"code.vikunja.io/api/pkg/files"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPNG makes a solid-colour PNG of the given size.
func testPNG(t *testing.T, width, height int) *bytes.Reader {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: 240, G: 140, B: 20, A: 255})
		}
	}
	buf := &bytes.Buffer{}
	require.NoError(t, png.Encode(buf, img))
	return bytes.NewReader(buf.Bytes())
}

func TestShoppingCatalog_Photo(t *testing.T) {
	t.Run("stored shrunk, as JPEG, and flagged on items", func(t *testing.T) {
		s := setup(t)
		entry := catalogEntry(t, s, "Vegemite")
		require.NoError(t, entry.SetPhoto(s, user1, testPNG(t, 2000, 1000)))
		assert.True(t, entry.HasPhoto)

		data, err := (&ShoppingCatalogItem{ID: entry.ID}).Photo(s)
		require.NoError(t, err)
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, "jpeg", format)
		assert.Equal(t, 1024, cfg.Width)
		assert.Equal(t, 512, cfg.Height)

		item := addItem(t, s, &ShoppingItem{ListID: groceries(t, s).ID, Name: "Vegemite"})
		assert.True(t, item.HasPhoto)
	})

	t.Run("small photos aren't enlarged", func(t *testing.T) {
		s := setup(t)
		entry := catalogEntry(t, s, "Milk")
		require.NoError(t, entry.SetPhoto(s, user1, testPNG(t, 300, 200)))
		data, err := entry.Photo(s)
		require.NoError(t, err)
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, 300, cfg.Width)
	})

	t.Run("replacing a photo deletes the old file", func(t *testing.T) {
		s := setup(t)
		entry := catalogEntry(t, s, "Milk")
		require.NoError(t, entry.SetPhoto(s, user1, testPNG(t, 50, 50)))
		oldFileID := entry.PhotoFileID
		require.NoError(t, entry.SetPhoto(s, user1, testPNG(t, 60, 60)))
		assert.NotEqual(t, oldFileID, entry.PhotoFileID)

		_, err := files.FileStat(&files.File{ID: oldFileID})
		require.Error(t, err, "old photo file removed")
	})

	t.Run("not an image", func(t *testing.T) {
		s := setup(t)
		err := catalogEntry(t, s, "Milk").SetPhoto(s, user1, strings.NewReader("definitely not a picture"))
		require.ErrorIs(t, err, ErrNotAnImage{})
	})

	t.Run("delete, and no photo is a 404 error", func(t *testing.T) {
		s := setup(t)
		entry := catalogEntry(t, s, "Milk")
		_, err := entry.Photo(s)
		require.ErrorIs(t, err, ErrNoPhoto{CatalogItemID: entry.ID})

		require.NoError(t, entry.SetPhoto(s, user1, testPNG(t, 50, 50)))
		require.NoError(t, entry.DeletePhoto(s))
		assert.False(t, entry.HasPhoto)
		_, err = entry.Photo(s)
		require.ErrorIs(t, err, ErrNoPhoto{CatalogItemID: entry.ID})
	})
}
