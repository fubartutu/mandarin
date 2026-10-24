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

// Package menu is Mandarin's household menu: what's planned for each meal of
// each day. There is one plan for the whole household and every member can
// change it. Online-only, like recipes (no sync engine). The /api/v2 handlers
// are in pkg/routes/api/v2/household_menu.go.
package menu

import (
	"code.vikunja.io/api/pkg/db"
)

func init() {
	db.RegisterTables(GetTables())
}

// GetTables returns all structs which are also a table. pkg/migration uses
// this to create them on a fresh install.
func GetTables() []any {
	return []any{
		&MenuEntry{},
	}
}
