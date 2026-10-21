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
	"errors"
	"image/jpeg"
	"io"

	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/modules/imageutils"
	"code.vikunja.io/api/pkg/web"

	"github.com/disintegration/imaging"
	"xorm.io/xorm"
)

// ErrNotAnImage means the upload couldn't be read as an image (or was too big
// to decode safely). Modules turn it into their own HTTP error.
var ErrNotAnImage = errors.New("not an image")

const jpegQuality = 80

// StoreJPEG stores src as a JPEG in Vikunja's file storage: turned upright
// from its EXIF orientation and shrunk to fit maxEdge×maxEdge pixels. A 5 MB
// phone photo ends up around 150 KB at 1024px.
func StoreJPEG(s *xorm.Session, a web.Auth, src io.Reader, maxEdge int, name string) (*files.File, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	// Check the size first so a huge image can't exhaust memory when decoded.
	if _, err := imageutils.ValidateReader(bytes.NewReader(data)); err != nil {
		return nil, ErrNotAnImage
	}
	// AutoOrientation reads the phone's EXIF rotation tag, so a photo taken
	// sideways comes out upright.
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, ErrNotAnImage
	}

	resized := imaging.Fit(img, maxEdge, maxEdge, imaging.Lanczos)
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, resized, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return files.CreateWithMimeAndSession(s, bytes.NewReader(buf.Bytes()), name, uint64(buf.Len()), a, "image/jpeg", true)
}
