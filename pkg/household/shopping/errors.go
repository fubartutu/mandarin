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
