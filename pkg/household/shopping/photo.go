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
	"image/jpeg"
	"io"

	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/modules/imageutils"
	"code.vikunja.io/api/pkg/web"

	"github.com/disintegration/imaging"
	"xorm.io/xorm"
)

// Photos are shrunk to fit this box and stored as JPEG, so a 5 MB phone
// photo takes about 150 KB. Items without a photo use no space at all.
const (
	photoMaxEdge     = 1024
	photoJPEGQuality = 80
)

// SetPhoto stores src as the entry's photo, replacing any earlier one.
// Callers must check CanUpdate first.
func (c *ShoppingCatalogItem) SetPhoto(s *xorm.Session, a web.Auth, src io.Reader) error {
	stored, err := getCatalogItem(s, c.ID)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	// Check the size first so a huge image can't exhaust memory when decoded.
	if _, err := imageutils.ValidateReader(bytes.NewReader(data)); err != nil {
		return ErrNotAnImage{}
	}
	// AutoOrientation reads the phone's EXIF rotation tag, so a photo taken
	// sideways comes out upright.
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return ErrNotAnImage{}
	}

	resized := imaging.Fit(img, photoMaxEdge, photoMaxEdge, imaging.Lanczos)
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, resized, &jpeg.Options{Quality: photoJPEGQuality}); err != nil {
		return err
	}

	f, err := files.CreateWithMimeAndSession(s, bytes.NewReader(buf.Bytes()), "shopping-item.jpg", uint64(buf.Len()), a, "image/jpeg", true)
	if err != nil {
		return err
	}
	if err := deletePhotoFile(s, stored.PhotoFileID); err != nil {
		return err
	}

	stored.PhotoFileID = f.ID
	if _, err := s.ID(stored.ID).Cols("photo_file_id").Update(stored); err != nil {
		return err
	}
	if err := recordChange(s, entityCatalog, stored.ID, "", false); err != nil {
		return err
	}
	*c = *stored
	return fillDetails(s, c)
}

// Photo returns the entry's photo as JPEG bytes. Callers must check CanRead first.
func (c *ShoppingCatalogItem) Photo(s *xorm.Session) ([]byte, error) {
	stored, err := getCatalogItem(s, c.ID)
	if err != nil {
		return nil, err
	}
	if stored.PhotoFileID == 0 {
		return nil, ErrNoPhoto{CatalogItemID: c.ID}
	}
	f := &files.File{ID: stored.PhotoFileID}
	if err := f.LoadFileByID(); err != nil {
		if files.IsErrFileDoesNotExist(err) {
			return nil, ErrNoPhoto{CatalogItemID: c.ID}
		}
		return nil, err
	}
	defer f.File.Close()
	return io.ReadAll(f.File)
}

// DeletePhoto removes the entry's photo. Callers must check CanUpdate first.
func (c *ShoppingCatalogItem) DeletePhoto(s *xorm.Session) error {
	stored, err := getCatalogItem(s, c.ID)
	if err != nil {
		return err
	}
	if err := deletePhotoFile(s, stored.PhotoFileID); err != nil {
		return err
	}
	stored.PhotoFileID = 0
	if _, err := s.ID(stored.ID).Cols("photo_file_id").Update(stored); err != nil {
		return err
	}
	if err := recordChange(s, entityCatalog, stored.ID, "", false); err != nil {
		return err
	}
	*c = *stored
	return fillDetails(s, c)
}
