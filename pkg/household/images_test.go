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

package household

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sidewaysJPEG is a 40×20 JPEG whose EXIF says "rotate 90° to view" (6),
// the way a phone stores a portrait photo.
func sidewaysJPEG(t *testing.T, order binary.ByteOrder, mark string) []byte {
	t.Helper()
	plain := &bytes.Buffer{}
	require.NoError(t, jpeg.Encode(plain, image.NewRGBA(image.Rect(0, 0, 40, 20)), nil))

	tiff := &bytes.Buffer{}
	tiff.WriteString(mark)
	_ = binary.Write(tiff, order, uint16(42))
	_ = binary.Write(tiff, order, uint32(8))           // first IFD right after the header
	_ = binary.Write(tiff, order, uint16(1))           // one entry
	_ = binary.Write(tiff, order, []uint16{0x0112, 3}) // Orientation, SHORT
	_ = binary.Write(tiff, order, uint32(1))
	_ = binary.Write(tiff, order, []uint16{6, 0})
	_ = binary.Write(tiff, order, uint32(0)) // no next IFD
	app1 := append([]byte("Exif\x00\x00"), tiff.Bytes()...)

	out := &bytes.Buffer{}
	out.Write(plain.Bytes()[:2]) // SOI
	out.Write([]byte{0xFF, 0xE1})
	_ = binary.Write(out, binary.BigEndian, uint16(len(app1)+2)) //nolint:gosec // a few dozen bytes
	out.Write(app1)
	out.Write(plain.Bytes()[2:])
	return out.Bytes()
}

func TestExifOrientation(t *testing.T) {
	assert.Equal(t, 6, exifOrientation(sidewaysJPEG(t, binary.BigEndian, "MM")))
	assert.Equal(t, 6, exifOrientation(sidewaysJPEG(t, binary.LittleEndian, "II")))

	plain := &bytes.Buffer{}
	require.NoError(t, jpeg.Encode(plain, image.NewRGBA(image.Rect(0, 0, 4, 4)), nil))
	assert.Equal(t, 1, exifOrientation(plain.Bytes()), "no EXIF: as stored")
	assert.Equal(t, 1, exifOrientation([]byte("\x89PNG")), "not a JPEG")
	assert.Equal(t, 1, exifOrientation([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0xFF}), "cut short")
}

func TestUpright(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	assert.Equal(t, image.Pt(20, 40), upright(img, 6).Bounds().Size(), "portrait photo turned upright")
	assert.Equal(t, image.Pt(40, 20), upright(img, 1).Bounds().Size())
}
