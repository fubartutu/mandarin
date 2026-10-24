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

package menu

import (
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/web"
)

// Menu error codes are 903xx (shopping has 900xx, recipes 901xx, push 902xx).

// ErrMenuEntryDoesNotExist is returned when an entry id is unknown.
type ErrMenuEntryDoesNotExist struct {
	EntryID int64
}

func (err ErrMenuEntryDoesNotExist) Error() string {
	return fmt.Sprintf("Menu entry does not exist [EntryID: %d]", err.EntryID)
}

// ErrCodeMenuEntryDoesNotExist holds the unique world-error code of this error
const ErrCodeMenuEntryDoesNotExist = 90301

// HTTPError holds the http error description
func (err ErrMenuEntryDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusNotFound,
		Code:     ErrCodeMenuEntryDoesNotExist,
		Message:  "This menu entry does not exist.",
	}
}

// ErrInvalidMenuEntry is returned for a bad day, meal, note or range.
type ErrInvalidMenuEntry struct {
	Reason string
}

func (err ErrInvalidMenuEntry) Error() string {
	return "Invalid menu entry: " + err.Reason
}

// ErrCodeInvalidMenuEntry holds the unique world-error code of this error
const ErrCodeInvalidMenuEntry = 90302

// HTTPError holds the http error description
func (err ErrInvalidMenuEntry) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusBadRequest,
		Code:     ErrCodeInvalidMenuEntry,
		Message:  err.Reason,
	}
}
