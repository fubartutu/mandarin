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
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/web"
)

// Mandarin's error codes start at 90001 so they never collide with upstream
// Vikunja's (which are below 20000). The frontend can match on Code.

// ErrListDoesNotExist is returned when a shopping list id is unknown.
type ErrListDoesNotExist struct {
	ListID int64
}

func (err ErrListDoesNotExist) Error() string {
	return fmt.Sprintf("Shopping list does not exist [ListID: %d]", err.ListID)
}

// ErrCodeListDoesNotExist holds the unique world-error code of this error
const ErrCodeListDoesNotExist = 90001

// HTTPError holds the http error description
func (err ErrListDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeListDoesNotExist, Message: "This shopping list does not exist."}
}

// ErrItemDoesNotExist is returned when an item id is unknown or not on the list in the URL.
type ErrItemDoesNotExist struct {
	ItemID int64
}

func (err ErrItemDoesNotExist) Error() string {
	return fmt.Sprintf("Shopping item does not exist [ItemID: %d]", err.ItemID)
}

// ErrCodeItemDoesNotExist holds the unique world-error code of this error
const ErrCodeItemDoesNotExist = 90002

// HTTPError holds the http error description
func (err ErrItemDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeItemDoesNotExist, Message: "This shopping item does not exist."}
}

// ErrCategoryDoesNotExist is returned when a category id is unknown.
type ErrCategoryDoesNotExist struct {
	CategoryID int64
}

func (err ErrCategoryDoesNotExist) Error() string {
	return fmt.Sprintf("Shopping category does not exist [CategoryID: %d]", err.CategoryID)
}

// ErrCodeCategoryDoesNotExist holds the unique world-error code of this error
const ErrCodeCategoryDoesNotExist = 90003

// HTTPError holds the http error description
func (err ErrCategoryDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeCategoryDoesNotExist, Message: "This shopping category does not exist."}
}

// ErrShopDoesNotExist is returned when a shop id is unknown or not on the list in the URL.
type ErrShopDoesNotExist struct {
	ShopID int64
}

func (err ErrShopDoesNotExist) Error() string {
	return fmt.Sprintf("Shop does not exist [ShopID: %d]", err.ShopID)
}

// ErrCodeShopDoesNotExist holds the unique world-error code of this error
const ErrCodeShopDoesNotExist = 90004

// HTTPError holds the http error description
func (err ErrShopDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeShopDoesNotExist, Message: "This shop does not exist."}
}

// ErrCatalogItemDoesNotExist is returned when a catalogue id is unknown.
type ErrCatalogItemDoesNotExist struct {
	CatalogItemID int64
}

func (err ErrCatalogItemDoesNotExist) Error() string {
	return fmt.Sprintf("Catalogue item does not exist [CatalogItemID: %d]", err.CatalogItemID)
}

// ErrCodeCatalogItemDoesNotExist holds the unique world-error code of this error
const ErrCodeCatalogItemDoesNotExist = 90005

// HTTPError holds the http error description
func (err ErrCatalogItemDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeCatalogItemDoesNotExist, Message: "This catalogue item does not exist."}
}

// ErrCatalogItemExists is returned when a name is already in the catalogue.
type ErrCatalogItemExists struct {
	Name string
}

func (err ErrCatalogItemExists) Error() string {
	return fmt.Sprintf("Catalogue item already exists [Name: %s]", err.Name)
}

// ErrCodeCatalogItemExists holds the unique world-error code of this error
const ErrCodeCatalogItemExists = 90006

// HTTPError holds the http error description
func (err ErrCatalogItemExists) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusConflict, Code: ErrCodeCatalogItemExists, Message: "An item with this name is already in the catalogue."}
}

// ErrNoPhoto is returned when a catalogue entry has no photo.
type ErrNoPhoto struct {
	CatalogItemID int64
}

func (err ErrNoPhoto) Error() string {
	return fmt.Sprintf("Catalogue item has no photo [CatalogItemID: %d]", err.CatalogItemID)
}

// ErrCodeNoPhoto holds the unique world-error code of this error
const ErrCodeNoPhoto = 90007

// HTTPError holds the http error description
func (err ErrNoPhoto) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeNoPhoto, Message: "This item has no photo."}
}

// ErrNotAnImage is returned when an uploaded photo can't be read as an image.
type ErrNotAnImage struct{}

func (err ErrNotAnImage) Error() string {
	return "Uploaded file is not an image"
}

// ErrCodeNotAnImage holds the unique world-error code of this error
const ErrCodeNotAnImage = 90008

// HTTPError holds the http error description
func (err ErrNotAnImage) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeNotAnImage, Message: "The uploaded file is not an image (PNG, JPEG, GIF, TIFF or BMP)."}
}
