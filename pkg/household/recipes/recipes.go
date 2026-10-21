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

// Package recipes is Mandarin's household recipe box: recipes imported from
// the web (through the scraper container) or typed in, with their
// ingredients split into amount and name so they can go on a shopping list.
//
// Like shopping data, recipes belong to the household: every member can
// read and change all of them. The /api/v2 handlers are in
// pkg/routes/api/v2/household_recipes.go.
package recipes

import (
	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web"
)

func init() {
	db.RegisterTables(GetTables())
}

// GetTables returns all structs which are also a table. pkg/migration uses
// this to create them on a fresh install.
func GetTables() []any {
	return []any{
		&Recipe{},
	}
}

// permissionAdmin is what CanRead reports: members may do anything with
// household data, so the frontend shows every edit control.
const permissionAdmin = models.PermissionAdmin

// requireMember: household members may do everything, anyone else nothing.
func requireMember(a web.Auth) error {
	if !household.IsMember(a) {
		return models.ErrGenericForbidden{}
	}
	return nil
}

// limitFromPage mirrors pkg/models' unexported getLimitFromPageIndex: page < 1
// means "everything", otherwise perPage (or the server default) per page.
func limitFromPage(page, perPage int) (limit, start int) {
	if page < 1 {
		return 0, 0
	}
	limit = config.ServiceMaxItemsPerPage.GetInt()
	if perPage > 0 {
		limit = perPage
	}
	return limit, limit * (page - 1)
}
