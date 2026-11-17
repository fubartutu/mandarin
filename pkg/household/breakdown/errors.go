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

package breakdown

import (
	"net/http"

	"code.vikunja.io/api/pkg/web"
)

// Task breakdown error codes are 905xx (90501 is models.ErrInvalidBreakdown).

// ErrAIUnavailable is returned when the scraper container can't be reached.
type ErrAIUnavailable struct{}

func (err ErrAIUnavailable) Error() string {
	return "Breakdown AI is unavailable"
}

// ErrCodeAIUnavailable holds the unique world-error code of this error
const ErrCodeAIUnavailable = 90502

// HTTPError holds the http error description
func (err ErrAIUnavailable) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusServiceUnavailable,
		Code:     ErrCodeAIUnavailable,
		Message:  "The AI helper isn't running (the mandarin-scraper container). You can still type the steps yourself.",
	}
}

// ErrSuggestFailed is returned when the AI couldn't break the task down.
type ErrSuggestFailed struct {
	Reason string
	// Unavailable is set when the AI isn't set up or can't be reached (503).
	Unavailable bool
}

func (err ErrSuggestFailed) Error() string {
	return "Breakdown failed: " + err.Reason
}

// ErrCodeSuggestFailed holds the unique world-error code of this error
const ErrCodeSuggestFailed = 90503

// HTTPError holds the http error description
func (err ErrSuggestFailed) HTTPError() web.HTTPError {
	code := http.StatusUnprocessableEntity
	if err.Unavailable {
		code = http.StatusServiceUnavailable
	}
	return web.HTTPError{HTTPCode: code, Code: ErrCodeSuggestFailed, Message: err.Reason}
}
