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
	"context"
	"errors"
	"io"

	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// Photos are shrunk to fit this box and stored as JPEG, so a 5 MB phone
// photo takes about 150 KB. Items without a photo use no space at all.
const photoMaxEdge = 1024

// SetPhoto stores src as the entry's photo, replacing any earlier one.
// Callers must check CanUpdate first.
func (c *ShoppingCatalogItem) SetPhoto(ctx context.Context, s *xorm.Session, a web.Auth, src io.Reader) error {
	stored, err := getCatalogItem(s, c.ID)
	if err != nil {
		return err
	}
	f, err := household.StoreJPEG(ctx, s, a, src, photoMaxEdge, "shopping-item.jpg")
	if errors.Is(err, household.ErrNotAnImage) {
		return ErrNotAnImage{}
	}
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
