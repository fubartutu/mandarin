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

package habits

import (
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/web"
)

// Habit error codes are 905xx (shopping has 900xx, recipes 901xx, push
// 902xx, menu 903xx, task repeats 904xx).

// ErrHabitDoesNotExist is returned when a habit id is unknown, or the habit
// is someone else's personal habit (which callers mustn't learn exists).
type ErrHabitDoesNotExist struct {
	HabitID int64
}

func (err ErrHabitDoesNotExist) Error() string {
	return fmt.Sprintf("Habit does not exist [HabitID: %d]", err.HabitID)
}

// ErrCodeHabitDoesNotExist holds the unique world-error code of this error
const ErrCodeHabitDoesNotExist = 90501

// HTTPError holds the http error description
func (err ErrHabitDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusNotFound,
		Code:     ErrCodeHabitDoesNotExist,
		Message:  "This habit does not exist.",
	}
}

// ErrInvalidHabit is returned for a bad field on a habit, group, tick or skip.
type ErrInvalidHabit struct {
	Reason string
}

func (err ErrInvalidHabit) Error() string {
	return "Invalid habit: " + err.Reason
}

// ErrCodeInvalidHabit holds the unique world-error code of this error
const ErrCodeInvalidHabit = 90502

// HTTPError holds the http error description
func (err ErrInvalidHabit) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusBadRequest,
		Code:     ErrCodeInvalidHabit,
		Message:  err.Reason,
	}
}

// ErrHabitGroupDoesNotExist is returned when a group id is unknown, or the
// group is someone else's personal group.
type ErrHabitGroupDoesNotExist struct {
	GroupID int64
}

func (err ErrHabitGroupDoesNotExist) Error() string {
	return fmt.Sprintf("Habit group does not exist [GroupID: %d]", err.GroupID)
}

// ErrCodeHabitGroupDoesNotExist holds the unique world-error code of this error
const ErrCodeHabitGroupDoesNotExist = 90503

// HTTPError holds the http error description
func (err ErrHabitGroupDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusNotFound,
		Code:     ErrCodeHabitGroupDoesNotExist,
		Message:  "This habit group does not exist.",
	}
}

// ErrOnlyOwnerCanUnshare is returned when someone other than a household
// routine's owner tries to make it personal again: it would vanish for them.
type ErrOnlyOwnerCanUnshare struct{}

func (err ErrOnlyOwnerCanUnshare) Error() string {
	return "Only the owner can make a household habit personal"
}

// ErrCodeOnlyOwnerCanUnshare holds the unique world-error code of this error
const ErrCodeOnlyOwnerCanUnshare = 90504

// HTTPError holds the http error description
func (err ErrOnlyOwnerCanUnshare) HTTPError() web.HTTPError {
	return web.HTTPError{
		HTTPCode: http.StatusForbidden,
		Code:     ErrCodeOnlyOwnerCanUnshare,
		Message:  "Only the person who made this household habit can make it personal again.",
	}
}
