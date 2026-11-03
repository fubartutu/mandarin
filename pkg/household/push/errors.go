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

package push

import (
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/web"
)

// Push error codes are 902xx.

// ErrDeviceDoesNotExist is returned when a device id isn't one of the caller's.
type ErrDeviceDoesNotExist struct {
	DeviceID int64
}

func (err ErrDeviceDoesNotExist) Error() string {
	return fmt.Sprintf("Push device does not exist [DeviceID: %d]", err.DeviceID)
}

// ErrCodeDeviceDoesNotExist holds the unique world-error code of this error
const ErrCodeDeviceDoesNotExist = 90201

// HTTPError holds the http error description
func (err ErrDeviceDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeDeviceDoesNotExist, Message: "This device isn't set up for notifications."}
}

// ErrInvalidDevice is returned when a subscription can't be used.
type ErrInvalidDevice struct {
	Reason string
}

func (err ErrInvalidDevice) Error() string {
	return "Invalid push device: " + err.Reason
}

// ErrCodeInvalidDevice holds the unique world-error code of this error
const ErrCodeInvalidDevice = 90202

// HTTPError holds the http error description
func (err ErrInvalidDevice) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeInvalidDevice, Message: err.Reason}
}

// ErrInvalidActionToken is returned when a notification button's token is
// forged, malformed or expired, or its user can no longer log in.
type ErrInvalidActionToken struct {
	Expired bool
}

func (err ErrInvalidActionToken) Error() string {
	if err.Expired {
		return "Push action token expired"
	}
	return "Invalid push action token"
}

// ErrCodeInvalidActionToken holds the unique world-error code of this error
const ErrCodeInvalidActionToken = 90203

// HTTPError holds the http error description
func (err ErrInvalidActionToken) HTTPError() web.HTTPError {
	msg := "This notification's buttons don't work. Open the task in Mandarin instead."
	if err.Expired {
		msg = "This notification is too old for its buttons to work. Open the task in Mandarin instead."
	}
	return web.HTTPError{HTTPCode: http.StatusUnauthorized, Code: ErrCodeInvalidActionToken, Message: msg}
}
