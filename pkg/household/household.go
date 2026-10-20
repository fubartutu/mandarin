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

// Package household holds Mandarin's single-household rules.
//
// A Mandarin server has exactly one household and every user account belongs
// to it, so there is no household table or picker: "is this caller a member?"
// is the same as "is this caller a real user?". Household data (shopping lists,
// recipes) is shared by all members. Link shares are not members: they only
// ever see the single project they were created for.
package household

import (
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web"
)

// IsMember reports whether the caller belongs to the household. Users, bot
// users and API tokens (which authenticate as their user) are members; link
// shares are not.
func IsMember(a web.Auth) bool {
	if a == nil {
		return false
	}
	// A "type switch" on an interface value asks which concrete type is
	// behind it. Link shares implement web.Auth too, so we exclude them.
	switch a.(type) {
	case *models.LinkSharing:
		return false
	default:
		return a.GetID() > 0
	}
}
