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
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	"io"

	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/modules/imageutils"
	"code.vikunja.io/api/pkg/web"

	"github.com/disintegration/imaging"
	"xorm.io/xorm"
)

// ErrNotAnImage means the upload couldn't be read as an image. Modules turn
// it into their own HTTP error. A picture too big to decode safely, or a
// server too busy decoding others, comes back as imageutils' own errors,
// which already carry an HTTP status.
var ErrNotAnImage = errors.New("not an image")

const jpegQuality = 80

// StoreJPEG stores src as a JPEG in Vikunja's file storage: turned upright
// from its EXIF orientation and shrunk to fit maxEdge×maxEdge pixels. A 5 MB
// phone photo ends up around 150 KB at 1024px.
func StoreJPEG(ctx context.Context, s *xorm.Session, a web.Auth, src io.Reader, maxEdge int, name string) (*files.File, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	// imageutils.Decode checks the size first and limits how many big images
	// are decoded at once, so uploads can't exhaust the server's memory.
	err = imageutils.Decode(ctx, imageutils.BytesOpener(data), func(img image.Image, _ string) error {
		img = upright(img, exifOrientation(data))
		resized := imaging.Fit(img, maxEdge, maxEdge, imaging.Lanczos)
		return jpeg.Encode(buf, resized, &jpeg.Options{Quality: jpegQuality})
	})
	if errors.Is(err, imageutils.ErrUndecodable) {
		return nil, ErrNotAnImage
	}
	if err != nil {
		return nil, err
	}
	return files.CreateWithMimeAndSession(s, bytes.NewReader(buf.Bytes()), name, uint64(buf.Len()), a, "image/jpeg", true)
}

// upright undoes the rotation a camera recorded instead of rotating the
// pixels (EXIF orientation 1–8), so a photo taken sideways shows upright.
func upright(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return imaging.FlipH(img)
	case 3:
		return imaging.Rotate180(img)
	case 4:
		return imaging.FlipV(img)
	case 5:
		return imaging.Transpose(img)
	case 6:
		return imaging.Rotate270(img)
	case 7:
		return imaging.Transverse(img)
	case 8:
		return imaging.Rotate90(img)
	}
	return img
}

// exifOrientation reads the orientation tag from a JPEG's EXIF data, or 1
// (as stored) when there is none.
func exifOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1 // not a JPEG
	}
	// Walk the JPEG segments to APP1 ("Exif\0\0" then a TIFF block).
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		length := int(binary.BigEndian.Uint16(data[i+2:]))
		if marker == 0xDA || length < 2 || i+2+length > len(data) {
			return 1 // image data starts: no EXIF before it
		}
		segment := data[i+4 : i+2+length]
		if marker == 0xE1 && len(segment) > 14 && string(segment[:6]) == "Exif\x00\x00" {
			return tiffOrientation(segment[6:])
		}
		i += 2 + length
	}
	return 1
}

func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	ifd := int(order.Uint32(tiff[4:8])) //nolint:gosec // len(tiff) >= 8 checked above
	if ifd < 8 || ifd+2 > len(tiff) {
		return 1
	}
	count := int(order.Uint16(tiff[ifd : ifd+2])) //nolint:gosec // ifd+2 <= len(tiff) checked above
	for n := 0; n < count; n++ {
		start := ifd + 2 + n*12
		if start+12 > len(tiff) {
			return 1
		}
		entry := tiff[start : start+12]         //nolint:gosec // start+12 <= len(tiff) checked above
		if order.Uint16(entry[0:2]) == 0x0112 { //nolint:gosec // entry is a [12]byte; tag 0x0112 is Orientation
			if v := int(order.Uint16(entry[8:10])); v >= 1 && v <= 8 { //nolint:gosec // entry is a [12]byte
				return v
			}
			return 1
		}
	}
	return 1
}
