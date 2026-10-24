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

// Package shopping is Mandarin's household shopping-list module: lists,
// items on them, and the categories items are grouped by.
//
// Everything here belongs to the household, not to a user: any household
// member can see and change every list (see household.IsMember). The models
// plug into Vikunja's generic CRUD machinery (pkg/web) the same way
// pkg/models does, so the /api/v2 handlers in
// pkg/routes/api/v2/household_shopping.go stay thin.
package shopping

import (
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"xorm.io/xorm"
)

// init runs once when the package is loaded. Registering the tables lets
// Vikunja's dump/restore and test helpers know about them.
func init() {
	db.RegisterTables(GetTables())
}

// GetTables returns all structs which are also a table. pkg/migration uses
// this to create them on a fresh install.
func GetTables() []any {
	return []any{
		&ShoppingList{},
		&ShoppingItem{},
		&ShoppingCategory{},
		&ShoppingShop{},
		&ShoppingCatalogItem{},
		&ShoppingNotStocked{},
		&ShoppingChange{},
		&ShoppingConflict{},
		&ShoppingItemAlias{},
	}
}

// permissionAdmin is what CanRead reports: members may do anything with
// household data, so the frontend shows every edit control.
const permissionAdmin = models.PermissionAdmin

// paginate applies household.LimitFromPage to a query. limit 0 means no limit.
func paginate(q *xorm.Session, page, perPage int) *xorm.Session {
	if limit, start := household.LimitFromPage(page, perPage); limit > 0 {
		return q.Limit(limit, start)
	}
	return q
}

// nextPosition returns a position after every existing row of table, so a
// new list or category without an explicit position goes to the end.
func nextPosition(s *xorm.Session, table string) (float64, error) {
	var maxPosition float64
	if _, err := s.Table(table).Select("COALESCE(MAX(position), 0)").Get(&maxPosition); err != nil {
		return 0, err
	}
	return maxPosition + 1, nil
}

// loadCreators fills in the created_by user for each id in ids. Lists seeded
// by the server have creator id 0 and are skipped.
func loadCreators(s *xorm.Session, ids []int64) (map[int64]*user.User, error) {
	wanted := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			wanted = append(wanted, id)
		}
	}
	if len(wanted) == 0 {
		return map[int64]*user.User{}, nil
	}
	return user.GetUsersByIDs(s, wanted)
}
